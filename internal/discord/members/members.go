package members

import (
	"context"
	"encoding/json"

	sl "github.com/bhechinger/spiffylogger"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type Member struct {
	dependencies common.Dependencies
	ctx          context.Context
	cad          common.CheckAndDelete
}

func New(ctx context.Context, deps common.Dependencies) *Member {
	return &Member{
		dependencies: deps,
		ctx:          ctx,
		cad:          common.NewCheckAndDelete(deps),
	}
}

func (m Member) HandleMessage(deliveries <-chan amqp.Delivery, done chan error, threadID int) {
	ctx, sp := sl.OpenSpan(m.ctx)
	defer sp.Close()

	sp.With(
		zap.String("queue", "members"),
		zap.Int("threadID", threadID),
	)

	sp.Info("Started members message handling")
	defer sp.Info("Completed members message handling")

	for d := range deliveries {
		func() {
			sp.Debug("processing delivery")
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
				role, err := m.dependencies.Storage.GetRoleByChatID(ctx, body.RoleID)
				if err != nil {
					sp.Error("Error getting role", zap.Error(err))
					err = d.Reject(false)
					if err != nil {
						sp.Error("Error rejecting invalid (ignored role) Role Add message", zap.Error(err))
					}
					return
				}

				sp.With(zap.Bool("sync", role.Sync))

				if !role.Sync {
					err = d.Reject(false)
					if err != nil {
						sp.Error("Rejecting role not set to sync", zap.Error(err))
					}

					return
				}

				err = m.dependencies.Session.GuildMemberRoleAdd(body.GuildID, body.MemberID, body.RoleID)
				if err != nil {
					handled, hErr := m.cad.CheckAndDelete(ctx, body.MemberID, err)
					if hErr != nil {
						sp.Error("Additional errors from checkAndDelete", zap.Error(hErr))
					}
					if handled {
						return
					}

					sp.Error("Error adding role to user", zap.Error(err), zap.NamedError("hErr", hErr))

					err = d.Reject(true)
					if err != nil {
						sp.Error("Error rejecting Role Add message: %s", zap.Error(err))
					}

					return
				}

			case payloads.Delete:
				err = m.dependencies.Session.GuildMemberRoleRemove(body.GuildID, body.MemberID, body.RoleID)
				if err != nil {
					handled, hErr := m.cad.CheckAndDelete(ctx, body.MemberID, err)
					if hErr != nil {
						sp.Error("Additional errors from checkAndDelete", zap.Error(hErr))
					}
					if handled {
						return
					}

					sp.Error("Error removing role from user", zap.Error(err), zap.NamedError("hErr", hErr))

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
