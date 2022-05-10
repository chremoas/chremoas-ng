package commands

import (
	"context"
	"strings"

	sl "github.com/bhechinger/spiffylogger"
	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/disgord/x/mux"
	"github.com/chremoas/chremoas-ng/internal/perms"
	"go.uber.org/zap"
)

const (
	permsUsage       = `!perms <subcommand> <arguments>`
	permsSubcommands = `
    list: List all Permissions
    create: Add Permission
    destroy: Delete Permission
    add: Add user to permission group
    remove: Remove user from permission group
    list users: List users in a permission group
    list perms: List all the permissions a user has
`
)

// Perms will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func (c Command) Perms(s *discordgo.Session, m *discordgo.Message, _ *mux.Context) {
	ctx, sp := sl.OpenCorrelatedSpan(c.ctx, sl.NewID())
	defer sp.Close()

	sp.With(zap.String("command", "perms"))

	for _, message := range c.doPerms(ctx, m) {
		_, err := s.ChannelMessageSendComplex(m.ChannelID, message)

		if err != nil {
			sp.Error("Error sending command", zap.Error(err))
		}
	}
}

func (c Command) doPerms(ctx context.Context, m *discordgo.Message) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.Info("Received chat command", zap.String("content", m.Content))

	cmdStr := strings.Split(m.Content, " ")

	if len(cmdStr) < 2 {
		return getHelp("!perms help", permsUsage, permsSubcommands)
	}

	switch cmdStr[1] {
	case "list":
		if len(cmdStr) < 3 {
			return perms.List(ctx, m.ChannelID, c.dependencies)
		}

		switch cmdStr[2] {
		case "users":
			if len(cmdStr) < 4 {
				return getHelp("!perms list help", "!perms list users <permissions>", "")
			}
			return perms.ListMembers(ctx, cmdStr[3], c.dependencies)

		case "perms":
			if len(cmdStr) < 4 {
				return getHelp("!perms list help", "!perms list perms <user>", "")
			}
			return perms.UserPerms(ctx, cmdStr[3], c.dependencies)

		default:
			return getHelp("!perms list help", "!perms list <sub-command>", permsSubcommands)
		}

	case "create":
		if len(cmdStr) < 4 {
			return getHelp("!perms create help", "!perms create <permission_name> <permission_description>", "")
		}
		return perms.Add(ctx, cmdStr[2], cmdStr[3], m.Author.ID, c.dependencies)

	case "destroy":
		if len(cmdStr) < 3 {
			return getHelp("!perms destroy help", "!perms destroy <permission_name>", "")
		}
		return perms.Delete(ctx, cmdStr[2], m.Author.ID, c.dependencies)

	case "add":
		if len(cmdStr) < 4 {
			return getHelp("!perms add help", "!perms add <user> <permissions>", "")
		}
		return perms.AddMember(ctx, cmdStr[2], cmdStr[3], m.Author.ID, c.dependencies)

	case "remove":
		if len(cmdStr) < 4 {
			return getHelp("!perms remove help", "!perms remove <user> <permissions>", "")
		}
		return perms.RemoveMember(ctx, cmdStr[2], cmdStr[3], m.Author.ID, c.dependencies)
	}

	return getHelp("!perms help", permsUsage, permsSubcommands)
}
