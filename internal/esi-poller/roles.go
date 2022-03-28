package esi_poller

import (
	"context"
	"encoding/json"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	sl "github.com/bhechinger/spiffylogger"
	"github.com/bwmarrin/discordgo"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	"go.uber.org/zap"
)

// Make sure the roles we have in the db match what's in discord
func (aep authEsiPoller) syncRoles(ctx context.Context) (int, int, error) {
	ctx, sp := sl.OpenSpan(ctx)
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
		return -1, -1, fmt.Errorf("error getting discord roles: %s", err)
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

	query := aep.dependencies.DB.Select("chat_id", "name", "managed", "mentionable", "hoist", "color", "position", "permissions").
		From("roles").
		Where(sq.Eq{"sync": "true"})

	common.LogSQL(sp, query)

	rows, err := query.QueryContext(ctx)
	if err != nil {
		sp.Error("error selecting role", zap.Error(err))
		return -1, -1, fmt.Errorf("error getting role list from db: %w", err)
	}
	defer func() {
		if err = rows.Close(); err != nil {
			sp.Error("error closing row", zap.Error(err))
		}
	}()

	for rows.Next() {
		var role payloads.Role
		err = rows.Scan(
			&role.ID,
			&role.Name,
			&role.Managed,
			&role.Mentionable,
			&role.Hoist,
			&role.Color,
			&role.Position,
			&role.Permissions,
		)
		if err != nil {
			sp.Error("error scanning role fields", zap.Error(err))
			errorCount += 1
			continue
		}

		// Check if we need to update the role ID in the database
		if val, ok := discordRoles[role.Name]; ok {
			if val.ID != role.ID {
				sp.Info("Discord role ID doesn't match what we have", zap.String("discord id", val.ID), zap.String("chrmoas id", role.ID))
				update := aep.dependencies.DB.Update("roles").
					Set("chat_id", val.ID).
					Where(sq.Eq{"name": role.Name})

				common.LogSQL(sp, update)

				_, err = update.QueryContext(ctx)
				if err != nil {
					sp.Error("Error updating role id in db",
						zap.Error(err), zap.String("role", role.Name))
					continue
				}

				role.ID = val.ID
			}
		}

		chremoasRoles[role.Name] = role
	}

	sp.Debug("current roles", zap.Any("chremoas", chremoasRoles))
	sp.Debug("current roles", zap.Any("discord", discordRoles))

	// Delete roles from discord that aren't in the bot
	rolesToDelete := difference(discordRoles, chremoasRoles)
	for _, role := range rolesToDelete {
		err := aep.queueUpdate(ctx, role, payloads.Delete)
		if err != nil {
			sp.Error("error updating role",
				zap.Error(err), zap.String("action", "delete"),
				zap.String("role", role.Name), zap.String("id", role.ID))
			errorCount += 1
			continue
		}

		count += 1
	}

	// Add roles to discord that are in the bot
	rolesToAdd := difference(chremoasRoles, discordRoles)
	for _, role := range rolesToAdd {
		err := aep.queueUpdate(ctx, role, payloads.Upsert)
		if err != nil {
			sp.Error("error updating role",
				zap.Error(err), zap.String("action", "add"),
				zap.String("role", role.Name), zap.String("id", role.ID))
			errorCount += 1
			continue
		}

		count += 1
	}

	rolesToUpdate := interDiff(ctx, chremoasRoles, discordRoles, aep.dependencies)
	for _, role := range rolesToUpdate {
		err := aep.queueUpdate(ctx, role, payloads.Upsert)
		if err != nil {
			sp.Error("error updating role",
				zap.Error(err), zap.String("action", "update"),
				zap.String("role", role.Name), zap.String("id", role.ID))
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
			func() {
				update := deps.DB.Update("roles").
					Set("chat_id", discordMap[r].ID).
					Where(sq.Eq{"name": r})

				common.LogSQL(sp, update)

				rows, err := update.QueryContext(ctx)
				if err != nil {
					sp.Error("error updating role's chat_id", zap.Error(err), zap.String("chat_id", discordMap[r].ID))
				}
				defer func() {
					err := rows.Close()
					if err != nil {
						sp.Error("error closing database", zap.Error(err))
					}
				}()
			}()
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

	sp.With(zap.String("sub-component", "roles"))

	payload := payloads.RolePayload{
		Action:  action,
		GuildID: aep.dependencies.GuildID,
		Role:    role,
	}

	b, err := json.Marshal(payload)
	if err != nil {
		sp.Error("error marshalling json for queue", zap.Error(err), zap.Any("payload", payload))
		return err
	}

	sp.Debug("Submitting role queue message",
		zap.String("name", role.Name),
		zap.String("id", role.ID),
		zap.Int64("chat_id", role.ChatID),
	)
	err = aep.dependencies.RolesProducer.Publish(ctx, b)
	if err != nil {
		sp.Error("error publishing message", zap.Error(err), zap.String("role", role.Name))
		return err
	}

	return nil
}
