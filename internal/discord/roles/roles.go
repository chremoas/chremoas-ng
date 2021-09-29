package roles

import (
	"encoding/json"

	sq "github.com/Masterminds/squirrel"
	"github.com/bwmarrin/discordgo"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	amqp "github.com/rabbitmq/amqp091-go"
)

// var matchDiscordError = regexp.MustCompile(`^The role '.*' already exists$`)

type Role struct {
	dependencies common.Dependencies
}

func New(deps common.Dependencies) *Role {
	return &Role{
		dependencies: deps,
	}
}

func (r Role) HandleMessage(deliveries <-chan amqp.Delivery, done chan error) {
	r.dependencies.Logger.Info("Started roles message handling")
	defer r.dependencies.Logger.Info("Completed roles message handling")

	for d := range deliveries {
		if len(d.Body) == 0 {
			err := d.Ack(false)
			if err != nil {
				r.dependencies.Logger.Errorf("Message has zero length body: %s", err)
			}
			continue
		}
		var body payloads.Payload
		err := json.Unmarshal(d.Body, &body)
		if err != nil {
			r.dependencies.Logger.Errorf("error unmarshalling payload: %s", err)
			continue
		}

		switch body.Action {
		case payloads.Upsert:
			err = r.upsert(body.Role)
		case payloads.Delete:
			err = r.delete(body.Role)
		default:
			r.dependencies.Logger.Errorf("Unknown action: %s", body.Action)
		}

		if err != nil {
			// we want to retry the message
			continue
		}

		err = d.Ack(false)
		if err != nil {
			r.dependencies.Logger.Errorf("Error Acking message: %s", err)
		}
	}

	r.dependencies.Logger.Infof("roles/HandleMessage: deliveries channel closed")
	done <- nil
}

// Only return an error if we want to keep the message and try again.
func (r Role) upsert(role *discordgo.Role) error {
	var err error

	if common.IgnoreRole(role.Name) {
		r.dependencies.Logger.Infof("Ignoring upsert request for role %s", role.Name)
		return nil
	}

	// Only one thing should write to discord at a time
	r.dependencies.Logger.Info("role.upsert() acquiring lock")
	r.dependencies.Session.Lock()
	defer func() {
		r.dependencies.Session.Unlock()
		r.dependencies.Logger.Info("role.upsert() released lock")
	}()

	// Check and see if this role has been created in discord or not
	if r.exists(role.Name) {
		// Update an existing role
		if r.dependencies.GuildID == "" || role.ID == "" || role.Name == "" {
			r.dependencies.Logger.Errorf("role.update() missing data: guildID=%s roleID=%s roleName=%s",
				r.dependencies.GuildID, role.ID, role.Name)
			return nil
		}

		r.dependencies.Logger.Infof("Update role `%s` with ID `%s`", role.Name, role.ID)
	} else {
		// Create a new role
		newRole, err := r.dependencies.Session.GuildRoleCreate(r.dependencies.GuildID)
		if err != nil {
			r.dependencies.Logger.Errorf("Error creating role: %s", err)
			return err
		}

		r.dependencies.Logger.Infof("Create role `%s` with ID `%s`", role.Name, newRole.ID)

		_, err = r.dependencies.DB.Update("roles").
			Set("chat_id", newRole.ID).
			Where(sq.Eq{"name": role.Name}).
			Query()
		if err != nil {
			r.dependencies.Logger.Errorf("Error updating role id in db: %s", err)
			return err
		}

		role.ID = newRole.ID
	}

	_, err = r.dependencies.Session.GuildRoleEdit(r.dependencies.GuildID, role.ID, role.Name, role.Color, role.Hoist,
		role.Permissions, role.Mentionable)
	if err != nil {
		r.dependencies.Logger.Errorf("Error editing role: %s", err)
		return err
	}

	r.dependencies.Logger.Infof("Upserted '%s' to discord", role.Name)

	return nil
}

// Only return an error if we want to keep the message and try again.
func (r Role) delete(role *discordgo.Role) error {
	if common.IgnoreRole(role.Name) {
		r.dependencies.Logger.Infof("Ignoring delete request for role %s", role.Name)
		return nil
	}

	// Only one thing should write to discord at a time
	r.dependencies.Logger.Info("role.create() acquiring lock")
	r.dependencies.Session.Lock()
	defer func() {
		r.dependencies.Session.Unlock()
		r.dependencies.Logger.Info("role.create() released lock")
	}()

	err := r.dependencies.Session.GuildRoleDelete(r.dependencies.GuildID, role.ID)
	if err != nil {
		if err.(*discordgo.RESTError).Response.StatusCode == 404 {
			r.dependencies.Logger.Warnf("Role doesn't exist in discord: %s", role.ID)
			return nil
		}
		r.dependencies.Logger.Errorf("Error deleting role: %s", err)
		return err
	}

	r.dependencies.Logger.Infof("Deleted '%s' from discord", role.Name)

	return nil
}

func (r Role) exists(name string) bool {
	roles, err := r.dependencies.Session.GuildRoles(r.dependencies.GuildID)
	if err != nil {
		r.dependencies.Logger.Errorf("Error fetching discord roles: %s", err)
		return true
	}

	for _, role := range roles {
		if name == role.Name {
			return true
		}
	}

	return false
}
