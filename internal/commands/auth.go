package commands

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/disgord/x/mux"
	"github.com/chremoas/chremoas-ng/internal/auth"
	"go.uber.org/zap"
)

const authHelpStr = `
Usage: !auth <token>
`

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func (c Command) Auth(s *discordgo.Session, m *discordgo.Message, ctx *mux.Context) {
	logger := c.dependencies.Logger.With(zap.String("command", "auth"))

	_, err := s.ChannelMessageSend(m.ChannelID, c.doAuth(m, logger))
	if err != nil {
		logger.Error("Error sending command", zap.Error(err))
	}
}

func (c Command) doAuth(m *discordgo.Message, logger *zap.Logger) string {
	logger.Info("Received chat command", zap.String("content", m.Content))

	cmdStr := strings.Split(m.Content, " ")

	if len(cmdStr) < 2 {
		return fmt.Sprintf("```%s```", authHelpStr)
	}

	switch cmdStr[1] {
	case "help":
		return fmt.Sprintf("```%s```", authHelpStr)

	default:
		return auth.Confirm(cmdStr[1], m.Author.ID, c.dependencies)
	}
}
