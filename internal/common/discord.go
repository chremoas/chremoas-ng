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
	case int32:
		_userID = fmt.Sprintf("%d", userID.(int32))
	case int64:
		_userID = fmt.Sprintf("%d", userID.(int64))
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
		zap.String("error_type", fmt.Sprintf("%T", checkErr)),
	)

	sp.Warn("Got an error, checking")

	if restError, ok := checkErr.(*discordgo.RESTError); ok {
		// if strings.Contains(checkErr.Error(), "HTTP 404 Not Found") {
		sp.With(
			zap.Int("StatusCode", restError.Response.StatusCode),
			zap.String("Status", restError.Response.Status),
		)
		sp.Info("Checking response code")
		if restError.Response.StatusCode == 404 {
			sp.Warn("Failed to update user in discord, user not found")

			err := cad.dependencies.Storage.DeleteDiscordUser(ctx, discordID)
			if err != nil {
				sp.Error("Error deleting discord user", zap.Error(err))
				return true, err
			}
		}
	}

	return false, nil
}
