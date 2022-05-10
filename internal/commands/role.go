package commands

import (
	"context"
	"strings"

	sl "github.com/bhechinger/spiffylogger"
	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/disgord/x/mux"
	"github.com/chremoas/chremoas-ng/internal/common"
	"go.uber.org/zap"

	"github.com/chremoas/chremoas-ng/internal/roles"
)

const (
	roleUsage       = `!role <subcommand> <parameters>`
	roleSubcommands = `
    list: List all Roles that are set to sync
    create: Add Role
    destroy: Delete role
    info: Get Role Info
    keys: Get valid role keys
    types: Get valid role types
    set: Set role key
    list members: List Role members
    list membership: List user Roles
`
)

// Role will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func (c Command) Role(s *discordgo.Session, m *discordgo.Message, _ *mux.Context) {
	ctx, sp := sl.OpenCorrelatedSpan(c.ctx, sl.NewID())
	defer sp.Close()

	sp.With(zap.String("command", "role"))

	for _, message := range c.doRole(ctx, m) {
		_, err := s.ChannelMessageSendComplex(m.ChannelID, message)

		if err != nil {
			sp.Error("Error sending command", zap.Error(err))
		}
	}
}

func (c Command) doRole(ctx context.Context, m *discordgo.Message) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.Info("Received chat command", zap.String("content", m.Content))

	cmdStr := strings.Split(m.Content, " ")

	if len(cmdStr) < 2 {
		return getHelp("!role help", roleUsage, roleSubcommands)
	}

	switch cmdStr[1] {
	case "list":
		if len(cmdStr) < 3 {
			return roles.List(ctx, roles.Role, false, m.ChannelID, c.dependencies)
		}

		switch cmdStr[2] {
		// case "all":
		// 	return roles.List(ctx, roles.Role, true, m.ChannelID, c.dependencies)

		case "members":
			if len(cmdStr) < 4 {
				return getHelp("!role list members help", "!role list members <role_name>", "")
			}
			return roles.ListMembers(ctx, roles.Role, cmdStr[3], c.dependencies)

		case "membership":
			if len(cmdStr) < 4 {
				return roles.ListUserRoles(ctx, roles.Role, m.Author.ID, c.dependencies)
			}

			if !common.IsDiscordUser(cmdStr[3]) {
				return common.SendError("member name must be a discord user")
			}

			return roles.ListUserRoles(ctx, roles.Role, common.ExtractUserId(cmdStr[3]), c.dependencies)

		default:
			return getHelp("!role list help", roleUsage, roleSubcommands)
		}

	case "create":
		if len(cmdStr) < 4 {
			return getHelp("!role create help", "!role create <role_name> <role_description>", "")
		}
		return roles.AuthedAdd(ctx, roles.Role, false, cmdStr[2], strings.Join(cmdStr[3:], " "), "discord", m.Author.ID, c.dependencies)

	case "destroy":
		if len(cmdStr) < 3 {
			return getHelp("!role destroy help", "!role destroy <role_name>", "")
		}
		return roles.AuthedDestroy(ctx, roles.Role, cmdStr[2], m.Author.ID, c.dependencies)

	case "set":
		if len(cmdStr) < 5 {
			return getHelp("!role set help", "!role set <role_name> <key> <value>", "")
		}
		return roles.AuthedUpdate(ctx, roles.Role, cmdStr[2], cmdStr[3], cmdStr[4], m.Author.ID, c.dependencies)

	case "info":
		if len(cmdStr) < 3 {
			return getHelp("!role info help", "!role info <role_name>", "")
		}
		return roles.Info(ctx, roles.Role, cmdStr[2], c.dependencies)

	case "keys":
		return roles.Keys()

	case "types":
		return roles.Types()
	}

	return getHelp("!role help", roleUsage, roleSubcommands)
}
