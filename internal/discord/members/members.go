package members

import (
	"encoding/json"

	sq "github.com/Masterminds/squirrel"
	"github.com/bhechinger/go-sets"
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
	var (
		role   string
		roles  sets.StringSet
		dRoles sets.StringSet
	)

	m.dependencies.Logger.Info("Started members message handling")
	defer m.dependencies.Logger.Info("Completed members message handling")

	for d := range deliveries {
		if len(d.Body) == 0 {
			m.dependencies.Logger.Info("message body was empty")
			err := d.Ack(false)
			if err != nil {
				m.dependencies.Logger.Errorf("Error Acking message: %s", err)
			}
			continue
		}

		var body payloads.Payload
		err := json.Unmarshal(d.Body, &body)
		if err != nil {
			m.dependencies.Logger.Errorf("error unmarshalling payload: %s", err)
			continue
		}

		rows, err := m.dependencies.DB.Select("roles.chat_id").
			From("filters").
			Join("filter_membership ON filters.id = filter_membership.filter").
			Join("role_filters ON filters.id = role_filters.filter").
			Join("roles ON role_filters.role = roles.id").
			Where(sq.Eq{"filter_membership.user_id": body.Member}).
			Query()
		if err != nil {
			m.dependencies.Logger.Errorf("error updating member `%s`: %s", body.Member, err)
			continue
		}

		for rows.Next() {
			err = rows.Scan(&role)
			if err != nil {
				m.dependencies.Logger.Errorf("error scanning role for update")
				continue
			}

			if role != "0" {
				roles.Add(role)
			}
		}

		member, err := m.dependencies.Session.GuildMember(m.dependencies.GuildID, body.Member)
		if err != nil {
			m.dependencies.Logger.Errorf("error getting guild member '%s': %s", body.Member, err)
			// ditch the message
			err = d.Ack(false)
			if err != nil {
				m.dependencies.Logger.Errorf("Error Acking message: %s", err)
			}
			continue
		}

		dRoles.FromSlice(member.Roles)

		m.dependencies.Logger.Infof("member.Roles: %+v\n", dRoles)
		m.dependencies.Logger.Infof("roles: %+v\n", roles)

		removeRoles := dRoles.Difference(&roles)
		removeRoles := compare(member.Roles, roles)
		if len(removeRoles) > 0 {
			m.dependencies.Logger.Infof("Removing roles from user %s: %s", body.Member, removeRoles)
		}
		for _, role = range removeRoles {
			err = m.dependencies.Session.GuildMemberRoleRemove(m.dependencies.GuildID, body.Member, role)
			if err != nil {
				m.dependencies.Logger.Errorf("Error removing role %s from user %s: %s", role, body.Member, err)
			}
		}

		addRoles := roles.Difference(&dRoles)
		addRoles := compare(roles, member.Roles)
		if len(addRoles) > 0 {
			m.dependencies.Logger.Infof("Adding roles to user %s: %s", body.Member, addRoles)
			for _, role = range addRoles {
				if role != "0" {
					err = m.dependencies.Session.GuildMemberRoleAdd(m.dependencies.GuildID, body.Member, role)
					if err != nil {
						m.dependencies.Logger.Errorf("Error adding role %s to user %s: %s", role, body.Member, err)
						break
					}
				}
			}
		}
		err = d.Ack(false)
		if err != nil {
			m.dependencies.Logger.Errorf("Error Acking message: %s", err)
		}
	}

	m.dependencies.Logger.Infof("members/HandleMessage: deliveries channel closed")
	done <- nil
}

// compare two string returning what the first one has that the second one doesn't
func compare(a, b []string) []string {
	for i := len(a) - 1; i >= 0; i-- {
		for _, vD := range b {
			if a[i] == vD {
				a = append(a[:i], a[i+1:]...)
				break
			}
		}
	}
	return a
}
