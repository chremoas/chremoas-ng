package roles

import (
	"encoding/json"

	sq "github.com/Masterminds/squirrel"
	"github.com/bwmarrin/discordgo"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	"github.com/nsqio/go-nsq"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

//var matchDiscordError = regexp.MustCompile(`^The role '.*' already exists$`)

type Role struct {
	logger  *zap.SugaredLogger
	session *discordgo.Session
	db      *sq.StatementBuilderType
	guildID string
}

func New(logger *zap.SugaredLogger, session *discordgo.Session, db *sq.StatementBuilderType) *Role {
	return &Role{
		logger:  logger,
		session: session,
		guildID: viper.GetString("bot.discordServerId"),
		db:      db,
	}
}

func (r Role) HandleMessage(m *nsq.Message) error {
	if len(m.Body) == 0 {
		// Returning nil will automatically send a FIN command to NSQ to mark the message as processed.
		return nil
	}
	var body payloads.Payload
	err := json.Unmarshal(m.Body, &body)
	if err != nil {
		r.logger.Errorf("error unmarshalling payload: %s", err)
		return err
	}

	switch body.Action {
	case payloads.Create:
		err = r.create(body.Role)
		return err
	case payloads.Update:
		err = r.update(body.Role)
		return err
	case payloads.Delete:
		err = r.delete(body.Role)
		return err
	default:
		r.logger.Errorf("Unknown action: %s", body.Action)
		return nil // We don't want to rery this, it'll never work
	}
}

func (r Role) create(role *discordgo.Role) error {
	// Check and see if this role has been created in discored or not
	rList, err := r.session.GuildRoles(r.guildID)
	for _, checkRole := range rList {
		if checkRole.Name == role.Name {
			r.logger.Infof("Role already exists in Discord: %s", role.Name)
			return nil
		}
	}

	// Only one thing should write to discord at a time
	r.logger.Info("role.create() acquiring lock")
	r.session.Lock()
	defer func() {
		r.session.Unlock()
		r.logger.Info("role.create() released lock")
	}()

	newRole, err := r.session.GuildRoleCreate(r.guildID)
	if err != nil {
		r.logger.Errorf("Error creating role: %s", err)
		return err
	}

	r.logger.Infof("Create role `%s` with ID `%s`", role.Name, newRole.ID)

	_, err = r.db.Update("roles").
		Set("chat_id", newRole.ID).
		Where(sq.Eq{"name": role.Name}).
		Query()
	if err != nil {
		r.logger.Errorf("Error updating role id in db: %s", err)
		return err
	}

	_, err = r.session.GuildRoleEdit(r.guildID, newRole.ID, role.Name, role.Color, role.Hoist,
		role.Permissions, role.Mentionable)
	if err != nil {
		r.logger.Errorf("Error editing role: %s", err)
		return err
	}

	r.logger.Infof("Added '%s' to discord", role.Name)

	return nil
}

func (r Role) update(role *discordgo.Role) error {
	// Only one thing should write to discord at a time
	r.logger.Info("role.create() acquiring lock")
	r.session.Lock()
	defer func() {
		r.session.Unlock()
		r.logger.Info("role.create() released lock")
	}()

	_, err := r.session.GuildRoleEdit(r.guildID, role.ID, role.Name, role.Color, role.Hoist,
		role.Permissions, role.Mentionable)
	if err != nil {
		r.logger.Errorf("Error updating role: %s", err)
		return err
	}

	r.logger.Infof("Updated '%s' in discord", role.Name)

	return nil
}

func (r Role) delete(role *discordgo.Role) error {
	// Only one thing should write to discord at a time
	r.logger.Info("role.create() acquiring lock")
	r.session.Lock()
	defer func() {
		r.session.Unlock()
		r.logger.Info("role.create() released lock")
	}()

	err := r.session.GuildRoleDelete(r.guildID, role.ID)
	if err != nil {
		if err.(*discordgo.RESTError).Response.StatusCode == 404 {
			r.logger.Warnf("Role doesn't exist in discord: %s", role.ID)
			return nil
		}
		r.logger.Errorf("Error deleting role: %s", err)
		return err
	}

	r.logger.Infof("Deleted '%s' from discord", role.Name)

	return nil
}
