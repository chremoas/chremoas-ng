package common

import (
	"context"
	"fmt"

	sl "github.com/bhechinger/spiffylogger"
	"github.com/bwmarrin/discordgo"
	"github.com/spf13/viper"
	"go.uber.org/zap"
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

type CheckAndDelete struct {
	dependencies Dependencies
}

func NewCheckAndDelete(deps Dependencies) CheckAndDelete {
	return CheckAndDelete{dependencies: deps}
}

func (cad CheckAndDelete) CheckAndDelete(ctx context.Context, discordID string, checkErr error) (bool, error) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("discordID", discordID),
		zap.NamedError("checkErr", checkErr),
	)

	sp.Warn("Got an error, checking")

	if restError, ok := checkErr.(discordgo.RESTError); ok {
		sp.With(zap.Int("http_status_code", restError.Response.StatusCode))
		sp.Warn("Received rest error")

		if restError.Response.StatusCode == 404 {
			sp.Warn("Failed to update user in discord, user not found")

			characters, err := cad.dependencies.Storage.GetDiscordCharacters(ctx, discordID)
			if err != nil {
				return true, err
			}

			for c := range characters {
				sp.With(zap.Any("character", characters[c]))

				sp.Warn("Deleting user's authentication codes")
				err := cad.dependencies.Storage.DeleteAuthCodes(ctx, characters[c].ID)
				if err != nil {
					return true, err
				}

				sp.Warn("Deleting user's character")
				err = cad.dependencies.Storage.DeleteCharacter(ctx, characters[c].ID)
				if err != nil {
					return true, err
				}
			}
		}
	}

	return false, nil
}
