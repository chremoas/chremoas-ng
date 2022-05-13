package common

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
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
	dependencies    Dependencies
	badDiscordUsers map[string]int
}

const deleteAfterCount = 10

func NewCheckAndDelete(deps Dependencies) CheckAndDelete {
	return CheckAndDelete{dependencies: deps}
}

func (cad CheckAndDelete) CheckAndDelete(ctx context.Context, memberID string, checkErr error) (bool, error) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	if restError, ok := checkErr.(discordgo.RESTError); ok {
		if restError.Response.StatusCode == 404 {
			if errCount, exists := cad.badDiscordUsers[memberID]; exists {

				sp.Warn("Failed to update user in discord, user not found",
					zap.Int("bad attempt count", cad.badDiscordUsers[memberID]),
				)

				if errCount > deleteAfterCount {
					sp.Warn("Deleting user after too many Discord failures",
						zap.Int("threshold", deleteAfterCount),
					)
					// Too many failures in Discord, deleting from chremoas
					query := cad.dependencies.DB.Select("character_id").
						From("user_character_map").
						Where(sq.Eq{"chat_id": memberID})

					sqlStr, args, err := query.ToSql()
					if err != nil {
						sp.Error("error getting sql", zap.Error(err))
						return true, err
					} else {
						sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
					}

					rows, err := query.QueryContext(ctx)
					if err != nil {
						sp.Error("error getting character list from the db", zap.Error(err))
						return true, err
					}

					defer func() {
						if err = rows.Close(); err != nil {
							sp.Error("error closing role", zap.Error(err))
						}
					}()

					var characterID int
					for rows.Next() {
						err = rows.Scan(&characterID)
						if err != nil {
							sp.Error("error scanning character id", zap.Error(err))
							return true, err
						}

						sp.With(zap.Int("character_id", characterID))

						sp.Warn("Deleting user's authentication codes")

						query := cad.dependencies.DB.Delete("authentication_codes").
							Where(sq.Eq{"character_id": characterID})

						sqlStr, args, err = query.ToSql()
						if err != nil {
							sp.Error("error getting sql", zap.Error(err))
							return true, err
						} else {
							sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
						}

						_, err := query.QueryContext(ctx)
						if err != nil {
							sp.Error("error deleting user's authentication codes from the db", zap.Error(err))
							return true, err
						}

						sp.Warn("Deleting user's character")

						query = cad.dependencies.DB.Delete("characters").
							Where(sq.Eq{"id": characterID})

						sqlStr, args, err = query.ToSql()
						if err != nil {
							sp.Error("error getting sql", zap.Error(err))
							return true, err
						} else {
							sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
						}

						_, err = query.QueryContext(ctx)
						if err != nil {
							sp.Error("error deleting user's character from the db", zap.Error(err))
							return true, err
						}
					}
				}
				cad.badDiscordUsers[memberID]++
			} else {
				cad.badDiscordUsers[memberID] = 1
			}

			return true, nil
		}
	}

	return false, nil
}
