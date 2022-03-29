package commands

import (
	sl "github.com/bhechinger/spiffylogger"
	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/disgord/x/mux"
	"go.uber.org/zap"
)

// Pong will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func (c Command) Pong(s *discordgo.Session, m *discordgo.Message, _ *mux.Context) {
	_, sp := sl.OpenCorrelatedSpan(c.ctx, sl.NewID())
	defer sp.Close()

	sp.With(zap.String("command", "pong"))

	_, err := s.ChannelMessageSend(m.ChannelID, "Ping!")
	if err != nil {
		sp.Error("Error sending command", zap.Error(err))
	}
}
