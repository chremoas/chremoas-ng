package members

import (
	"encoding/json"

	sq "github.com/Masterminds/squirrel"
	"github.com/bwmarrin/discordgo"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	"github.com/nsqio/go-nsq"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type Member struct {
	logger  *zap.SugaredLogger
	session *discordgo.Session
	db      *sq.StatementBuilderType
	guildID string
}

func New(logger *zap.SugaredLogger, session *discordgo.Session, db *sq.StatementBuilderType) *Member {
	return &Member{
		logger:  logger,
		session: session,
		guildID: viper.GetString("bot.discordServerId"),
		db:      db,
	}
}

func (m Member) HandleMessage(msg *nsq.Message) error {
	var role string
	var roles []string

	m.logger.Info("Started members message handling")
	defer m.logger.Info("Completed members message handling")

	if len(msg.Body) == 0 {
		// Returning nil will automatically send a FIN command to NSQ to mark the message as processed.
		m.logger.Info("message body was empty")
		return nil
	}
	var body payloads.Payload
	err := json.Unmarshal(msg.Body, &body)
	if err != nil {
		m.logger.Errorf("error unmarshalling payload: %s", err)
		return err
	}

	rows, err := m.db.Select("roles.chat_id").
		From("filters").
		Join("filter_membership ON filters.id = filter_membership.filter").
		Join("role_filters ON filters.id = role_filters.filter").
		Join("roles ON role_filters.role = roles.id").
		Where(sq.Eq{"filter_membership.user_id": body.Member}).
		Query()
	if err != nil {
		m.logger.Errorf("error updating member `%s`: %s", body.Member, err)
		return err
	}

	for rows.Next() {
		err = rows.Scan(&role)
		if err != nil {
			m.logger.Errorf("error scanning role for update")
			return err
		}

		roles = append(roles, role)
	}

	member, err := m.session.GuildMember(m.guildID, body.Member)
	if err != nil {
		m.logger.Errorf("error getting guild member: %s", err)
	}

	removeRoles := compare(member.Roles, roles)
	if len(removeRoles) > 0 {
		m.logger.Infof("Removing roles from user %s: %s", body.Member, removeRoles)
	}
	for _, role = range removeRoles {
		err = m.session.GuildMemberRoleRemove(m.guildID, body.Member, role)
		if err != nil {
			m.logger.Errorf("Error removing role %s from user %s: %s", role, body.Member, err)
		}
	}

	addRoles := compare(roles, member.Roles)
	if len(addRoles) > 0 {
		m.logger.Infof("Adding roles to user %s: %s", body.Member, addRoles)
	}
	for _, role = range addRoles {
		err = m.session.GuildMemberRoleAdd(m.guildID, body.Member, role)
		if err != nil {
			m.logger.Errorf("Error adding role %s to user %s: %s", role, body.Member, err)
		}
	}

	return nil
}

// compare compares two string returning what the first one has that the second one doesn't
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