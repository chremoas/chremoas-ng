package commands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/disgord/x/mux"
	"go.uber.org/zap"
)

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func (c Command) Pong(s *discordgo.Session, m *discordgo.Message, ctx *mux.Context) {
	logger := c.dependencies.Logger.With(zap.String("command", "pong"))

	_, err := s.ChannelMessageSend(m.ChannelID, "Ping!")
	if err != nil {
		logger.Error("Error sending command", zap.Error(err))
	}
}
