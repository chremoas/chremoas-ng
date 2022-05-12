package members

import (
	"context"
	"encoding/json"

	sq "github.com/Masterminds/squirrel"
	sl "github.com/bhechinger/spiffylogger"
	"github.com/bwmarrin/discordgo"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

const deleteAfterCount = 10

type Member struct {
	dependencies common.Dependencies
	ctx          context.Context

	badDiscordUsers map[string]int
}

func New(ctx context.Context, deps common.Dependencies) *Member {
	return &Member{
		dependencies: deps,
		ctx:          ctx,
	}
}

func (m Member) HandleMessage(deliveries <-chan amqp.Delivery, done chan error) {
	ctx, sp := sl.OpenSpan(m.ctx)
	defer sp.Close()

	sp.With(zap.String("queue", "members"))

	sp.Info("Started members message handling")
	defer sp.Info("Completed members message handling")

	for d := range deliveries {
		func() {
			if len(d.Body) == 0 {
				sp.Info("message body was empty")
				err := d.Ack(false)
				if err != nil {
					sp.Error("Error ACKing message", zap.Error(err))
				}

				err = d.Reject(false)
				if err != nil {
					sp.Error("Error rejecting message", zap.Error(err))
				}

				return
			}

			var body payloads.MemberPayload
			err := json.Unmarshal(d.Body, &body)
			if err != nil {
				sp.Error("error unmarshalling payload", zap.Error(err))

				err = d.Reject(false)
				if err != nil {
					sp.Error("Error rejecting message", zap.Error(err))
				}

				return
			}

			_, sp = sl.OpenCorrelatedSpan(ctx, body.CorrelationID)
			defer sp.Close()

			sp.With(zap.Any("payload", body))

			sp.Debug("Handling message")

			m.dependencies.Session.Lock()
			defer func() {
				m.dependencies.Session.Unlock()
			}()

			if body.RoleID == "0" {
				err = d.Reject(false)
				if err != nil {
					sp.Error("Error rejecting invalid (RoleID==0) Role Add message: %s", zap.Error(err))
				}

				return
			}

			// We have the role's ID but the ignore list is the role names so let's look it up
			roles, err := m.dependencies.Session.GuildRoles(m.dependencies.GuildID)
			if err != nil {
				sp.Error("Error fetching discord roles", zap.Error(err))
				return
			}

			var roleName string
			for _, role := range roles {
				if body.RoleID == role.ID {
					roleName = role.Name
					break
				}
			}

			sp.With(zap.String("role_name", roleName))

			if common.IgnoreRole(roleName) {
				err = d.Reject(false)
				if err != nil {
					sp.Error("Error rejecting invalid (ignored role) Role Add message", zap.Error(err))
				}

				return
			}

			switch body.Action {
			case payloads.Add, payloads.Upsert:
				var sync bool
				query := m.dependencies.DB.Select("sync").
					From("roles").
					Where(sq.Eq{"chat_id": body.RoleID})

				sqlStr, args, err := query.ToSql()
				if err != nil {
					sp.Error("error getting sql", zap.Error(err))
					return
				} else {
					sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
				}

				err = query.Scan(&sync)
				if err != nil {
					sp.Error(
						"Error getting role sync status",
						zap.Error(err),
						zap.String("role", body.RoleID),
					)
					return
				}

				sp.With(zap.Bool("sync", sync))

				if !sync {
					err = d.Reject(false)
					if err != nil {
						sp.Error("Error rejecting role not set to sync", zap.Error(err))
					}

					return
				}

				err = m.dependencies.Session.GuildMemberRoleAdd(body.GuildID, body.MemberID, body.RoleID)
				if err != nil {
					handled, hErr := m.checkAndDelete(ctx, body.MemberID, err)
					if handled {
						return
					}
					if hErr != nil {
						sp.Error("Additional errors from checkAndDelete", zap.Error(hErr))
					}

					sp.Error("Error adding role to user", zap.Error(err))

					err = d.Reject(true)
					if err != nil {
						sp.Error("Error rejecting Role Add message: %s", zap.Error(err))
					}

					return
				}

			case payloads.Delete:
				err = m.dependencies.Session.GuildMemberRoleRemove(body.GuildID, body.MemberID, body.RoleID)
				if err != nil {
					handled, hErr := m.checkAndDelete(ctx, body.MemberID, err)
					if handled {
						return
					}
					if hErr != nil {
						sp.Error("Additional errors from checkAndDelete", zap.Error(hErr))
					}

					sp.Error("Error removing role from user", zap.Error(err))

					err = d.Reject(true)
					if err != nil {
						sp.Error("Error rejecting Role Remove message", zap.Error(err))
					}

					return
				}

			default:
				sp.Error("Unknown action")
			}

			err = d.Ack(false)
			if err != nil {
				sp.Error("Error ACKing message", zap.Error(err))
			}
		}()
	}

	done <- nil
}

func (m Member) checkAndDelete(ctx context.Context, memberID string, checkErr error) (bool, error) {
	ctx, sp := sl.OpenSpan(m.ctx)
	defer sp.Close()

	if restError, ok := checkErr.(discordgo.RESTError); ok {
		if restError.Response.StatusCode == 404 {
			if errCount, exists := m.badDiscordUsers[memberID]; exists {

				sp.Debug("Failed to update user in discord, user not found",
					zap.Int("bad attempt count", m.badDiscordUsers[memberID]),
				)

				if errCount > deleteAfterCount {
					sp.Warn("Deleting user after too many Discord failures",
						zap.Int("threshold", deleteAfterCount),
					)
					// Too many failures in Discord, deleting from chremoas
					query := m.dependencies.DB.Select("character_id").
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

						sp.Info("Deleting user's authentication codes")

						query := m.dependencies.DB.Delete("authentication_codes").
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

						sp.Info("Deleting user's character")

						query = m.dependencies.DB.Delete("characters").
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
				m.badDiscordUsers[memberID]++
			} else {
				m.badDiscordUsers[memberID] = 1
			}

			return true, nil
		}
	}

	return false, nil
}
