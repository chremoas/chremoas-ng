package commands

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	sl "github.com/bhechinger/spiffylogger"
	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/disgord/x/mux"
	"github.com/chremoas/chremoas-ng/internal/sigs"
	"go.uber.org/zap"

	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/roles"
)

const (
	sigUsage       = `!sig <subcommand> <parameters>`
	sigSubcommands = `
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
	filter list: List filters associated with sig
	filter add: Add filter to sig
	filter remove: Remove filter from sig
`
)

// Sig will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func (c Command) Sig(s *discordgo.Session, m *discordgo.Message, _ *mux.Context) {
	ctx, sp := sl.OpenCorrelatedSpan(c.ctx, sl.NewID())
	defer sp.Close()

	sp.With(zap.String("command", "sig"))

	for _, message := range c.doSig(ctx, m) {
		_, err := s.ChannelMessageSendComplex(m.ChannelID, message)

		if err != nil {
			sp.Error("Error sending command", zap.Error(err))
		}
	}
}

func (c Command) doSig(ctx context.Context, m *discordgo.Message) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.Info("Received chat command", zap.String("content", m.Content))

	cmdStr := strings.Split(m.Content, " ")

	if len(cmdStr) < 2 {
		return getHelp("!sig help", sigUsage, sigSubcommands)
	}

	switch cmdStr[1] {
	case "list":
		if len(cmdStr) < 3 {
			return roles.List(ctx, roles.Sig, false, c.dependencies)
		}

		switch cmdStr[2] {
		case "all":
			return roles.List(ctx, roles.Sig, true, c.dependencies)

		case "members":
			if len(cmdStr) < 4 {
				return getHelp("!sig list members help", "!sig list members <sig_name>", "")
			}
			return roles.ListMembers(ctx, roles.Sig, cmdStr[3], c.dependencies)

		case "membership":
			if len(cmdStr) < 4 {
				return roles.ListUserRoles(ctx, roles.Sig, m.Author.ID, c.dependencies)
			}

			if !common.IsDiscordUser(cmdStr[3]) {
				return common.SendError("member name must be a discord user")
			}

			return roles.ListUserRoles(ctx, roles.Sig, common.ExtractUserId(cmdStr[3]), c.dependencies)

		default:
			return getHelp("!sig list help", "!sig list <sub-command>", sigSubcommands)
		}

	case "create":
		if len(cmdStr) < 5 {
			return getHelp("!sig create help", "!sig create <sig_name> <joinable> <sig_description>", "")
		} else {
			joinable, err := strconv.ParseBool(cmdStr[3])
			if err != nil {
				return common.SendError(fmt.Sprintf("Error parsing joinable `%s` is not a bool value", cmdStr[3]))
			}
			return roles.AuthedAdd(ctx, roles.Sig, joinable, cmdStr[2], strings.Join(cmdStr[4:], " "), "discord", m.Author.ID, c.dependencies)
		}

	case "destroy":
		if len(cmdStr) < 3 {
			return getHelp("!sig destroy help", "!sig destroy <sig_name>", "")
		}
		return roles.AuthedDestroy(ctx, roles.Sig, cmdStr[2], m.Author.ID, c.dependencies)

	case "info":
		if len(cmdStr) < 3 {
			return getHelp("!sig info help", "!sig info <sig_name>", "")
		}
		return roles.Info(ctx, roles.Sig, cmdStr[2], c.dependencies)

	case "set":
		if len(cmdStr) < 5 {
			return getHelp("!sig set help", "!sig set <sig_name> <key> <value>", "")
		}
		return roles.AuthedUpdate(ctx, roles.Sig, cmdStr[2], cmdStr[3], cmdStr[4], m.Author.ID, c.dependencies)

	case "add":
		var (
			sig *sigs.Sig
			err error
		)
		if len(cmdStr) < 4 {
			return getHelp("!sig add help", "!sig add <user> <sig>", "")
		}
		sig, err = sigs.New(ctx, cmdStr[2], cmdStr[3], m.Author.ID, c.dependencies)
		if err != nil {
			return common.SendError(err.Error())
		}
		return sig.Add(ctx)

	case "remove":
		var (
			sig *sigs.Sig
			err error
		)
		if len(cmdStr) < 4 {
			return getHelp("!sig remove help", "!sig remove <user> <sig>", "")
		}
		sig, err = sigs.New(ctx, cmdStr[2], cmdStr[3], m.Author.ID, c.dependencies)
		if err != nil {
			return common.SendError(err.Error())
		}
		return sig.Remove(ctx)

	case "join":
		var (
			sig *sigs.Sig
			err error
		)
		if len(cmdStr) < 3 {
			return getHelp("!sig join help", "!sig join <sig>", "")
		}
		sig, err = sigs.New(ctx, m.Author.ID, cmdStr[2], m.Author.ID, c.dependencies)
		if err != nil {
			return common.SendError(err.Error())
		}
		return sig.Join(ctx)

	case "leave":
		var (
			sig *sigs.Sig
			err error
		)
		if len(cmdStr) < 3 {
			return getHelp("!sig leave help", "!sig leave <sig>", "")
		}
		sig, err = sigs.New(ctx, m.Author.ID, cmdStr[2], m.Author.ID, c.dependencies)
		if err != nil {
			return common.SendError(err.Error())
		}
		return sig.Leave(ctx)

	case "keys":
		return roles.Keys()

	case "types":
		return roles.Types()

	case "filter":
		if len(cmdStr) < 3 {
			return getHelp("!sig filter help", "!sig filter", sigSubcommands)
		}

		switch cmdStr[2] {
		case "list":
			if len(cmdStr) < 4 {
				return getHelp("!sig filter list help", "!sig filter list <role>", "")
			}
			return roles.ListFilters(ctx, roles.Sig, cmdStr[3], c.dependencies)

		case "add":
			if len(cmdStr) < 5 {
				return getHelp("!sig filter add help", "!sig filter add <filter> <role>", "")
			}
			return roles.AuthedAddFilter(ctx, roles.Sig, cmdStr[3], cmdStr[4], m.Author.ID, c.dependencies)

		case "remove":
			if len(cmdStr) < 5 {
				return getHelp("!sig filter remove help", "!sig filter remove <filter> <role>", "")
			}
			return roles.AuthedRemoveFilter(ctx, roles.Sig, cmdStr[3], cmdStr[4], m.Author.ID, c.dependencies)

		default:
			return getHelp("!sig filter list help", "!sig filter list <role>", "")
		}
	}

	return getHelp("!sig help", sigUsage, sigSubcommands)
}
