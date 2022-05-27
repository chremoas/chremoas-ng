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
	return SendSuccess(sender, fmt.Sprintf(format, args...))
}

func SendSuccess(sender *string, message string) []*discordgo.MessageSend {
	var (
		s        string
		messages []*discordgo.MessageSend
	)

	if sender == nil {
		s = ""
	} else {
		s = *sender
	}

	return append(messages, &discordgo.MessageSend{Content: makeString(message, s, ":white_check_mark:")})
}

func SendErrorf(sender *string, format string, args ...interface{}) []*discordgo.MessageSend {
	return SendError(sender, fmt.Sprintf(format, args...))
}

func SendError(sender *string, message string) []*discordgo.MessageSend {
	var (
		s        string
		messages []*discordgo.MessageSend
	)

	if sender == nil {
		s = ""
	} else {
		s = *sender
	}

	return append(messages, &discordgo.MessageSend{Content: makeString(message, s, ":warning:")})
}

func SendFatalf(sender *string, format string, args ...interface{}) []*discordgo.MessageSend {
	return SendFatal(sender, fmt.Sprintf(format, args...))
}

func SendFatal(sender *string, message string) []*discordgo.MessageSend {
	var (
		s        string
		messages []*discordgo.MessageSend
	)

	if sender == nil {
		s = ""
	} else {
		s = *sender
	}

	return append(messages, &discordgo.MessageSend{Content: makeString(message, s, ":octagonal_sign:")})
}
