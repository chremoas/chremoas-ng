package storage

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	sl "github.com/bhechinger/spiffylogger"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	"go.uber.org/zap"
)

func (s Storage) GetRoleFilters(ctx context.Context, sig bool, name string) ([]payloads.RoleFilter, error) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := s.DB.Select("role_filters.filter").
		From("role_filters").
		InnerJoin("roles ON role_filters.role = roles.id").
		Where(sq.Eq{"sig": sig}).
		Where(sq.Eq{"role_nick": name})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return nil, err
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	rows, err := query.QueryContext(ctx)
	if err != nil {
		sp.Error("error getting role filters", zap.Error(err))
		return nil, err
	}
	defer func() {
		if err = rows.Close(); err != nil {
			sp.Error("error closing row", zap.Error(err))
		}
	}()

	var roleFilters []payloads.RoleFilter

	for rows.Next() {
		var roleFilter payloads.RoleFilter

		err = rows.Scan(&roleFilter.Filter)
		if err != nil {
			sp.Error("error scanning role's id", zap.Error(err))
			return nil, err
		}

		roleFilters = append(roleFilters, roleFilter)
	}

	return roleFilters, nil
}

func (s Storage) DeleteRoleFilter(ctx context.Context, filterID int) error {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := s.DB.Delete("role_filters").
		Where(sq.Eq{"filter": filterID})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return err
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	_, err = query.QueryContext(ctx)
	if err != nil {
		sp.Error("error deleting filter membership", zap.Error(err))
		return err
	}

	return nil
}

func (s Storage) InsertRoleFilter(ctx context.Context, roleID, filterID int) error {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := s.DB.Insert("role_filters").
		Columns("role", "filter").
		Values(roleID, filterID)

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return err
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	rows, err := query.QueryContext(ctx)
	if err != nil {
		sp.Error("error adding role_filter", zap.Error(err))
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
