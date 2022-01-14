package members

import (
	"encoding/json"

	sq "github.com/Masterminds/squirrel"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	amqp "github.com/rabbitmq/amqp091-go"
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
	m.dependencies.Logger.Info("Started members message handling")
	defer m.dependencies.Logger.Info("Completed members message handling")

	for d := range deliveries {
		func() {
			if len(d.Body) == 0 {
				m.dependencies.Logger.Info("message body was empty")
				err := d.Ack(false)
				if err != nil {
					m.dependencies.Logger.Errorf("Error Acking message: %s", err)
				}

				err = d.Reject(false)
				if err != nil {
					m.dependencies.Logger.Errorf("Error Acking message: %s", err)
				}

				return
			}

			var body payloads.MemberPayload
			err := json.Unmarshal(d.Body, &body)
			if err != nil {
				m.dependencies.Logger.Errorf("error unmarshalling payload: %s", err)

				err = d.Reject(false)
				if err != nil {
					m.dependencies.Logger.Errorf("Error Acking message: %s", err)
				}

				return
			}

			m.dependencies.Logger.Debugf("Handling message for %s", body.MemberID)
			m.dependencies.Logger.Debug("members handler acquiring lock")
			m.dependencies.Session.Lock()
			defer func() {
				m.dependencies.Session.Unlock()
				m.dependencies.Logger.Debug("members handler released lock")
			}()

			if body.RoleID == "0" {
				err = d.Reject(false)
				if err != nil {
					m.dependencies.Logger.Errorf("Error Nacking invalid (RoleID==0) Role Add message: %s", err)
				}

				return
			}

			if common.IgnoreRole(body.RoleID) {
				err = d.Reject(false)
				if err != nil {
					m.dependencies.Logger.Errorf("Error Nacking invalid (ignored role) Role Add message: %s", err)
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
					m.dependencies.Logger.Errorf("Error getting role sync status: %s", err)
					return
				}

				if !sync {
					m.dependencies.Logger.Debugf("member handler: Skipping role not set to sync: %s", body.RoleID)
					err = d.Reject(false)
					if err != nil {
						m.dependencies.Logger.Errorf("Error Nacking Role not set to sync: %s", err)
					}

					return
				}

				err = m.dependencies.Session.GuildMemberRoleAdd(body.GuildID, body.MemberID, body.RoleID)
				if err != nil {
					m.dependencies.Logger.Errorf("Error adding role %s to user %s: %s",
						body.RoleID, body.MemberID, err)

					err = d.Reject(true)
					if err != nil {
						m.dependencies.Logger.Errorf("Error Nacking Role Add message: %s", err)
					}

					return
				}

			case payloads.Delete:
				err = m.dependencies.Session.GuildMemberRoleRemove(body.GuildID, body.MemberID, body.RoleID)
				if err != nil {
					m.dependencies.Logger.Errorf("Error removing role %s from user %s: %s",
						body.RoleID, body.MemberID, err)

					err = d.Reject(true)
					if err != nil {
						m.dependencies.Logger.Errorf("Error Nacking Role Remove message: %s", err)
					}

					return
				}

			default:
				m.dependencies.Logger.Errorf("Unknown action: %s", body.Action)
			}

			err = d.Ack(false)
			if err != nil {
				m.dependencies.Logger.Errorf("Error Acking message: %s", err)
			}
		}()
	}

	done <- nil
}
