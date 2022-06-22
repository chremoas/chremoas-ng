package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	sq "github.com/Masterminds/squirrel"
	sl "github.com/bhechinger/spiffylogger"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	"github.com/lib/pq"
	"go.uber.org/zap"
)

const (
	Role = false
	Sig  = true
)

var RoleType = map[bool]string{Role: "role", Sig: "sig"}
var ErrNoRole = errors.New("no such role")
var ErrRoleExists = errors.New("role already exists")

func (s Storage) GetRoleCount(ctx context.Context, sig bool, ticker string) (int, error) {
	ctx, sp := sl.OpenCorrelatedSpan(ctx, sl.NewID())
	defer sp.Close()

	var count int

	query := s.DB.Select("count(id)").
		From("roles").
		Where(sq.Eq{"role_nick": ticker}).
		Where(sq.Eq{"sig": sig})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return -1, err
	} else {
		sp.With(
			zap.String("query", sqlStr),
			zap.Any("args", args),
		)
		sp.Debug("GetRoleCount(): sql query")
	}

	err = query.Scan(&count)
	if err != nil {
		sp.Error(
			"error getting count of alliances by name",
			zap.Error(err),
			zap.String("ticker", ticker),
		)
		return -1, err
	}

	return count, nil
}

func (s Storage) GetRole(ctx context.Context, name, ticker string, sig *bool) (payloads.Role, error) {
	ctx, sp := sl.OpenCorrelatedSpan(ctx, sl.NewID())
	defer sp.Close()

	query := s.DB.Select("sync", "chat_id").
		From("roles")

	if name != "" {
		query = query.Where(sq.Eq{"name": name})
	}

	if ticker != "" {
		query = query.Where(sq.Eq{"role_nick": ticker})
	}

	if sig != nil {
		query = query.Where(sq.Eq{"sig": sig})
	}

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return payloads.Role{}, err
	} else {
		sp.With(
			zap.String("query", sqlStr),
			zap.Any("args", args),
		)
		sp.Debug("GetRole(): sql query")
	}

	var role payloads.Role

	err = query.Scan(&role.Sync, &role.ChatID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return payloads.Role{}, ErrNoRole
		}

		sp.Error("Error getting role", zap.Error(err))
		return payloads.Role{}, err
	}

	return role, nil
}

func (s Storage) GetRoleByChatID(ctx context.Context, chatID string) (payloads.Role, error) {
	ctx, sp := sl.OpenCorrelatedSpan(ctx, sl.NewID())
	defer sp.Close()

	query := s.DB.Select("sync", "chat_id").
		From("roles").
		Where(sq.Eq{"chat_id": chatID})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return payloads.Role{}, err
	} else {
		sp.With(
			zap.String("query", sqlStr),
			zap.Any("args", args),
		)
		sp.Debug("GetRoleByChatID(): sql query")
	}

	var role payloads.Role

	err = query.Scan(&role.Sync, &role.ChatID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return payloads.Role{}, ErrRoleExists
		}

		sp.Error("Error getting role", zap.Error(err))
		return payloads.Role{}, err
	}

	return role, nil
}

func (s Storage) GetRoleByType(ctx context.Context, sig bool, shortName string) (payloads.Role, error) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.Bool("sig", sig),
		zap.String("shortName", shortName),
	)

	roles, err := s.doGetRoles(ctx, sig, &shortName)
	if err != nil {
		sp.Error("Error getting role by type", zap.Error(err))
		return payloads.Role{}, err
	}

	if len(roles) > 1 {
		sp.Error("Somehow we got more than one role back. That makes no sense.", zap.Int("roles", len(roles)))
		return payloads.Role{}, fmt.Errorf("more than one role returned when only one expected")
	}

	if len(roles) == 0 {
		sp.Error("no such role", zap.Error(err))
		return payloads.Role{}, fmt.Errorf("no such role: %s", shortName)
	}

	return roles[0], nil
}

func (s Storage) GetRolesByType(ctx context.Context, sig bool) ([]payloads.Role, error) {
	return s.doGetRoles(ctx, sig, nil)

}

