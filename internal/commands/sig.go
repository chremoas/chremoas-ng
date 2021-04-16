package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/disgord/x/mux"
	"github.com/chremoas/chremoas-ng/internal/sigs"

	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/roles"
)

const sigHelpStr = `
Usage: !sig <subcommand> <arguments>

Subcommands:
    list: List all SIGs
    create: Add SIGs
    destroy: Delete SIGs
    info: Get SIG info
    set: Set sig key
    add: Add user to SIG
    remove: Remove user from SIG
    join: Join SIG
    leave: Leave SIG
    list_members: List SIG members
    list_sigs: List user SIGs
`

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func (c Command) Sig(s *discordgo.Session, m *discordgo.Message, ctx *mux.Context) {
	_, err := s.ChannelMessageSend(m.ChannelID, c.doSig(s, m, ctx))
	if err != nil {
		c.logger.Errorf("Error sending command: %s", err)
	}
}

func (c Command) doSig(s *discordgo.Session, m *discordgo.Message, ctx *mux.Context) string {
	c.logger.Infof("Recieved: %s", m.Content)
	cmdStr := strings.Split(m.Content, " ")

	if len(cmdStr) < 2 {
		return fmt.Sprintf("```%s```", sigHelpStr)
	}

	switch cmdStr[1] {
	case "list":
		var all bool

		if len(cmdStr) > 2 && cmdStr[2] == "all" {
			all = true
		}
		return roles.List(roles.Sig, all, c.logger, c.db)

	case "create":
		if len(cmdStr) < 5 {
			return "Usage: !sig create <sig_name> <joinable> <sig_description>"
		} else {
			joinable, err := strconv.ParseBool(cmdStr[3])
			if err != nil {
				return common.SendError(fmt.Sprintf("Error parsing joinable `%s` is not a bool value", cmdStr[3]))
			}
			return roles.Add(roles.Sig, joinable, cmdStr[2], strings.Join(cmdStr[4:], " "), "discord", m.Author.ID, c.logger, c.db, c.nsq)
		}

	case "destroy":
		if len(cmdStr) < 3 {
			return "Usage: !sig destroy <sig_name>"
		}
		return roles.Destroy(roles.Sig, cmdStr[2], m.Author.ID, c.logger, c.db, c.nsq)

	case "info":
		if len(cmdStr) < 3 {
			return "Usage: !sig info <sig_name>"
		}
		return roles.Info(roles.Sig, cmdStr[2], c.logger, c.db)

	case "set":
		if len(cmdStr) < 5 {
			return "Usage: !sig set <sig_name> <key> <value>"
		}
		return roles.Update(roles.Sig, cmdStr[2], cmdStr[3], cmdStr[4], m.Author.ID, c.logger, c.db, c.nsq)

	case "add":
		var (
			sig *sigs.Sig
			err error
		)
		if len(cmdStr) < 4 {
			return "Usage: !sig add <user> <sig>"
		}
		sig, err = sigs.New(cmdStr[2], cmdStr[3], m.Author.ID, c.logger, c.db, c.nsq)
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
		sig, err = sigs.New(cmdStr[2], cmdStr[3], m.Author.ID, c.logger, c.db, c.nsq)
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
		sig, err = sigs.New(m.Author.ID, cmdStr[2], m.Author.ID, c.logger, c.db, c.nsq)
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
		sig, err = sigs.New(m.Author.ID, cmdStr[2], m.Author.ID, c.logger, c.db, c.nsq)
		if err != nil {
			return common.SendError(err.Error())
		}
		return sig.Leave()

	case "keys":
		return roles.Keys()

	case "types":
		return roles.Types()

	case "list_members":
		if len(cmdStr) < 3 {
			return "Usage: !sig list_members <sig_name>"
		}
		return roles.Members(roles.Sig, cmdStr[2], c.logger, c.db)

	case "list_sigs":
		if len(cmdStr) < 3 {
			return roles.ListUserRoles(roles.Sig, m.Author.ID, c.logger, c.db)
		}
		return roles.ListUserRoles(roles.Sig, common.ExtractUserId(cmdStr[2]), c.logger, c.db)

	case "filter":
		if len(cmdStr) < 3 {
			return "Usage: subcommands are: list, add and remove"
		}

		switch cmdStr[2] {
		case "list":
			if len(cmdStr) < 4 {
				return "Usage: !role filter list <role>"
			}
			return roles.ListFilters(roles.Sig, cmdStr[3], c.logger, c.db)

		case "add":
			if len(cmdStr) < 5 {
				return "Usage: !role filter add <filter> <role>"
			}
			return roles.AddFilter(roles.Sig, cmdStr[3], cmdStr[4], m.Author.ID, c.logger, c.db, c.nsq)

		case "remove":
			if len(cmdStr) < 5 {
				return "Usage: !role filter remove <filter> <role>"
			}
			return roles.RemoveFilter(roles.Sig, cmdStr[3], cmdStr[4], m.Author.ID, c.logger, c.db, c.nsq)
		}

	case "help":
		return fmt.Sprintf("```%s```", sigHelpStr)

	default:
		return fmt.Sprintf("```%s```", sigHelpStr)
	}

	return "Something I don't know"
}
