package commands

import (
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/disgord/x/mux"
	"github.com/chremoas/chremoas-ng/internal/auth"
	"go.uber.org/zap"
)

const authUsage = `!auth <token>`

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func (c Command) Auth(s *discordgo.Session, m *discordgo.Message, ctx *mux.Context) {
	logger := c.dependencies.Logger.With(zap.String("command", "auth"))

	for _, message := range c.doAuth(m, logger) {
		_, err := s.ChannelMessageSendComplex(m.ChannelID, message)

		// _, err := s.ChannelMessageSend(m.ChannelID, c.doSig(m, logger))
		if err != nil {
			logger.Error("Error sending command", zap.Error(err))
		}
	}
}

func (c Command) doAuth(m *discordgo.Message, logger *zap.Logger) []*discordgo.MessageSend {
	logger.Info("Received chat command", zap.String("content", m.Content))

	cmdStr := strings.Split(m.Content, " ")

	if len(cmdStr) < 2 {
		return getHelp("!auth help", authUsage, "")
	}

	switch cmdStr[1] {
	case "help":
		return getHelp("!auth help", authUsage, "")

	default:
		return auth.Confirm(cmdStr[1], m.Author.ID, c.dependencies)
	}
}
