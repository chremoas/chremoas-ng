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
	_, err := s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("```Version: %s```", version))
	if err != nil {
		c.dependencies.Logger.Error("Error sending command",
			zap.Error(err), zap.String("command", "version"))
	}
}
