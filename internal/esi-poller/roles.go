package esi_poller

import (
	"context"
	"encoding/json"

	sl "github.com/bhechinger/spiffylogger"
	"github.com/bwmarrin/discordgo"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	"go.uber.org/zap"
)

// Make sure the roles we have in the db match what's in discord
func (aep authEsiPoller) syncRoles(ctx context.Context) (int, int, error) {
	ctx, sp := sl.OpenCorrelatedSpan(ctx, sl.NewID())
	defer sp.Close()

	sp.With(zap.String("sub-component", "roles"))

	var (
		count         int
		errorCount    int
		roles         []*discordgo.Role
		discordRoles  = make(map[string]payloads.Role)
		chremoasRoles = make(map[string]payloads.Role)
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	roles, err := aep.dependencies.Session.GuildRoles(aep.dependencies.GuildID)
	if err != nil {
		sp.Error("error getting discord roles", zap.Error(err))
		return -1, -1, err
	}

	for _, role := range roles {
		if !common.IgnoreRole(role.Name) {
			var dr payloads.Role

			dr.ID = role.ID
			dr.Name = role.Name
			dr.Managed = role.Managed
			dr.Mentionable = role.Mentionable
			dr.Hoist = role.Hoist
			dr.Color = role.Color
			dr.Position = role.Position
			dr.Permissions = role.Permissions

			discordRoles[role.Name] = dr
		}
	}

	dbRoles, err := aep.dependencies.Storage.GetRolesBySync(ctx, true)
	if err != nil {
		sp.Error("error getting roles by sync", zap.Error(err))
		return -1, -1, err
	}

	for r := range dbRoles {
		// Check if we need to update the role ID in the database
		if val, ok := discordRoles[dbRoles[r].Name]; ok {
			if val.ID != dbRoles[r].ID {
				sp.Info("Discord role ID doesn't match what we have", zap.String("discord_id", val.ID))
				err = aep.dependencies.Storage.UpdateRole(ctx, val.ID, dbRoles[r].Name, "")
				if err != nil {
					sp.Error("error updating role", zap.Error(err))
					return -1, -1, err
				}

				dbRoles[r].ID = val.ID
			}
		}

		chremoasRoles[dbRoles[r].Name] = dbRoles[r]
	}

	sp.Debug("current roles", zap.Any("chremoas", chremoasRoles))
	sp.Debug("current roles", zap.Any("discord", discordRoles))

	// Delete roles from discord that aren't in the bot
	rolesToDelete := difference(discordRoles, chremoasRoles)
	for _, role := range rolesToDelete {
		sp.With(zap.Any("role", role), zap.Any("action", payloads.Delete))
		err = aep.queueUpdate(ctx, role, payloads.Delete)
		if err != nil {
			sp.Error("error updating role", zap.Error(err))
			errorCount += 1
			continue
		}

		count += 1
	}

	// Add roles to discord that are in the bot
	rolesToAdd := difference(chremoasRoles, discordRoles)
	for _, role := range rolesToAdd {
		sp.With(zap.Any("role", role), zap.Any("action", payloads.Upsert))
		err = aep.queueUpdate(ctx, role, payloads.Upsert)
		if err != nil {
			sp.Error("error updating role", zap.Error(err))
			errorCount += 1
			continue
		}

		count += 1
	}

	rolesToUpdate := interDiff(ctx, chremoasRoles, discordRoles, aep.dependencies)
	for _, role := range rolesToUpdate {
		sp.With(zap.Any("role", role), zap.Any("action", payloads.Upsert))
		err = aep.queueUpdate(ctx, role, payloads.Upsert)
		if err != nil {
			sp.Error(
				"error updating role",
				zap.Error(err),
				zap.String("action", "update"),
				zap.String("role", role.Name),
				zap.String("id", role.ID),
			)
			errorCount += 1
			continue
		}

		count += 1
	}

	return count, errorCount, nil
}

// interDiff finds the intersection of the two maps of roles and then checks if there are any
// differences between what we (chremoas) thinks the roles should be and what they actually are
// in discord.
func interDiff(ctx context.Context, chremoasMap, discordMap map[string]payloads.Role, deps common.Dependencies) []payloads.Role {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(zap.String("sub-component", "roles"))

	var (
		roleList []string
		output   []payloads.Role
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// find the intersection and make as list
	for m1 := range chremoasMap {
		for m2 := range discordMap {
			if m1 == m2 {
				roleList = append(roleList, m1)
			}
		}
	}

	for _, r := range roleList {
		if chremoasMap[r].ID != discordMap[r].ID {
			// The role was probably recreated manually.
			err := deps.Storage.UpdateRole(ctx, discordMap[r].ID, r, "")
			if err != nil {
				sp.Error("error updating role", zap.Error(err))
				continue
			}
		}

		if chremoasMap[r].Mentionable != discordMap[r].Mentionable ||
			chremoasMap[r].Hoist != discordMap[r].Hoist ||
			chremoasMap[r].Color != discordMap[r].Color ||
			chremoasMap[r].Permissions != discordMap[r].Permissions {
			sp.Info("roles differ", zap.String("name", r))

			output = append(output, chremoasMap[r])
		}
	}

	return output
}

func difference(map1, map2 map[string]payloads.Role) []payloads.Role {
	var (
		found  bool
		output []payloads.Role
	)

	for m1, role := range map1 {
		found = false

		for m2 := range map2 {
			if m1 == m2 {
				found = true
			}
		}

		if !found {
			output = append(output, role)
		}
	}

	return output
}

func (aep authEsiPoller) queueUpdate(ctx context.Context, role payloads.Role, action payloads.Action) error {
	_, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	payload := payloads.RolePayload{
		Action:  action,
		GuildID: aep.dependencies.GuildID,
		Role:    role,
	}

	sp.With(
		zap.String("sub-component", "roles"),
		zap.Any("role", role),
		zap.Any("action", action),
		zap.Any("payload", payload),
	)

	b, err := json.Marshal(payload)
	if err != nil {
		sp.Error("error marshalling json for queue", zap.Error(err))
		return err
	}

	sp.Debug("Submitting role queue message")
	err = aep.dependencies.RolesProducer.Publish(ctx, b)
	if err != nil {
		sp.Error("error publishing message", zap.Error(err))
		return err
	}

	return nil
}
