package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/disgord/x/mux"
	"github.com/chremoas/chremoas-ng/internal/sigs"
	"go.uber.org/zap"

	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/roles"
)

const sigHelpStr = `
Usage: !sig <subcommand> <arguments>

Subcommands:
    list: List all SIGs
    list members: List SIG members
    list membership: List user SIGs
    create: Add SIGs
    destroy: Delete SIGs
    info: Get SIG info
    set: Set sig key
    add: Add user to SIG
    remove: Remove user from SIG
    join: Join SIG
    leave: Leave SIG
	filter list: list filters associated with role
	filter add: add filter to role
	filter remove: remove filter from role
`

// Sig will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func (c Command) Sig(s *discordgo.Session, m *discordgo.Message, ctx *mux.Context) {
	logger := c.dependencies.Logger.With(zap.String("command", "sig"))

	_, err := s.ChannelMessageSend(m.ChannelID, c.doSig(m, logger))
	if err != nil {
		logger.Error("Error sending command", zap.Error(err))
	}
}

func (c Command) doSig(m *discordgo.Message, logger *zap.Logger) string {
	logger.Info("Received chat command", zap.String("content", m.Content))

	cmdStr := strings.Split(m.Content, " ")

	if len(cmdStr) < 2 {
		return fmt.Sprintf("```%s```", sigHelpStr)
	}

	switch cmdStr[1] {
	case "list":
		if len(cmdStr) < 3 {
			return roles.List(roles.Sig, false, c.dependencies)
		}

		switch cmdStr[2] {
		case "all":
			return roles.List(roles.Sig, true, c.dependencies)

		case "members":
			if len(cmdStr) < 4 {
				return "Usage: !role list members <role_name>"
			}
			return roles.ListMembers(roles.Sig, cmdStr[3], c.dependencies)

		case "membership":
			if len(cmdStr) < 4 {
				return roles.ListUserRoles(roles.Sig, m.Author.ID, c.dependencies)
			}

			if !common.IsDiscordUser(cmdStr[3]) {
				return common.SendError("member name must be a discord user")
			}

			return roles.ListUserRoles(roles.Sig, common.ExtractUserId(cmdStr[3]), c.dependencies)

		default:
			return fmt.Sprintf("```%s```", sigHelpStr)
		}

	case "create":
		if len(cmdStr) < 5 {
			return "Usage: !sig create <sig_name> <joinable> <sig_description>"
		} else {
			joinable, err := strconv.ParseBool(cmdStr[3])
			if err != nil {
				return common.SendError(fmt.Sprintf("Error parsing joinable `%s` is not a bool value", cmdStr[3]))
			}
			return roles.AuthedAdd(roles.Sig, joinable, cmdStr[2], strings.Join(cmdStr[4:], " "), "discord", m.Author.ID, c.dependencies)
		}

	case "destroy":
		if len(cmdStr) < 3 {
			return "Usage: !sig destroy <sig_name>"
		}
		return roles.AuthedDestroy(roles.Sig, cmdStr[2], m.Author.ID, c.dependencies)

	case "info":
		if len(cmdStr) < 3 {
			return "Usage: !sig info <sig_name>"
		}
		return roles.Info(roles.Sig, cmdStr[2], c.dependencies)

	case "set":
		if len(cmdStr) < 5 {
			return "Usage: !sig set <sig_name> <key> <value>"
		}
		return roles.AuthedUpdate(roles.Sig, cmdStr[2], cmdStr[3], cmdStr[4], m.Author.ID, c.dependencies)

	case "add":
		var (
			sig *sigs.Sig
			err error
		)
		if len(cmdStr) < 4 {
			return "Usage: !sig add <user> <sig>"
		}
		sig, err = sigs.New(cmdStr[2], cmdStr[3], m.Author.ID, c.dependencies)
		if err != nil {
			return common.SendError(err.Error())
		}
		return sig.Add()

	case "remove":
		var (
			sig *sigs.Sig
			err error
		)
		if len(cmdStr) < 4 {
			return "Usage: !sig remove <user> <sig>"
		}
		sig, err = sigs.New(cmdStr[2], cmdStr[3], m.Author.ID, c.dependencies)
		if err != nil {
			return common.SendError(err.Error())
		}
		return sig.Remove()

	case "join":
		var (
			sig *sigs.Sig
			err error
		)
		if len(cmdStr) < 3 {
			return "Usage: !sig join <sig>"
		}
		sig, err = sigs.New(m.Author.ID, cmdStr[2], m.Author.ID, c.dependencies)
		if err != nil {
			return common.SendError(err.Error())
		}
		return sig.Join()

	case "leave":
		var (
			sig *sigs.Sig
			err error
		)
		if len(cmdStr) < 3 {
			return "Usage: !sig leave <sig>"
		}
		sig, err = sigs.New(m.Author.ID, cmdStr[2], m.Author.ID, c.dependencies)
		if err != nil {
			return common.SendError(err.Error())
		}
		return sig.Leave()

	case "keys":
		return roles.Keys()

	case "types":
		return roles.Types()

	case "filter":
		if len(cmdStr) < 3 {
			return "Usage: subcommands are: list, add and remove"
		}

		switch cmdStr[2] {
		case "list":
			if len(cmdStr) < 4 {
				return "Usage: !role filter list <role>"
			}
			return roles.ListFilters(roles.Sig, cmdStr[3], c.dependencies)

		case "add":
			if len(cmdStr) < 5 {
				return "Usage: !role filter add <filter> <role>"
			}
			return roles.AuthedAddFilter(roles.Sig, cmdStr[3], cmdStr[4], m.Author.ID, c.dependencies)

		case "remove":
			if len(cmdStr) < 5 {
				return "Usage: !role filter remove <filter> <role>"
			}
			return roles.AuthedRemoveFilter(roles.Sig, cmdStr[3], cmdStr[4], m.Author.ID, c.dependencies)

		default:
			return "Usage: !role filter list <role>"
		}

	case "help":
		return fmt.Sprintf("```%s```", sigHelpStr)

	default:
		return fmt.Sprintf("```%s```", sigHelpStr)
	}
}
