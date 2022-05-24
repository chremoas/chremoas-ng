package commands

import (
	"context"
	"strings"

	sl "github.com/bhechinger/spiffylogger"
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
func (c Command) Filter(s *discordgo.Session, m *discordgo.Message, _ *mux.Context) {
	ctx, sp := sl.OpenCorrelatedSpan(c.ctx, sl.NewID())
	defer sp.Close()

	sp.With(zap.String("command", "filter"))

	for _, message := range c.doFilter(ctx, m) {
		_, err := s.ChannelMessageSendComplex(m.ChannelID, message)

		if err != nil {
			sp.Error("Error sending command", zap.Error(err))
		}
	}
}

func (c Command) doFilter(ctx context.Context, m *discordgo.Message) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.Info("Received chat command", zap.String("content", m.Content))

	cmdStr := strings.Split(m.Content, " ")

	if len(cmdStr) < 2 {
		return getHelp("!filter help", filterUsage, filterSubcommands)
	}

	switch cmdStr[1] {
	case "list":
		if len(cmdStr) < 3 {
			return filters.List(ctx, m.ChannelID, c.dependencies)
		}

		switch cmdStr[2] {
		case "members":
			if len(cmdStr) < 4 {
				return getHelp("!filter list help", "!filter list members <filter_name>", "")
			}
			return filters.ListMembers(ctx, cmdStr[3], m.ChannelID, c.dependencies)

		default:
			return getHelp("!filter list help", "!filter list members <filter_name>", "")
		}

	case "create":
		if len(cmdStr) < 4 {
			return getHelp("!filter create help", "!filter create <filter_name> <filter_description>", "")
		}
		f, _ := filters.AuthedAdd(ctx, cmdStr[2], strings.Join(cmdStr[3:], " "), m.Author.ID, c.dependencies)
		return f

	case "destroy":
		if len(cmdStr) < 3 {
			return getHelp("!filter destroy help", "!filter destroy <filter_name>", "")
		}
		f := filters.AuthedDelete(ctx, cmdStr[2], m.Author.ID, c.dependencies)
		return f

	case "add":
		if len(cmdStr) < 4 {
			return getHelp("!filter add help", "!filter add <user> <filter_name>", "")
		}
		return filters.AuthedAddMember(ctx, cmdStr[2], cmdStr[3], m.Author.ID, c.dependencies)

	case "remove":
		if len(cmdStr) < 4 {
			return getHelp("!filter remove help", "!filter remove <user> <filter_name>", "")
		}
		return filters.AuthedRemoveMember(ctx, cmdStr[2], cmdStr[3], m.Author.ID, c.dependencies)
	}

	return getHelp("!filter help", filterUsage, filterSubcommands)
}
