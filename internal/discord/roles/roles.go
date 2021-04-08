package roles

import (
	"encoding/json"
	"fmt"

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

	fmt.Printf("%+v\n", body)

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

func (r Role) create(role discordgo.Role) error {
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

func (r Role) update(role discordgo.Role) error {
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

func (r Role) delete(role discordgo.Role) error {
	// Only one thing should write to discord at a time
	r.logger.Info("role.create() acquiring lock")
	r.session.Lock()
	defer func() {
		r.session.Unlock()
		r.logger.Info("role.create() released lock")
	}()

	err := r.session.GuildRoleDelete(r.guildID, role.ID)
	if err != nil {
		r.logger.Errorf("Error deleting role: %s", err)
		return err
	}

	r.logger.Infof("Deleted '%s' from discord", role.Name)

	return nil
}

//
//func (r Role) AddDiscordRole(name string, logger *zap.SugaredLogger, s *discordgo.Session) error {
//	// TODO: push to queue
//	_, err := h.clients.discord.CreateRole(ctx, &discord.CreateRoleRequest{Name: name})
//	if err != nil {
//		if matchDiscordError.MatchString(err.Error()) {
//			// The role list was cached most likely so we'll pretend we didn't try
//			// to create it just now. -brian
//		} else {
//			msg := fmt.Sprintf("addDiscordRole '%s': %s", name, err.Error())
//			logger.Error(msg)
//			return err
//		}
//	}
//
//	return nil
//}
//
//func CreateRole() string {
//	err := dgh.roleMap.UpdateRoles()
//	if err != nil {
//		dgh.Logger.Error(err.Error())
//		return err
//	}
//
//	allRoles := dgh.roleMap.GetRoles()
//
//	for key := range allRoles {
//		if allRoles[key].Name == request.Name {
//			dgh.Logger.Sugar().Errorf("The role '%s' already exists", allRoles[key].Name)
//			return fmt.Errorf("The role '%s' already exists", allRoles[key].Name)
//		}
//	}
//
//	role, err := dgh.client.CreateRole(dgh.discordServerId)
//	if err != nil {
//		dgh.Logger.Error(err.Error())
//		return err
//	}
//
//	editedRole, err := dgh.client.EditRole(dgh.discordServerId, role.ID, request.Name, int(request.Color), int64(request.Permissions), request.Hoist, request.Mentionable)
//	if err != nil {
//		deleteErr := dgh.client.DeleteRole(dgh.discordServerId, role.ID)
//		if deleteErr != nil {
//			dgh.Logger.Sugar().Errorf("edit failure (%s), delete failure (%s)", err.Error(), deleteErr.Error())
//			return errors.New(fmt.Sprintf("edit failure (%s), delete failure (%s)", err.Error(), deleteErr.Error()))
//		}
//
//		dgh.Logger.Error(err.Error())
//		return err
//	}
//
//	//Now validate the edited role
//	if !validateRole(request, editedRole) {
//		err = dgh.client.DeleteRole(dgh.discordServerId, role.ID)
//		if err != nil {
//			dgh.Logger.Sugar().Errorf("attempted to delete role due to invalid response but received error (%s)", err.Error())
//			return errors.New(fmt.Sprintf("attempted to delete role due to invalid response but received error (%s)", err.Error()))
//		}
//
//		dgh.Logger.Error("role create failed due to invalid response from discord")
//		return errors.New("role create failed due to invalid response from discord")
//	}
//
//	response.RoleId = editedRole.ID
//
//	// Reset cache as we've made changes to discord that need to be picked up next run
//	dgh.lastRoleCall = dgh.lastRoleCall.AddDate(0, 0, -1)
//	return nil
//}
