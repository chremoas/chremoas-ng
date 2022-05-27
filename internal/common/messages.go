package common

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func makeString(message string, sender string, sign string) string {
	var output string

	if len(sender) > 0 {
		output = fmt.Sprintf("<@%s> ", sender)
	}
	output = fmt.Sprintf("%s %s %s", output, sign, message)

	return output
}

func SendSuccessf(sender *string, format string, args ...interface{}) []*discordgo.MessageSend {
	return SendSuccess(fmt.Sprintf(format, args...), *sender)
}

func SendSuccess(message string, sender ...string) []*discordgo.MessageSend {
	var (
		s        string
		messages []*discordgo.MessageSend
	)

	if len(sender) == 0 {
		s = ""
	} else {
		s = sender[0]
	}
	return append(messages, &discordgo.MessageSend{Content: makeString(message, s, ":white_check_mark:")})
}

func SendErrorf(sender *string, format string, args ...interface{}) []*discordgo.MessageSend {
	return SendSuccess(fmt.Sprintf(format, args...), *sender)
}

func SendError(message string, sender ...string) []*discordgo.MessageSend {
	var (
		s        string
		messages []*discordgo.MessageSend
	)

	if len(sender) == 0 {
		s = ""
	} else {
		s = sender[0]
	}
	return append(messages, &discordgo.MessageSend{Content: makeString(message, s, ":warning:")})
}

func SendFatalf(sender *string, format string, args ...interface{}) []*discordgo.MessageSend {
	return SendSuccess(fmt.Sprintf(format, args...), *sender)
}

func SendFatal(message string, sender ...string) []*discordgo.MessageSend {
	var (
		s        string
		messages []*discordgo.MessageSend
	)

	if len(sender) == 0 {
		s = ""
	} else {
		s = sender[0]
	}
	return append(messages, &discordgo.MessageSend{Content: makeString(message, s, ":octagonal_sign:")})
}
