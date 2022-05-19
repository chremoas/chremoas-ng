package storage

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	sl "github.com/bhechinger/spiffylogger"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	"github.com/lib/pq"
	"go.uber.org/zap"
)

func (s Storage) GetPermission(ctx context.Context, name string) (payloads.Permission, error) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	query := s.DB.Select("id").
		From("permissions").
		Where(sq.Eq{"name": name})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return payloads.Permission{}, err
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	var permission payloads.Permission

	err = query.Scan(&permission.ID)
	if err != nil {
		sp.Error("error scanning permissionID", zap.Error(err))
		return payloads.Permission{}, err
	}

	return permission, nil
}

func (s Storage) GetPermissions(ctx context.Context) ([]payloads.Permission, error) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	query := s.DB.Select("name", "description").
		From("permissions")

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return nil, err
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	rows, err := query.QueryContext(ctx)
	if err != nil {
		sp.Error("error getting permissions", zap.Error(err))
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			sp.Error("error closing database", zap.Error(err))
		}
	}()

	var permissions []payloads.Permission
	for rows.Next() {
		var permission payloads.Permission

		err = rows.Scan(&permission.Name, &permission.Description)
		if err != nil {
			sp.Error("error scanning permissions", zap.Error(err))
			return nil, err
		}

		permissions = append(permissions, permission)
	}

	return permissions, nil
}

func (s Storage) InsertPermission(ctx context.Context, name, description string) error {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	query := s.DB.Insert("permissions").
		Columns("name", "description").
		Values(name, description)

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return err
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	rows, err := query.QueryContext(ctx)
	if err != nil {
		// I don't love this, but I can't find a better way right now
		if err.(*pq.Error).Code == "23505" {
			sp.Error("permission already exists")
			return err
		}
		sp.Error("error inserting permissions", zap.Error(err))
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

func (s Storage) DeletePermission(ctx context.Context, name string) error {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	query := s.DB.Delete("permissions").
		Where(sq.Eq{"name": name})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return err
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	rows, err := query.QueryContext(ctx)
	if err != nil {
		sp.Error("error deleting permissions", zap.Error(err))
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

func (s Storage) ListPermissionMembers(ctx context.Context, name string) ([]int, error) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := s.DB.Select("user_id").
		From("permission_membership").
		Join("permissions ON permission_membership.permission = permissions.id").
		Where(sq.Eq{"permissions.name": name})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return nil, err
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	rows, err := query.QueryContext(ctx)
	if err != nil {
		sp.Error("error getting permissions membership list", zap.Error(err))
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			sp.Error("error closing database", zap.Error(err))
		}
	}()

	var userIDs []int
	for rows.Next() {
		var userID int

		err = rows.Scan(&userID)
		if err != nil {
			sp.Error("error scanning permission_membership userID", zap.Error(err))
			return nil, err
		}

		userIDs = append(userIDs, userID)
	}

	return userIDs, nil
}

func (s Storage) InsertPermissionMembership(ctx context.Context, permissionID int, userID string) error {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := s.DB.Insert("permission_membership").
		Columns("permission", "user_id").
		Values(permissionID, userID)

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return err
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	rows, err := query.QueryContext(ctx)
	if err != nil {
		// I don't love this, but I can't find a better way right now
		if err.(*pq.Error).Code == "23505" {
			sp.Error("user already a member of permission")
			return err
		}
		sp.Error("error inserting permission", zap.Error(err))
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

func (s Storage) DeletePermissionMembership(ctx context.Context, permissionID int, userID string) error {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := s.DB.Delete("permission_membership").
		Where(sq.Eq{"permission": permissionID}).
		Where(sq.Eq{"user_id": userID})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return err
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	rows, err := query.QueryContext(ctx)
	if err != nil {
		sp.Error("error deleting permission", zap.Error(err))
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

func (s Storage) GetUserPermissions(ctx context.Context, userID string) ([]payloads.Permission, error) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := s.DB.Select("name").
		From("permissions").
		Join("permission_membership ON permission_membership.permission = permissions.id").
		Where(sq.Eq{"permission_membership.user_id": userID})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return nil, err
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	rows, err := query.QueryContext(ctx)
	if err != nil {
		sp.Error("error getting user perms", zap.Error(err))
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			sp.Error("error closing database", zap.Error(err))
		}
	}()

	var permissions []payloads.Permission

	for rows.Next() {
		var permission payloads.Permission

		err = rows.Scan(&permission.Name)
		if err != nil {
			sp.Error("Error scanning permission id", zap.Error(err))
			return nil, err
		}

		permissions = append(permissions, permission)
	}

	return permissions, nil
}

func (s Storage) GetPermissionCount(ctx context.Context, authorID string, permissionID int) (int, error) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	query := s.DB.Select("COUNT(*)").
		From("permission_membership").
		Where(sq.Eq{"user_id": authorID}).
		Where(sq.Eq{"permission": permissionID})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return -1, err
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	var count int

	err = query.Scan(&count)
	if err != nil {
		sp.Error("error scanning permission count", zap.Error(err))
		return -1, err
	}

	return count, nil
}
