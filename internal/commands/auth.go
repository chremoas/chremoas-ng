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
	_, err := s.ChannelMessageSend(m.ChannelID, c.doAuth(s, m, ctx))
	if err != nil {
		c.dependencies.Logger.Error("Error sending command",
			zap.Error(err), zap.String("command", "auth"))
	}
}

func (c Command) doAuth(_ *discordgo.Session, m *discordgo.Message, _ *mux.Context) string {
	c.dependencies.Logger.Info("Received",
		zap.String("content", m.Content), zap.String("command", "auth"))
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
