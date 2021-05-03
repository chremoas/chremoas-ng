package commands

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/disgord/x/mux"
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
    list members: List all Filter Members
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
	c.logger.Infof("Received: %s", m.Content)
	cmdStr := strings.Split(m.Content, " ")

	if len(cmdStr) < 2 {
		return fmt.Sprintf("```%s```", filterHelpStr)
	}

	switch cmdStr[1] {
	case "list":
		if len(cmdStr) < 3 {
			return filters.List(c.logger, c.db)
		}

		switch cmdStr[2] {
		case "members":
			if len(cmdStr) < 4 {
				return "Usage: !role list members <role_name>"
			}
			return filters.ListMembers(cmdStr[3], c.logger, c.db)

		default:
			return "Usage: !role list members <role_name>"
		}

	case "create":
		if len(cmdStr) < 4 {
			return "Usage: !filter create <filter_name> <filter_description>"
		}
		f, _ := filters.AuthedAdd(cmdStr[2], strings.Join(cmdStr[3:], " "), m.Author.ID, c.logger, c.db)
		return f

	case "destroy":
		if len(cmdStr) < 3 {
			return "Usage: !role destroy <filter_name>"
		}
		f, _ := filters.AuthedDelete(cmdStr[2], m.Author.ID, c.logger, c.db)
		return f

	case "add":
		if len(cmdStr) < 4 {
			return "Usage: !filter add <user> <filter_name>"
		}
		return filters.AuthedAddMember(cmdStr[2], cmdStr[3], m.Author.ID, c.logger, c.db, c.nsq)

	case "remove":
		if len(cmdStr) < 4 {
			return "Usage: !filter remove <user> <filter_name>"
		}
		return filters.AuthedRemoveMember(cmdStr[2], cmdStr[3], m.Author.ID, c.logger, c.db, c.nsq)

	case "help":
		return fmt.Sprintf("```%s```", filterHelpStr)

	default:
		return fmt.Sprintf("```%s```", filterHelpStr)
	}
}
