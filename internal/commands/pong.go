package commands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/disgord/x/mux"
)

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func (c Command) Pong(s *discordgo.Session, m *discordgo.Message, ctx *mux.Context) {
	_, err := s.ChannelMessageSend(m.ChannelID, "Ping!")
	if err != nil {
		c.dependencies.Logger.Errorf("Error sending command: %s", err)
	}
}
