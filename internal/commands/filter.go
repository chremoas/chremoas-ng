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
	_, err := s.ChannelMessageSend(m.ChannelID, c.doFilter(s, m, ctx))
	if err != nil {
		c.logger.Errorf("Error sending command: %s", err)
	}
}

func (c Command) doFilter(s *discordgo.Session, m *discordgo.Message, ctx *mux.Context) string {
	c.logger.Infof("Recieved: %s", m.Content)
	cmdStr := strings.Split(m.Content, " ")

	if len(cmdStr) < 2 {
		return fmt.Sprintf("```%s```", filterHelpStr)
	}

	switch cmdStr[1] {
	case "list":
		return filters.List(c.logger, c.db)

	case "create":
		if len(cmdStr) < 5 {
			return "Usage: !role create <role_name> <sig> <role_description>"
		}
		sig, err := strconv.ParseBool(cmdStr[3])
		if err != nil {
			return common.SendError(fmt.Sprintf("Error parsing sig `%s` is not a bool value", cmdStr[3]))
		}
		f, _ := filters.Add(cmdStr[2], strings.Join(cmdStr[4:], " "), sig, m.Author.ID, c.logger, c.db)
		return f

	case "destroy":
		if len(cmdStr) < 4 {
			return "Usage: !role destroy <role_name> <sig>"
		}
		sig, err := strconv.ParseBool(cmdStr[3])
		if err != nil {
			return common.SendError(fmt.Sprintf("Error parsing sig `%s` is not a bool value", cmdStr[3]))
		}
		f, _ := filters.Delete(cmdStr[2], sig, m.Author.ID, c.logger, c.db)
		return f

	case "add":
		if len(cmdStr) < 5 {
			return "Usage: !filter add <user> <filter_name> <sig>"
		}
		sig, err := strconv.ParseBool(cmdStr[4])
		if err != nil {
			return common.SendError(fmt.Sprintf("Error parsing sig `%s` is not a bool value", cmdStr[4]))
		}
		return filters.AddMember(sig, cmdStr[2], cmdStr[3], m.Author.ID, c.logger, c.db, c.nsq)

	case "remove":
		if len(cmdStr) < 5 {
			return "Usage: !filter remove <user> <filter_name> <sig>"
		}
		sig, err := strconv.ParseBool(cmdStr[4])
		if err != nil {
			return common.SendError(fmt.Sprintf("Error parsing sig `%s` is not a bool value", cmdStr[3]))
		}
		return filters.RemoveMember(sig, cmdStr[2], cmdStr[3], m.Author.ID, c.logger, c.db, c.nsq)

	case "list_members":
		if len(cmdStr) < 3 {
			return "Usage: !role list_members <role_name>"
		}
		return filters.Members(cmdStr[2], c.logger, c.db)

	case "help":
		return fmt.Sprintf("```%s```", filterHelpStr)

	default:
		return fmt.Sprintf("```%s```", filterHelpStr)
	}
}
