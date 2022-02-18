package common

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/spf13/viper"
)

func makeString(message string, sender string, sign string) string {
	var output string

	if len(sender) > 0 {
		output = fmt.Sprintf("<@%s> ", sender)
	}
	output = fmt.Sprintf("%s %s %s", output, sign, message)

	return output
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

func GetTopic(topic string) string {
	return fmt.Sprintf("%s-discord.%s", viper.GetString("namespace"), topic)
}

func NewConnectionString() string {
	return viper.GetString("database.driver") +
		"://" +
		viper.GetString("database.username") +
		":" +
		viper.GetString("database.password") +
		"@" +
		viper.GetString("database.host") +
		":" +
		fmt.Sprintf("%d", viper.GetInt("database.port")) +
		"/" +
		viper.GetString("database.database") +
		"?" +
		viper.GetString("database.options")
}
