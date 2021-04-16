package commands

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/disgord/x/mux"
	"github.com/chremoas/chremoas-ng/internal/common"

	"github.com/chremoas/chremoas-ng/internal/roles"
)

const roleHelpStr = `
Usage: !role <subcommand> <arguments>

Subcommands:
    list: List all Roles
    create: Add Role
    destroy: Delete role
    info: Get Role Info
    keys: Get valid role keys
    types: Get valid role types
    set: Set role key
    list members: List Role members
    list membership: List user Roles
	filter list: list filters associated with role
	filter add: add filter to role
	filter remove: remove filter from role
`

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func (c Command) Role(s *discordgo.Session, m *discordgo.Message, ctx *mux.Context) {
	_, err := s.ChannelMessageSend(m.ChannelID, c.doRole(s, m, ctx))
	if err != nil {
		c.logger.Errorf("Error sending command: %s", err)
	}
}

func (c Command) doRole(s *discordgo.Session, m *discordgo.Message, ctx *mux.Context) string {
	c.logger.Infof("Recieved: %s", m.Content)
	cmdStr := strings.Split(m.Content, " ")

	if len(cmdStr) < 2 {
		return fmt.Sprintf("```%s```", roleHelpStr)
	}

	switch cmdStr[1] {
	case "list":
		if len(cmdStr) < 3 {
			return roles.List(roles.Role, false, c.logger, c.db)
		}

		switch cmdStr[2] {
		case "all":
			return roles.List(roles.Role, true, c.logger, c.db)

		case "members":
			if len(cmdStr) < 4 {
				return "Usage: !role list members <role_name>"
			}
			return roles.Members(roles.Role, cmdStr[2], c.logger, c.db)

		case "membership":
			if len(cmdStr) < 4 {
				return roles.ListUserRoles(roles.Role, m.Author.ID, c.logger, c.db)
			}
			return roles.ListUserRoles(roles.Role, common.ExtractUserId(cmdStr[2]), c.logger, c.db)
		}

	case "filter":
		if len(cmdStr) < 3 {
			return "Usage: subcommands are: list, add and remove"
		}

		switch cmdStr[2] {
		case "list":
			if len(cmdStr) < 4 {
				return "Usage: !role filter list <role>"
			}
			return roles.ListFilters(roles.Role, cmdStr[3], c.logger, c.db)

		case "add":
			if len(cmdStr) < 5 {
				return "Usage: !role filter add <filter> <role>"
			}
			return roles.AddFilter(roles.Role, cmdStr[3], cmdStr[4], m.Author.ID, c.logger, c.db, c.nsq)

		case "remove":
			if len(cmdStr) < 5 {
				return "Usage: !role filter remove <filter> <role>"
			}
			return roles.RemoveFilter(roles.Role, cmdStr[3], cmdStr[4], m.Author.ID, c.logger, c.db, c.nsq)
		}

	case "create":
		if len(cmdStr) < 4 {
			return "Usage: !role create <role_name> <role_description>"
		}
		return roles.Add(roles.Role, false, cmdStr[2], strings.Join(cmdStr[3:], " "), "discord", m.Author.ID, c.logger, c.db, c.nsq)

	case "destroy":
		if len(cmdStr) < 3 {
			return "Usage: !role destroy <role_name>"
		}
		return roles.Destroy(roles.Role, cmdStr[2], m.Author.ID, c.logger, c.db, c.nsq)

	case "set":
		if len(cmdStr) < 5 {
			return "Usage: !role set <role_name> <key> <value>"
		}
		return roles.Update(roles.Role, cmdStr[2], cmdStr[3], cmdStr[4], m.Author.ID, c.logger, c.db, c.nsq)

	case "info":
		if len(cmdStr) < 3 {
			return "Usage: !role info <role_name>"
		}
		return roles.Info(roles.Role, cmdStr[2], c.logger, c.db)

	case "keys":
		return roles.Keys()

	case "types":
		return roles.Types()

	case "help":
		return fmt.Sprintf("```%s```", roleHelpStr)

	default:
		return fmt.Sprintf("```%s```", roleHelpStr)
	}

	return "Something I don't know"
}
