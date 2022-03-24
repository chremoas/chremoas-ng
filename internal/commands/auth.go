package commands

import (
	"context"
	"strings"

	sl "github.com/bhechinger/spiffylogger"
	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/disgord/x/mux"
	"github.com/chremoas/chremoas-ng/internal/auth"
	"go.uber.org/zap"
)

const authUsage = `!auth <token>`

// Auth will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func (c Command) Auth(s *discordgo.Session, m *discordgo.Message, ctx *mux.Context) {
	spCtx, sp := sl.OpenSpan(c.ctx)
	defer sp.Close()

	sp.With(zap.String("command", "auth"))

	for _, message := range c.doAuth(spCtx, m) {
		_, err := s.ChannelMessageSendComplex(m.ChannelID, message)

		if err != nil {
			sp.Error("Error sending command", zap.Error(err))
		}
	}
}

func (c Command) doAuth(ctx context.Context, m *discordgo.Message) []*discordgo.MessageSend {
	_, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.Info("Received chat command", zap.String("content", m.Content))

	cmdStr := strings.Split(m.Content, " ")

	if len(cmdStr) < 2 {
		return getHelp("!auth help", authUsage, "")
	}

	switch cmdStr[1] {
	case "help":
		return getHelp("!auth help", authUsage, "")

	default:
		return auth.Confirm(ctx, cmdStr[1], m.Author.ID, c.dependencies)
	}
}
