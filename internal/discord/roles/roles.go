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

			err = d.Reject(false)
			if err != nil {
				r.dependencies.Logger.Errorf("Error Acking message: %s", err)
			}

			continue
		}

		var body payloads.RolePayload
		err := json.Unmarshal(d.Body, &body)
		if err != nil {
			r.dependencies.Logger.Errorf("error unmarshalling payload: %s", err)

			err = d.Reject(false)
			if err != nil {
				r.dependencies.Logger.Errorf("Error Acking message: %s", err)
			}

			continue
		}

		if common.IgnoreRole(body.Role.Name) {
			r.dependencies.Logger.Infof("Ignoring request for role %s", body.Role.Name)

			err = d.Reject(false)
			if err != nil {
				r.dependencies.Logger.Errorf("Error Acking message: %s", err)
			}

			continue
		}

		switch body.Action {
		case payloads.Upsert:
			err = r.upsert(body)
		case payloads.Delete:
			err = r.delete(body)
		default:
			r.dependencies.Logger.Errorf("Unknown action: %s", body.Action)
		}

		if err != nil {
			// we want to retry the message
			err = d.Reject(true)
			if err != nil {
				r.dependencies.Logger.Errorf("Error Nacking message: %s", err)
			}

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
func (r Role) upsert(role payloads.RolePayload) error {
	var err error
	var sync bool

	// Only one thing should write to discord at a time
	r.dependencies.Logger.Info("role.upsert() acquiring lock")
	r.dependencies.Session.Lock()
	defer func() {
		r.dependencies.Session.Unlock()
		r.dependencies.Logger.Info("role.upsert() released lock")
	}()

	err = r.dependencies.DB.Select("sync").
		From("roles").
		Where(sq.Eq{"name": role.Role.Name}).
		Scan(&sync)
	if err != nil {
		r.dependencies.Logger.Errorf("Error getting role sync status: %s", err)
		return err
	}

	// If this role isn't set to sync, ignore it.
	if !sync {
		r.dependencies.Logger.Debugf("role.upsert(): Skipping role not set to sync: %s", role.Role.Name)
		return nil
	}

	// Check and see if this role has been created in discord or not
	if r.exists(role.Role.Name, role.GuildID) {
		// Update an existing role
		if role.GuildID == "" || role.Role.ID == "" || role.Role.Name == "" {
			r.dependencies.Logger.Errorf("role.update() missing data: guildID=%s roleID=%s roleName=%s",
				role.GuildID, role.Role.ID, role.Role.Name)
			return nil
		}

		r.dependencies.Logger.Infof("Update role `%s` with ID `%s`", role.Role.Name, role.Role.ID)
	} else {
		// Create a new role
		newRole, err := r.dependencies.Session.GuildRoleCreate(role.GuildID)
		if err != nil {
			r.dependencies.Logger.Errorf("Error creating role: %s", err)
			return err
		}

		r.dependencies.Logger.Infof("Create role `%s` with ID `%s`", role.Role.Name, newRole.ID)

		_, err = r.dependencies.DB.Update("roles").
			Set("chat_id", newRole.ID).
			Where(sq.Eq{"name": role.Role.Name}).
			Query()
		if err != nil {
			r.dependencies.Logger.Errorf("Error updating role id in db: %s", err)
			return err
		}

		role.Role.ID = newRole.ID
	}

	_, err = r.dependencies.Session.GuildRoleEdit(role.GuildID, role.Role.ID, role.Role.Name, role.Role.Color, role.Role.Hoist,
		role.Role.Permissions, role.Role.Mentionable)
	if err != nil {
		r.dependencies.Logger.Errorf("Error editing role: %s", err)
		return err
	}

	r.dependencies.Logger.Infof("Upserted '%s' to discord", role.Role.Name)

	return nil
}

// Only return an error if we want to keep the message and try again.
func (r Role) delete(role payloads.RolePayload) error {
	// Only one thing should write to discord at a time
	r.dependencies.Logger.Info("role.delete() acquiring lock")
	r.dependencies.Session.Lock()
	defer func() {
		r.dependencies.Session.Unlock()
		r.dependencies.Logger.Info("role.delete() released lock")
	}()

	err := r.dependencies.Session.GuildRoleDelete(role.GuildID, role.Role.ID)
	if err != nil {
		if err.(*discordgo.RESTError).Response.StatusCode == 404 {
			r.dependencies.Logger.Warnf("Role doesn't exist in discord: %s", role.Role.ID)
			return nil
		}
		r.dependencies.Logger.Errorf("Error deleting role: %s", err)
		return err
	}

	r.dependencies.Logger.Infof("Deleted '%s' from discord", role.Role.Name)

	return nil
}

// Maybe ditch this in favor of just trying to create and if that fails update. Maybe.
func (r Role) exists(name, guildID string) bool {
	roles, err := r.dependencies.Session.GuildRoles(guildID)
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
