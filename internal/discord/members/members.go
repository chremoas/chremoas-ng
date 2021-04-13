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

	if len(msg.Body) == 0 {
		// Returning nil will automatically send a FIN command to NSQ to mark the message as processed.
		return nil
	}
	var body payloads.Payload
	err := json.Unmarshal(msg.Body, &body)
	if err != nil {
		m.logger.Errorf("error unmarshalling payload: %s", err)
		return err
	}

	// we don't switch on Action because there is only one action, update.

	rows, err := m.db.Select("roles.chat_id").
		From("filters").
		Join("filter_membership ON filters.id = filter_membership.filter").
		Join("role_filters ON filters.id = role_filters.filter").
		Join("roles ON role_filters.role = roles.id").
		Where(sq.Eq{"filter_membership.user_id": body.Member}).
		Where(sq.Eq{"filters.namespace": viper.GetString("namespace")}).
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

	err = m.session.GuildMemberEdit(m.guildID, body.Member, roles)
	if err != nil {
		m.logger.Errorf("Error updating user %s: %s", body.Member, err)
	}

	return err
}
