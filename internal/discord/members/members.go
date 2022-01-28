package members

import (
	"encoding/json"

	sq "github.com/Masterminds/squirrel"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type Member struct {
	dependencies common.Dependencies
}

func New(deps common.Dependencies) *Member {
	return &Member{
		dependencies: deps,
	}
}

func (m Member) HandleMessage(deliveries <-chan amqp.Delivery, done chan error) {
	logger := m.dependencies.Logger.With(zap.String("queue", "members"))

	logger.Info("Started members message handling")
	defer logger.Info("Completed members message handling")

	for d := range deliveries {
		func() {
			if len(d.Body) == 0 {
				logger.Info("message body was empty")
				err := d.Ack(false)
				if err != nil {
					logger.Error("Error ACKing message", zap.Error(err))
				}

				err = d.Reject(false)
				if err != nil {
					logger.Error("Error ACKing message", zap.Error(err))
				}

				return
			}

			var body payloads.MemberPayload
			err := json.Unmarshal(d.Body, &body)
			if err != nil {
				logger.Error("error unmarshalling payload", zap.Error(err))

				err = d.Reject(false)
				if err != nil {
					logger.Error("Error ACKing message", zap.Error(err))
				}

				return
			}

			logger.Debug("Handling message", zap.Any("payload", body))

			m.dependencies.Session.Lock()
			defer func() {
				m.dependencies.Session.Unlock()
			}()

			if body.RoleID == "0" {
				err = d.Reject(false)
				if err != nil {
					logger.Error("Error NACKing invalid (RoleID==0) Role Add message: %s", zap.Error(err))
				}

				return
			}

			// We have the role's ID but the ignore list is the role names so let's look it up
			roles, err := m.dependencies.Session.GuildRoles(m.dependencies.GuildID)
			if err != nil {
				logger.Error("Error fetching discord roles", zap.Error(err))
				return
			}

			var roleName string
			for _, role := range roles {
				if body.RoleID == role.ID {
					roleName = role.Name
					break
				}
			}

			if common.IgnoreRole(roleName) {
				err = d.Reject(false)
				if err != nil {
					logger.Error("Error NACKing invalid (ignored role) Role Add message",
						zap.Error(err), zap.String("role", roleName))
				}

				return
			}

			switch body.Action {
			case payloads.Add, payloads.Upsert:
				var sync bool
				err = m.dependencies.DB.Select("sync").
					From("roles").
					Where(sq.Eq{"chat_id": body.RoleID}).
					Scan(&sync)
				if err != nil {
					logger.Error("Error getting role sync status",
						zap.Error(err), zap.String("role", body.RoleID))
					return
				}

				if !sync {
					err = d.Reject(false)
					if err != nil {
						logger.Error("Error NACKing role not set to sync",
							zap.Error(err), zap.String("role", body.RoleID))
					}

					return
				}

				err = m.dependencies.Session.GuildMemberRoleAdd(body.GuildID, body.MemberID, body.RoleID)
				if err != nil {
					logger.Error("Error adding role to user",
						zap.String("role", body.RoleID),
						zap.String("member id", body.MemberID),
						zap.Error(err))

					err = d.Reject(true)
					if err != nil {
						logger.Error("Error NACKing Role Add message: %s", zap.Error(err))
					}

					return
				}

			case payloads.Delete:
				err = m.dependencies.Session.GuildMemberRoleRemove(body.GuildID, body.MemberID, body.RoleID)
				if err != nil {
					logger.Error("Error removing role from user",
						zap.String("role", body.RoleID),
						zap.String("member id", body.MemberID),
						zap.Error(err))

					err = d.Reject(true)
					if err != nil {
						logger.Error("Error NACKing Role Remove message", zap.Error(err))
					}

					return
				}

			default:
				logger.Error("Unknown action", zap.Any("action", body.Action))
			}

			err = d.Ack(false)
			if err != nil {
				logger.Error("Error ACKing message", zap.Error(err))
			}
		}()
	}

	done <- nil
}
