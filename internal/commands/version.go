package commands

import (
	_ "embed"

	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/disgord/x/mux"
	"github.com/chremoas/chremoas-ng/internal/common"
	"go.uber.org/zap"
)

//go:embed VERSION
var version string

// Version will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func (c Command) Version(s *discordgo.Session, m *discordgo.Message, ctx *mux.Context) {
	logger := c.dependencies.Logger.With(zap.String("command", "version"))

	logger.Info("Received chat command", zap.String("content", m.Content))

	embed := common.NewEmbed()
	embed.AddField("version", version)

	_, err := s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{Embed: embed.GetMessageEmbed()})
	if err != nil {
		logger.Error("Error sending command", zap.Error(err))
	}
}