func (s Storage) doGetRoles(ctx context.Context, sig bool, shortName *string) ([]payloads.Role, error) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := s.DB.Select(
		"color",
		"hoist",
		"joinable",
		"managed",
		"mentionable",
		"name",
		"permissions",
		"position",
		"role_nick",
		"sig",
		"sync",
	).
		Where(sq.Eq{"sig": sig}).
		From("roles")

	if shortName != nil {
		query = query.Where(sq.Eq{"role_nick": shortName})
	}

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return nil, err
	} else {
		sp.With(
			zap.String("query", sqlStr),
			zap.Any("args", args),
		)
		sp.Debug("doGetRoles(): sql query")
	}

	rows, err := query.QueryContext(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoRole
		}

		sp.Error("error getting role", zap.Error(err))
		return nil, err
	}
	defer func() {
		if err = rows.Close(); err != nil {
			sp.Error("error closing row", zap.Error(err))
		}
	}()

	var roles []payloads.Role

	for rows.Next() {
		var role payloads.Role

		err = rows.Scan(
			&role.Color,
			&role.Hoist,
			&role.Joinable,
			&role.Managed,
			&role.Mentionable,
			&role.Name,
			&role.Permissions,
			&role.Position,
			&role.ShortName,
			&role.Sig,
			&role.Sync,
		)
		if err != nil {
			sp.Error("error scanning role", zap.Error(err))
			return nil, err
		}
		roles = append(roles, role)
	}

	return roles, nil
}

func (s Storage) GetRolesBySync(ctx context.Context, syncOnly bool) ([]payloads.Role, error) {
	ctx, sp := sl.OpenCorrelatedSpan(ctx, sl.NewID())
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := s.DB.Select("chat_id", "name", "managed", "mentionable", "hoist", "color", "position", "permissions").
		From("roles")

	if syncOnly {
		query = query.Where(sq.Eq{"sync": "true"})
	}

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return nil, err
	} else {
		sp.With(
			zap.String("query", sqlStr),
			zap.Any("args", args),
		)
		sp.Debug("GetRolesBySync(): sql query")
	}

	rows, err := query.QueryContext(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoRole
		}

		sp.Error("error getting role", zap.Error(err))
		return nil, err
	}
	defer func() {
		if err = rows.Close(); err != nil {
			sp.Error("error closing row", zap.Error(err))
		}
	}()

	var roles []payloads.Role

	for rows.Next() {
		var role payloads.Role
		err = rows.Scan(
			&role.ID,
			&role.Name,
			&role.Managed,
			&role.Mentionable,
			&role.Hoist,
			&role.Color,
			&role.Position,
			&role.Permissions,
		)
		if err != nil {
			sp.Error("error scanning role fields", zap.Error(err))
			continue
		}

		roles = append(roles, role)
	}

	return roles, nil
}

func (s Storage) UpdateRole(ctx context.Context, chatID, name, id string) error {
	ctx, sp := sl.OpenCorrelatedSpan(ctx, sl.NewID())
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := s.DB.Update("roles")

	if chatID == "" && id == "" {
		sp.Error("chatID or id need to be set")
		return fmt.Errorf("chatID or id need to be set")
	}

	if chatID != "" {
		query = query.Set("chat_id", chatID)
	}

	if id != "" {
		query = query.Set("id", id)
	}

	query = query.Where(sq.Eq{"name": name})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return err
	} else {
		sp.With(
			zap.String("query", sqlStr),
			zap.Any("args", args),
		)
		sp.Debug("UpdateRole(): sql query")
	}

	_, err = query.QueryContext(ctx)
	if err != nil {
		sp.Error("Error updating role id in db", zap.Error(err))
		return err
	}

	return nil
}

// UpdateRoleValues is just a quick fix until I find a better way to merge these two
func (s Storage) UpdateRoleValues(ctx context.Context, sig bool, name string, values map[string]string) error {
	ctx, sp := sl.OpenCorrelatedSpan(ctx, sl.NewID())
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := s.DB.Update("roles")

	for k, v := range values {
		key := strings.ToLower(k)
		if key == "color" {
			if string(v[0]) == "#" {
				i, _ := strconv.ParseInt(v[1:], 16, 64)
				v = strconv.Itoa(int(i))
			}
		}

		if key == "sync" {
			sync, err := strconv.ParseBool(v)
			if err != nil {
				sp.Warn("error updating sync", zap.Error(err))
				return err
			}
			sp.With(zap.Bool("sync", sync))
		}

		query = query.Set(key, v)
	}

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return err
	} else {
		sp.With(
			zap.String("query", sqlStr),
			zap.Any("args", args),
		)
		sp.Debug("UpdateRoleValues(): sql query")
	}

	_, err = query.Where(sq.Eq{"name": name}).
		Where(sq.Eq{"sig": sig}).
		QueryContext(ctx)
	if err != nil {
		sp.Error("error adding role", zap.Error(err))
		return err
	}

	return nil
}

