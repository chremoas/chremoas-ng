package members

import (
	"encoding/json"

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

			m.dependencies.Logger.Infof("Handling message for %s", body.MemberID)
			m.dependencies.Logger.Info("members handler acquiring lock")
			m.dependencies.Session.Lock()
			defer func() {
				m.dependencies.Session.Unlock()
				m.dependencies.Logger.Info("members handler released lock")
			}()

			switch body.Action {
			case payloads.Add:
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

	m.dependencies.Logger.Infof("members/HandleMessage: deliveries channel closed")
	done <- nil
}

// compare two string returning what the first one has that the second one doesn't
// func compare(a, b []string) []string {
// 	for i := len(a) - 1; i >= 0; i-- {
// 		for _, vD := range b {
// 			if a[i] == vD {
// 				a = append(a[:i], a[i+1:]...)
// 				break
// 			}
// 		}
// 	}
// 	return a
// }

// rows, err := m.dependencies.DB.Select("roles.chat_id").
// 	From("filters").
// 	Join("filter_membership ON filters.id = filter_membership.filter").
// 	Join("role_filters ON filters.id = role_filters.filter").
// 	Join("roles ON role_filters.role = roles.id").
// 	Where(sq.Eq{"filter_membership.user_id": body.Member}).
// 	Where(sq.Eq{"roles.sync": true}).
// 	Query()
// if err != nil {
// 	m.dependencies.Logger.Errorf("error updating member `%s`: %s", body.Member, err)
// 	return
// }