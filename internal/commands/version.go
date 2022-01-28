package commands

import (
	_ "embed"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/disgord/x/mux"
	"go.uber.org/zap"
)

//go:embed VERSION
var version string

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func (c Command) Version(s *discordgo.Session, m *discordgo.Message, ctx *mux.Context) {
	logger := c.dependencies.Logger.With(zap.String("command", "version"))

	logger.Info("Received chat command", zap.String("content", m.Content))

	_, err := s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("```Version: %s```", version))
	if err != nil {
		logger.Error("Error sending command", zap.Error(err))
	}
}