func (s Storage) GetMemberRoles(ctx context.Context, userID string, sig bool) ([]payloads.Role, error) {
	ctx, sp := sl.OpenCorrelatedSpan(ctx, sl.NewID())
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := s.DB.Select("role_nick", "name", "chat_id").
		From("").
		Suffix("getMemberRoles(?, ?)", userID, strconv.FormatBool(sig))

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return nil, err
	} else {
		sp.With(
			zap.String("query", sqlStr),
			zap.Any("args", args),
		)
		sp.Debug("GetMemberRoles(): sql query")
	}

	rows, err := query.QueryContext(ctx)
	if err != nil {
		sp.Error("error getting role membership", zap.Error(err))
		return nil, fmt.Errorf("error getting user %ss (%s): %s", RoleType[sig], userID, err)
	}

	defer func() {
		if err = rows.Close(); err != nil {
			sp.Error("error closing row", zap.Error(err))
		}
	}()

	var roles []payloads.Role

	for rows.Next() {
		var role payloads.Role

		err = rows.Scan(
			&role.ShortName,
			&role.Name,
			&role.ChatID,
		)
		if err != nil {
			sp.Error("error scanning row", zap.Error(err))
			return nil, fmt.Errorf("error scanning %s row: %s", RoleType[sig], err)
		}

		roles = append(roles, role)
	}

	return roles, nil
}

// InsertRole creates a new role in the database. sync is set to sig which causes sigs to be synced by default
// and roles to not be synced by default.
func (s Storage) InsertRole(ctx context.Context, name, ticker, chatType string, sig, joinable bool) (int, error) {
	ctx, sp := sl.OpenCorrelatedSpan(ctx, sl.NewID())
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := s.DB.Insert("roles").
		Columns("sig", "joinable", "name", "role_nick", "chat_type", "sync").
		// a sig is sync-ed by default, so we overload the sig bool because it does the right thing here.
		Values(sig, joinable, name, ticker, chatType, sig).
		Suffix("RETURNING \"id\"")

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return -1, err
	} else {
		sp.With(
			zap.String("query", sqlStr),
			zap.Any("args", args),
		)
		sp.Debug("InsertRole(): sql query")
	}

	var roleID int
	err = query.Scan(&roleID)
	if err != nil {
		// I don't love this, but I can't find a better way right now
		if err.(*pq.Error).Code == "23505" {
			return -1, ErrRoleExists
		}
		sp.Error("error adding role", zap.Error(err))
		return -1, err
	}

	return roleID, nil
}

func (s Storage) DeleteRole(ctx context.Context, ticker string, sig bool) error {
	ctx, sp := sl.OpenCorrelatedSpan(ctx, sl.NewID())
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	roleFilters, err := s.GetRoleFilters(ctx, sig, ticker)
	if err != nil {
		sp.Error("Error getting role filters", zap.Error(err))
		return err
	}

	for r := range roleFilters {
		err = s.DeleteFilterMembership(ctx, roleFilters[r].ID, "")
		if err != nil {
			sp.Error("Error deleting filter memberships for role", zap.Error(err))
		}

		err = s.DeleteFilterByID(ctx, roleFilters[r].ID)
		if err != nil {
			sp.Error("Error deleting filter by ID", zap.Error(err))
		}

		err = s.DeleteRoleFilter(ctx, roleFilters[r].ID)
		if err != nil {
			sp.Error("Error deleting role filter", zap.Error(err))
		}
	}

	query := s.DB.Delete("roles").
		Where(sq.Eq{"role_nick": ticker}).
		Where(sq.Eq{"sig": sig})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return err
	} else {
		sp.With(
			zap.String("query", sqlStr),
			zap.Any("args", args),
		)
		sp.Debug("DeleteRoles(): sql query")
	}

	rows, err := query.QueryContext(ctx)
	if err != nil {
		sp.Error("error deleting role", zap.Error(err))
		return err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			sp.Error("error closing database", zap.Error(err))
		}
	}()

	return nil
}
