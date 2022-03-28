package roles

import (
	"context"
	"encoding/json"

	sq "github.com/Masterminds/squirrel"
	sl "github.com/bhechinger/spiffylogger"
	"github.com/bwmarrin/discordgo"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// var matchDiscordError = regexp.MustCompile(`^The role '.*' already exists$`)

type Role struct {
	dependencies common.Dependencies
	ctx          context.Context
}

func New(ctx context.Context, deps common.Dependencies) *Role {
	return &Role{
		dependencies: deps,
		ctx:          ctx,
	}
}

func (r Role) HandleMessage(deliveries <-chan amqp.Delivery, done chan error) {
	ctx, sp := sl.OpenSpan(r.ctx)
	defer sp.Close()

	sp.With(zap.String("queue", "role"))

	sp.Info("Started roles message handling")
	defer sp.Info("Completed roles message handling")

	for d := range deliveries {
		func() {
			if len(d.Body) == 0 {
				sp.Error("Message has zero length body", zap.Any("delivery", d))

				err := d.Reject(false)
				if err != nil {
					sp.Error("Error ACKing message", zap.Error(err))
				}

				return
			}

			var body payloads.RolePayload
			err := json.Unmarshal(d.Body, &body)
			if err != nil {
				sp.Error("error unmarshalling payload", zap.Error(err))

				err = d.Reject(false)
				if err != nil {
					sp.Error("Error ACKing message", zap.Error(err))
				}

				return
			}

			ctx, sp := sl.OpenCorrelatedSpan(ctx, body.CorrelationID)
			defer sp.Close()

			if common.IgnoreRole(body.Role.Name) {
				sp.Info("Ignoring request for role", zap.String("role", body.Role.Name))

				err = d.Reject(false)
				if err != nil {
					sp.Error("Error ACKing message", zap.Error(err))
				}

				return
			}

			sp.Debug("Handling message", zap.Any("payload", body))

			switch body.Action {
			case payloads.Upsert:
				err = r.upsert(ctx, body)
			case payloads.Delete:
				err = r.delete(ctx, body)
			default:
				sp.Error("Unknown action", zap.Any("action", body.Action))
			}

			if err != nil {
				// we want to retry the message
				err = d.Reject(true)
				if err != nil {
					sp.Error("Error NACKing message", zap.Error(err))
				}

				return
			}

			err = d.Ack(false)
			if err != nil {
				sp.Error("Error ACKing message", zap.Error(err))
			}

			return
		}()
	}

	sp.Info("roles/HandleMessage: deliveries channel closed")
	done <- nil
}

// Only return an error if we want to keep the message and try again.
func (r Role) upsert(ctx context.Context, role payloads.RolePayload) error {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(zap.String("queue", "role"))

	var err error
	var sync bool

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Only one thing should write to discord at a time
	r.dependencies.Session.Lock()
	defer func() {
		r.dependencies.Session.Unlock()
	}()

	query := r.dependencies.DB.Select("sync").
		From("roles").
		Where(sq.Eq{"name": role.Role.Name})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	err = query.Scan(&sync)
	if err != nil {
		sp.Error("Error getting role sync status",
			zap.Error(err), zap.String("role", role.Role.Name))
		return err
	}

	// If this role isn't set to sync, ignore it.
	if !sync {
		return nil
	}

	// Check and see if this role has been created in discord or not
	if r.exists(ctx, role.Role.Name, role.GuildID) {
		// Update an existing role
		if role.GuildID == "" || role.Role.ID == "" || role.Role.Name == "" {
			sp.Error("role.update() missing data",
				zap.String("guild id", role.GuildID),
				zap.String("role id", role.Role.ID),
				zap.String("role", role.Role.Name))
			return nil
		}

		sp.Info("Updated role",
			zap.String("name", role.Role.Name), zap.String("id", role.Role.ID))
	} else {
		// Create a new role
		newRole, err := r.dependencies.Session.GuildRoleCreate(role.GuildID)
		if err != nil {
			sp.Error("Error creating role", zap.Error(err))
			return err
		}

		sp.Info("Create role",
			zap.String("guild id", role.GuildID),
			zap.String("role id", newRole.ID),
			zap.String("role", role.Role.Name))

		update := r.dependencies.DB.Update("roles").
			Set("chat_id", newRole.ID).
			Where(sq.Eq{"name": role.Role.Name})

		sqlStr, args, err = update.ToSql()
		if err != nil {
			sp.Error("error getting sql", zap.Error(err))
		} else {
			sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
		}

		_, err = update.QueryContext(ctx)
		if err != nil {
			sp.Error("Error updating role id in db",
				zap.Error(err), zap.String("role", role.Role.Name))
			return err
		}

		role.Role.ID = newRole.ID
	}

	_, err = r.dependencies.Session.GuildRoleEdit(role.GuildID, role.Role.ID, role.Role.Name, role.Role.Color, role.Role.Hoist,
		role.Role.Permissions, role.Role.Mentionable)
	if err != nil {
		sp.Error("Error editing role",
			zap.String("name", role.Role.Name),
			zap.String("id", role.Role.ID),
			zap.Error(err))
		return err
	}

	sp.Info("Upserted role to discord", zap.String("name", role.Role.Name))

	return nil
}

// Only return an error if we want to keep the message and try again.
func (r Role) delete(ctx context.Context, role payloads.RolePayload) error {
	_, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(zap.String("queue", "role"))

	// Only one thing should write to discord at a time
	r.dependencies.Session.Lock()
	defer func() {
		r.dependencies.Session.Unlock()
	}()

	err := r.dependencies.Session.GuildRoleDelete(role.GuildID, role.Role.ID)
	if err != nil {
		if err.(*discordgo.RESTError).Response.StatusCode == 404 {
			sp.Warn("Role doesn't exist in discord", zap.String("role", role.Role.ID))
			return nil
		}
		sp.Error("Error deleting role",
			zap.String("name", role.Role.Name),
			zap.String("id", role.Role.ID),
			zap.Error(err))
		return err
	}

	sp.Info("Deleted role from discord", zap.String("role", role.Role.Name))

	return nil
}

// Maybe ditch this in favor of just trying to create and if that fails update. Maybe.
func (r Role) exists(ctx context.Context, name, guildID string) bool {
	_, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(zap.String("queue", "role"))

	roles, err := r.dependencies.Session.GuildRoles(guildID)
	if err != nil {
		sp.Error("Error fetching discord roles", zap.Error(err))
		return true
	}

	for _, role := range roles {
		if name == role.Name {
			return true
		}
	}

	return false
}
