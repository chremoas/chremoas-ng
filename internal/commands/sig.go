package commands

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/disgord/x/mux"

	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/roles"
)

const sigHelpStr = `
Usage: !sig <subcommand> <arguments>

Subcommands:
    *list: List all SIGs
    *create: Add SIGs
    *destroy: Delete SIGs
    add: Add user to SIG
    remove: Remove user from SIG
    *info: Get SIG info
    join: Join SIG
    leave: Leave SIG
    *set: Set sig key
    *list_members: List SIG members
    *list_sigs: List user SIGs
`

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func (c Command) Sig(s *discordgo.Session, m *discordgo.Message, ctx *mux.Context) {
	var (
		all bool
		rs  string
	)

	c.logger.Infof("Recieved: %s", m.Content)
	cmdStr := strings.Split(m.Content, " ")

	if len(cmdStr) < 2 {
		rs = fmt.Sprintf("```%s```", sigHelpStr)
		goto sendMessage
	}

	switch cmdStr[1] {
	case "list":
		if len(cmdStr) > 2 && cmdStr[2] == "all" {
			all = true
		}
		rs = roles.List(roles.Sig, all, c.logger, c.db)

	case "create":
		if len(cmdStr) < 4 {
			rs = "Usage: !sig create <sig_name> <sig_description>"
		} else {
			rs = roles.Add(roles.Sig, cmdStr[2], strings.Join(cmdStr[3:], " "), "discord", c.logger, c.db, c.nsq)
		}

	case "destroy":
		if len(cmdStr) < 3 {
			rs = "Usage: !sig destroy <sig_name>"
		} else {
			rs = roles.Destroy(roles.Sig, cmdStr[2], c.logger, c.db, c.nsq)
		}

	case "set":
		if len(cmdStr) < 5 {
			rs = "Usage: !sig set <sig_name> <key> <value>"
		} else {
			rs = roles.Update(roles.Sig, cmdStr[2], cmdStr[3], cmdStr[4], c.logger, c.db, c.nsq)
		}

	case "info":
		if len(cmdStr) < 3 {
			rs = "Usage: !sig info <sig_name>"
		} else {
			rs = roles.Info(roles.Sig, cmdStr[2], c.logger, c.db)
		}

	case "keys":
		rs = roles.Keys()

	case "types":
		rs = roles.Types()

	case "list_members":
		if len(cmdStr) < 3 {
			rs = "Usage: !sig list_members <sig_name>"
		} else {
			rs = roles.Members(roles.Sig, cmdStr[2], c.logger, c.db)
		}

	case "list_sigs":
		if len(cmdStr) < 3 {
			rs = roles.ListUserRoles(roles.Sig, m.Author.ID, c.logger, c.db)
		} else {
			rs = roles.ListUserRoles(roles.Sig, common.ExtractUserId(cmdStr[2]), c.logger, c.db)
		}

	case "help":
		rs = fmt.Sprintf("```%s```", sigHelpStr)

	default:
		rs = fmt.Sprintf("```%s```", sigHelpStr)
	}

sendMessage:
	_, err := s.ChannelMessageSend(m.ChannelID, rs)
	if err != nil {
		c.logger.Errorf("Error sending command: %s", err)
	}
}
