package common

import (
	"bytes"
	"context"
	"fmt"

	sl "github.com/bhechinger/spiffylogger"
	"go.uber.org/zap"
)

func SendChunkedMessage(ctx context.Context, channelID, title string, message []string, deps Dependencies) error {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	var (
		buffer bytes.Buffer

		firstChunk = true
	)

	for _, m := range message {
		// Check if the current role + desc pushes us over the limit for descriptions
		sp.Debug(
			"debug info",
			zap.Int("bugger_len", buffer.Len()),
			zap.Bool("firstChunk", firstChunk),
			zap.Int("EmbedLimitDescription", EmbedLimitDescription),
		)

		if buffer.Len()+len(m)+2 > EmbedLimitDescription {
			sp.Info(
				"Starting a new embed",
				zap.Int("buffer_len", buffer.Len()),
			)

			// send the current one and start a new one
			embed := NewEmbed()
			if firstChunk {
				sp.Info("First chunk so setting first title")
				embed.SetTitle(title)
				firstChunk = false
			} else {
				sp.Info("Not first chunk so setting continuation title")
				embed.SetTitle(fmt.Sprintf("%s (cont)", title))
			}

			embed.SetDescription(buffer.String())
			sp.Debug("description buffer for chunk", zap.String("buffer", buffer.String()))

			_, err := deps.Session.ChannelMessageSendEmbed(channelID, embed.GetMessageEmbed())
			if err != nil {
				sp.Error("Error sending message", zap.Error(err))
				return err
			}
			buffer.Reset()
			buffer.WriteString(fmt.Sprintf("%s\n", m))
		} else {
			sp.Info(
				"still within chunk boundary, appending",
				zap.Int("buffer_len", buffer.Len()),
			)

			buffer.WriteString(fmt.Sprintf("%s\n", m))
		}
	}

	sp.Debug(
		"leftover buffer to send",
		zap.String("buffer", buffer.String()),
		zap.Int("buffer_len", buffer.Len()),
	)

	embed := NewEmbed()
	if firstChunk {
		embed.SetTitle(title)
	}
	embed.SetDescription(buffer.String())

	_, err := deps.Session.ChannelMessageSendEmbed(channelID, embed.GetMessageEmbed())
	if err != nil {
		sp.Error("Error sending message", zap.Error(err))
		return err
	}

	return nil
}
