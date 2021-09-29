package esi_poller

import (
	"encoding/json"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/bwmarrin/discordgo"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/payloads"
)

// Make sure the roles we have in the db match what's in discord
func (aep authEsiPoller) syncRoles() (int, int, error) {
	var (
		count         int
		errorCount    int
		roles         []*discordgo.Role
		discordRoles  = make(map[string]*discordgo.Role)
		chremoasRoles = make(map[string]*discordgo.Role)
	)

	roles, err := aep.dependencies.Session.GuildRoles(aep.dependencies.GuildID)
	if err != nil {
		return -1, -1, fmt.Errorf("Error getting discord roles: %s", err)
	}

	for _, role := range roles {
		if !common.IgnoreRole(role.Name) {
			discordRoles[role.Name] = role
		}
	}

	rows, err := aep.dependencies.DB.Select("chat_id", "name", "managed", "mentionable", "hoist", "color", "position", "permissions").
		From("roles").
		Where(sq.Eq{"sync": "true"}).
		Query()
	if err != nil {
		aep.dependencies.Logger.Error(err)
		return -1, -1, fmt.Errorf("error getting role list from db: %w", err)
	}

	defer func() {
		if err = rows.Close(); err != nil {
			aep.dependencies.Logger.Error(err)
		}
	}()

	for rows.Next() {
		var role discordgo.Role
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
			aep.dependencies.Logger.Errorf("error scanning role fields: %s", err)
			errorCount += 1
			continue
		}

		chremoasRoles[role.Name] = &role
	}

	// Delete roles from discord that aren't in the bot
	rolesToDelete := difference(discordRoles, chremoasRoles)
	for _, role := range rolesToDelete {
		err := aep.queueUpdate(role, payloads.Delete)
		if err != nil {
			aep.dependencies.Logger.Errorf("error updating role: %s", err)
			errorCount += 1
			continue
		}

		count += 1
	}

	// Add roles to discord that are in the bot
	rolesToAdd := difference(chremoasRoles, discordRoles)
	for _, role := range rolesToAdd {
		err := aep.queueUpdate(role, payloads.Upsert)
		if err != nil {
			aep.dependencies.Logger.Errorf("error updating role: %s", err)
			errorCount += 1
			continue
		}

		count += 1
	}

	rolesToUpdate := interDiff(chremoasRoles, discordRoles, aep.dependencies)
	for _, role := range rolesToUpdate {
		err := aep.queueUpdate(role, payloads.Upsert)
		if err != nil {
			aep.dependencies.Logger.Errorf("error updating role: %s", err)
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
func interDiff(chremoasMap, discordMap map[string]*discordgo.Role, deps common.Dependencies) []*discordgo.Role {
	var (
		roleList []string
		output   []*discordgo.Role
	)

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
			_, err := deps.DB.Update("roles").
				Set("ID", discordMap[r].ID).
				Where(sq.Eq{"name": r}).
				Query()
			if err != nil {
				deps.Logger.Errorf("error updating role's ID: %s", err)
			}
		}

		if chremoasMap[r].Mentionable != discordMap[r].Mentionable ||
			chremoasMap[r].Hoist != discordMap[r].Hoist ||
			chremoasMap[r].Color != discordMap[r].Color ||
			chremoasMap[r].Permissions != discordMap[r].Permissions {
			deps.Logger.Infof("roles differ for %s", r)

			output = append(output, chremoasMap[r])
		}
	}

	return output
}

func difference(map1, map2 map[string]*discordgo.Role) []*discordgo.Role {
	var found bool
	var output []*discordgo.Role

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

func (aep authEsiPoller) queueUpdate(role *discordgo.Role, action payloads.Action) error {
	payload := payloads.Payload{
		Action: action,
		Role:   role,
	}

	b, err := json.Marshal(payload)
	if err != nil {
		aep.dependencies.Logger.Errorf("error marshalling json for queue: %s", err)
		return err
	}

	aep.dependencies.Logger.Debugf("Submitting role `%s` queue message", role.Name)
	err = aep.dependencies.RolesProducer.Publish(b)
	if err != nil {
		aep.dependencies.Logger.Errorf("error publishing message: %s", err)
		return err
	}

	return nil
}
