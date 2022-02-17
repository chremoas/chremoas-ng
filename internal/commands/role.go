package commands

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/disgord/x/mux"
	"go.uber.org/zap"

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
`

// Role will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func (c Command) Role(s *discordgo.Session, m *discordgo.Message, ctx *mux.Context) {
	logger := c.dependencies.Logger.With(zap.String("command", "role"))

	_, err := s.ChannelMessageSend(m.ChannelID, c.doRole(m, logger))
	if err != nil {
		logger.Error("Error sending command", zap.Error(err))
	}
}

func (c Command) doRole(m *discordgo.Message, logger *zap.Logger) string {
	logger.Info("Received chat command", zap.String("content", m.Content))

	cmdStr := strings.Split(m.Content, " ")

	if len(cmdStr) < 2 {
		return fmt.Sprintf("```%s```", roleHelpStr)
	}

	switch cmdStr[1] {
	// case "list":
	// 	if len(cmdStr) < 3 {
	// 		return roles.List(roles.Role, false, c.dependencies)
	// 	}
	//
	// 	switch cmdStr[2] {
	// 	case "all":
	// 		return roles.List(roles.Role, true, c.dependencies)
	//
	// 	case "members":
	// 		if len(cmdStr) < 4 {
	// 			return "Usage: !role list members <role_name>"
	// 		}
	// 		return roles.ListMembers(roles.Role, cmdStr[3], c.dependencies)
	//
	// 	case "membership":
	// 		if len(cmdStr) < 4 {
	// 			return roles.ListUserRoles(roles.Role, m.Author.ID, c.dependencies)
	// 		}
	//
	// 		if !common.IsDiscordUser(cmdStr[3]) {
	// 			return common.SendError("member name must be a discord user")
	// 		}
	//
	// 		return roles.ListUserRoles(roles.Role, common.ExtractUserId(cmdStr[3]), c.dependencies)
	//
	// 	default:
	// 		return "Usage: !role list members <role_name>"
	// 	}

	case "create":
		if len(cmdStr) < 4 {
			return "Usage: !role create <role_name> <role_description>"
		}
		return roles.AuthedAdd(roles.Role, false, cmdStr[2], strings.Join(cmdStr[3:], " "), "discord", m.Author.ID, c.dependencies)

	case "destroy":
		if len(cmdStr) < 3 {
			return "Usage: !role destroy <role_name>"
		}
		return roles.AuthedDestroy(roles.Role, cmdStr[2], m.Author.ID, c.dependencies)

	case "set":
		if len(cmdStr) < 5 {
			return "Usage: !role set <role_name> <key> <value>"
		}
		return roles.AuthedUpdate(roles.Role, cmdStr[2], cmdStr[3], cmdStr[4], m.Author.ID, c.dependencies)

	case "info":
		if len(cmdStr) < 3 {
			return "Usage: !role info <role_name>"
		}
		return roles.Info(roles.Role, cmdStr[2], c.dependencies)

	case "keys":
		return roles.Keys()

	case "types":
		return roles.Types()

	case "help":
		return fmt.Sprintf("```%s```", roleHelpStr)

	default:
		return fmt.Sprintf("```%s```", roleHelpStr)
	}
}
