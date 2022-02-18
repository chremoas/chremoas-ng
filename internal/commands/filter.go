package commands

import (
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/disgord/x/mux"
	"github.com/chremoas/chremoas-ng/internal/filters"
	"go.uber.org/zap"
)

const (
	filterUsage       = `!filter <subcommand> <arguments>`
	filterSubcommands = `
    list: List all Filters
    create: Add Filter
    destroy: Delete Filter
    add: Add Filter Member
    remove: Remove Filter Member
    list members: List all Filter Members
`
)

// Filter will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func (c Command) Filter(s *discordgo.Session, m *discordgo.Message, ctx *mux.Context) {
	logger := c.dependencies.Logger.With(zap.String("command", "filter"))

	for _, message := range c.doFilter(m, logger) {
		_, err := s.ChannelMessageSendComplex(m.ChannelID, message)

		// _, err := s.ChannelMessageSend(m.ChannelID, c.doSig(m, logger))
		if err != nil {
			logger.Error("Error sending command", zap.Error(err))
		}
	}
}

func (c Command) doFilter(m *discordgo.Message, logger *zap.Logger) []*discordgo.MessageSend {
	logger.Info("Received chat command", zap.String("content", m.Content))

	cmdStr := strings.Split(m.Content, " ")

	if len(cmdStr) < 2 {
		return getHelp("!filter help", filterUsage, filterSubcommands)
	}

	switch cmdStr[1] {
	case "list":
		if len(cmdStr) < 3 {
			return filters.List(c.dependencies)
		}

		switch cmdStr[2] {
		case "members":
			if len(cmdStr) < 4 {
				return getHelp("!filter list help", "!filter list members <filter_name>", "")
			}
			return filters.ListMembers(cmdStr[3], c.dependencies)

		default:
			return getHelp("!filter list help", "!filter list members <filter_name>", "")
		}

	case "create":
		if len(cmdStr) < 4 {
			return getHelp("!filter create help", "!filter create <filter_name> <filter_description>", "")
		}
		f, _ := filters.AuthedAdd(cmdStr[2], strings.Join(cmdStr[3:], " "), m.Author.ID, c.dependencies)
		return f

	case "destroy":
		if len(cmdStr) < 3 {
			return getHelp("!filter destroy help", "!filter destroy <filter_name>", "")
		}
		f, _ := filters.AuthedDelete(cmdStr[2], m.Author.ID, c.dependencies)
		return f

	case "add":
		if len(cmdStr) < 4 {
			return getHelp("!filter add help", "!filter add <user> <filter_name>", "")
		}
		return filters.AuthedAddMember(cmdStr[2], cmdStr[3], m.Author.ID, c.dependencies)

	case "remove":
		if len(cmdStr) < 4 {
			return getHelp("!filter remove help", "!filter remove <user> <filter_name>", "")
		}
		return filters.AuthedRemoveMember(cmdStr[2], cmdStr[3], m.Author.ID, c.dependencies)
	}

	return getHelp("!filter help", filterUsage, filterSubcommands)
}
