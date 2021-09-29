package common

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/spf13/viper"
)

func GetUsername(userID interface{}, discord *discordgo.Session) string {
	var _userID string

	switch userID.(type) {
	case int:
		_userID = fmt.Sprintf("%d", userID.(int))
	case string:
		_userID = userID.(string)
	}

	user, err := discord.GuildMember(
		viper.GetString("bot.discordServerId"),
		_userID,
	)
	if err != nil {
		return _userID
	} else {
		if user.Nick != "" {
			return user.Nick
		}
		return user.User.Username
	}
}

func IgnoreRole(role string) bool {
	ignoredRoles := viper.GetStringSlice("bot.ignoredRoles")
	ignoredRoles = append(ignoredRoles, "@everyone")

	for _, r := range ignoredRoles {
		if role == r {
			return true
		}
	}

	return false
}