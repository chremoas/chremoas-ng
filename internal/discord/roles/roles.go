package roles

import (
	"context"
	"encoding/json"

	sq "github.com/Masterminds/squirrel"
	"github.com/bwmarrin/discordgo"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
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
	logger := r.dependencies.Logger.With(zap.String("queue", "role"))

	logger.Info("Started roles message handling")
	defer logger.Info("Completed roles message handling")

	for d := range deliveries {
		if len(d.Body) == 0 {
			logger.Error("Message has zero length body", zap.Any("delivery", d))

			err := d.Reject(false)
			if err != nil {
				logger.Error("Error ACKing message", zap.Error(err))
			}

			continue
		}

		var body payloads.RolePayload
		err := json.Unmarshal(d.Body, &body)
		if err != nil {
			logger.Error("error unmarshalling payload", zap.Error(err))

			err = d.Reject(false)
			if err != nil {
				logger.Error("Error ACKing message", zap.Error(err))
			}

			continue
		}

		if common.IgnoreRole(body.Role.Name) {
			logger.Info("Ignoring request for role", zap.String("role", body.Role.Name))

			err = d.Reject(false)
			if err != nil {
				logger.Error("Error ACKing message", zap.Error(err))
			}

			continue
		}

		logger.Debug("Handling message", zap.Any("payload", body))

		switch body.Action {
		case payloads.Upsert:
			err = r.upsert(body)
		case payloads.Delete:
			err = r.delete(body)
		default:
			logger.Error("Unknown action", zap.Any("action", body.Action))
		}

		if err != nil {
			// we want to retry the message
			err = d.Reject(true)
			if err != nil {
				logger.Error("Error NACKing message", zap.Error(err))
			}

			continue
		}

		err = d.Ack(false)
		if err != nil {
			logger.Error("Error ACKing message", zap.Error(err))
		}
	}

	logger.Info("roles/HandleMessage: deliveries channel closed")
	done <- nil
}

// Only return an error if we want to keep the message and try again.
func (r Role) upsert(role payloads.RolePayload) error {
	logger := r.dependencies.Logger.With(zap.String("queue", "role"))

	var err error
	var sync bool

	ctx, cancel := context.WithCancel(r.dependencies.Context)
	defer cancel()

	// Only one thing should write to discord at a time
	r.dependencies.Session.Lock()
	defer func() {
		r.dependencies.Session.Unlock()
	}()

	err = r.dependencies.DB.Select("sync").
		From("roles").
		Where(sq.Eq{"name": role.Role.Name}).
		Scan(&sync)
	if err != nil {
		logger.Error("Error getting role sync status",
			zap.Error(err), zap.String("role", role.Role.Name))
		return err
	}

	// If this role isn't set to sync, ignore it.
	if !sync {
		return nil
	}

	// Check and see if this role has been created in discord or not
	if r.exists(role.Role.Name, role.GuildID) {
		// Update an existing role
		if role.GuildID == "" || role.Role.ID == "" || role.Role.Name == "" {
			logger.Error("role.update() missing data",
				zap.String("guild id", role.GuildID),
				zap.String("role id", role.Role.ID),
				zap.String("role", role.Role.Name))
			return nil
		}

		logger.Info("Updated role",
			zap.String("name", role.Role.Name), zap.String("id", role.Role.ID))
	} else {
		// Create a new role
		newRole, err := r.dependencies.Session.GuildRoleCreate(role.GuildID)
		if err != nil {
			logger.Error("Error creating role", zap.Error(err))
			return err
		}

		logger.Info("Create role",
			zap.String("guild id", role.GuildID),
			zap.String("role id", newRole.ID),
			zap.String("role", role.Role.Name))

		_, err = r.dependencies.DB.Update("roles").
			Set("chat_id", newRole.ID).
			Where(sq.Eq{"name": role.Role.Name}).
			QueryContext(ctx)
		if err != nil {
			logger.Error("Error updating role id in db",
				zap.Error(err), zap.String("role", role.Role.Name))
			return err
		}

		role.Role.ID = newRole.ID
	}

	_, err = r.dependencies.Session.GuildRoleEdit(role.GuildID, role.Role.ID, role.Role.Name, role.Role.Color, role.Role.Hoist,
		role.Role.Permissions, role.Role.Mentionable)
	if err != nil {
		logger.Error("Error editing role",
			zap.String("name", role.Role.Name),
			zap.String("id", role.Role.ID),
			zap.Error(err))
		return err
	}

	logger.Info("Upserted role to discord", zap.String("name", role.Role.Name))

	return nil
}

// Only return an error if we want to keep the message and try again.
func (r Role) delete(role payloads.RolePayload) error {
	logger := r.dependencies.Logger.With(zap.String("queue", "role"))

	// Only one thing should write to discord at a time
	r.dependencies.Session.Lock()
	defer func() {
		r.dependencies.Session.Unlock()
	}()

	err := r.dependencies.Session.GuildRoleDelete(role.GuildID, role.Role.ID)
	if err != nil {
		if err.(*discordgo.RESTError).Response.StatusCode == 404 {
			logger.Warn("Role doesn't exist in discord", zap.String("role", role.Role.ID))
			return nil
		}
		logger.Error("Error deleting role",
			zap.String("name", role.Role.Name),
			zap.String("id", role.Role.ID),
			zap.Error(err))
		return err
	}

	logger.Info("Deleted role from discord", zap.String("role", role.Role.Name))

	return nil
}

// Maybe ditch this in favor of just trying to create and if that fails update. Maybe.
func (r Role) exists(name, guildID string) bool {
	logger := r.dependencies.Logger.With(zap.String("queue", "role"))

	roles, err := r.dependencies.Session.GuildRoles(guildID)
	if err != nil {
		logger.Error("Error fetching discord roles", zap.Error(err))
		return true
	}

	for _, role := range roles {
		if name == role.Name {
			return true
		}
	}

	return false
}
