package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/disgord/x/mux"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/filters"
)

const filterHelpStr = `
Usage: !filter <subcommand> <arguments>

Subcommands:
    list: List all Filters
    create: Add Filter
    destroy: Delete Filter
    add: Add Filter Member
    remove: Remove Filter Member
    list_members: List all Filter Members
`

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func (c Command) Filter(s *discordgo.Session, m *discordgo.Message, ctx *mux.Context) {
	var (
		rs string
	)

	c.logger.Infof("Recieved: %s", m.Content)
	cmdStr := strings.Split(m.Content, " ")

	if len(cmdStr) < 2 {
		rs = fmt.Sprintf("```%s```", filterHelpStr)
		goto sendMessage
	}

	switch cmdStr[1] {
	case "list":
		rs = filters.List(c.logger, c.db)

	case "create":
		if len(cmdStr) < 5 {
			rs = "Usage: !role create <role_name> <sig> <role_description>"
		} else {
			sig, err := strconv.ParseBool(cmdStr[3])
			if err != nil {
				rs = common.SendError(fmt.Sprintf("Error parsing sig `%s` is not a bool value", cmdStr[3]))
			} else {
				rs = filters.Add(cmdStr[2], strings.Join(cmdStr[4:], " "), sig, c.logger, c.db)
			}
		}

	case "destroy":
		if len(cmdStr) < 4 {
			rs = "Usage: !role destroy <role_name> <sig>"
		} else {
			sig, err := strconv.ParseBool(cmdStr[3])
			if err != nil {
				rs = common.SendError(fmt.Sprintf("Error parsing sig `%s` is not a bool value", cmdStr[3]))
			} else {
				rs = filters.Delete(cmdStr[2], sig, c.logger, c.db)
			}
		}

	case "add":
		if len(cmdStr) < 4 {
			rs = "Usage: !filter add <user> <filter_name>"
		} else {
			rs = filters.AddMember(cmdStr[2], cmdStr[3], c.logger, c.db)
		}

	case "remove":
		if len(cmdStr) < 4 {
			rs = "Usage: !filter remove <user> <filter_name>"
		} else {
			rs = filters.RemoveMember(cmdStr[2], cmdStr[3], c.logger, c.db)
		}

	case "list_members":
		if len(cmdStr) < 3 {
			rs = "Usage: !role list_members <role_name>"
		} else {
			rs = filters.Members(cmdStr[2], c.logger, c.db)
		}

	case "help":
		rs = fmt.Sprintf("```%s```", filterHelpStr)

	default:
		rs = fmt.Sprintf("```%s```", filterHelpStr)
	}

sendMessage:
	_, err := s.ChannelMessageSend(m.ChannelID, rs)
	if err != nil {
		c.logger.Errorf("Error sending command: %s", err)
	}
}
