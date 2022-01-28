package commands

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/disgord/x/mux"
	"github.com/chremoas/chremoas-ng/internal/perms"
	"go.uber.org/zap"
)

const permsHelpStr = `
Usage: !perms <subcommand> <arguments>

Subcommands:
    list: List all Permissions
    create: Add Permission
    destroy: Delete Permission
    add: Add user to permission group
    remove: Remove user from permission group
    list users: List users in a permission group
    list perms: List all the permissions a user has
`

// Perms will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func (c Command) Perms(s *discordgo.Session, m *discordgo.Message, ctx *mux.Context) {
	logger := c.dependencies.Logger.With(zap.String("command", "perms"))

	_, err := s.ChannelMessageSend(m.ChannelID, c.doPerms(m, logger))
	if err != nil {
		logger.Error("Error sending command", zap.Error(err))
	}
}

func (c Command) doPerms(m *discordgo.Message, logger *zap.Logger) string {
	logger.Info("Received chat command", zap.String("content", m.Content))

	cmdStr := strings.Split(m.Content, " ")

	if len(cmdStr) < 2 {
		return fmt.Sprintf("```%s```", permsHelpStr)
	}

	switch cmdStr[1] {
	case "list":
		if len(cmdStr) < 3 {
			return perms.List(c.dependencies)
		}

		switch cmdStr[2] {
		case "users":
			if len(cmdStr) < 4 {
				return "Usage: !perms list users <permission>"
			}
			return perms.ListMembers(cmdStr[3], c.dependencies)

		case "perms":
			if len(cmdStr) < 4 {
				return "Usage: !perms list user_perms <user>"
			}
			return perms.UserPerms(cmdStr[3], c.dependencies)

		default:
			return "Usage: !perms list user_perms <user>"
		}

	case "create":
		if len(cmdStr) < 4 {
			return "Usage: !perms create <permission_name> <permission_description>"
		}
		return perms.Add(cmdStr[2], cmdStr[3], m.Author.ID, c.dependencies)

	case "destroy":
		if len(cmdStr) < 3 {
			return "Usage: !perms destroy <permission_name>"
		}
		return perms.Delete(cmdStr[2], m.Author.ID, c.dependencies)

	case "add":
		if len(cmdStr) < 4 {
			return "Usage: !perms add <user> <permission>"
		}
		return perms.AddMember(cmdStr[2], cmdStr[3], m.Author.ID, c.dependencies)

	case "remove":
		if len(cmdStr) < 4 {
			return "Usage: !perms remove <user> <permission>"
		}
		return perms.RemoveMember(cmdStr[2], cmdStr[3], m.Author.ID, c.dependencies)

	case "help":
		return fmt.Sprintf("```%s```", permsHelpStr)

	default:
		return fmt.Sprintf("```%s```", permsHelpStr)
	}
}
