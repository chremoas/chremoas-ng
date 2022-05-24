package storage

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	sl "github.com/bhechinger/spiffylogger"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	"github.com/lib/pq"
	"go.uber.org/zap"
)

func (s Storage) GetFilter(ctx context.Context, name string) (payloads.Filter, error) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := s.DB.Select("id", "name", "description").
		From("filters").
		Where(sq.Eq{"name": name})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return payloads.Filter{}, err
	} else {
		sp.With(
			zap.String("query", sqlStr),
			zap.Any("args", args),
		)
		sp.Debug("GetFilter(): sql query")
	}

	var filter payloads.Filter

	err = query.Scan(&filter.ID, &filter.Name, &filter.Description)
	if err != nil {
		if err == sql.ErrNoRows {
			return payloads.Filter{}, fmt.Errorf("no such filter: %s", name)
		}
		sp.Error(
			"error getting corporation info",
			zap.Error(err),
		)
		return payloads.Filter{}, err
	}

	return filter, nil
}

func (s Storage) GetFilters(ctx context.Context) ([]payloads.Filter, error) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := s.DB.Select("name", "description").
		From("filters")

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return nil, err
	} else {
		sp.With(
			zap.String("query", sqlStr),
			zap.Any("args", args),
		)
		sp.Debug("GetFilters(): sql query")
	}

	rows, err := query.QueryContext(ctx)
	if err != nil {
		sp.Error("error getting filter", zap.Error(err))
		return nil, err
	}

	defer func() {
		err := rows.Close()
		if err != nil {
			sp.Error("error closing database", zap.Error(err))
		}
	}()

	var filters []payloads.Filter

	for rows.Next() {
		var filter payloads.Filter

		err = rows.Scan(&filter.Name, &filter.Description)
		if err != nil {
			sp.Error("error scanning filter row", zap.Error(err))
			return nil, err
		}

		filters = append(filters, filter)
	}

	return filters, nil
}

func (s Storage) GetTickerFilters(ctx context.Context, sig bool, ticker string) ([]payloads.Filter, error) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := s.DB.Select("filters.name").
		From("filters").
		Join("role_filters ON role_filters.filter = filters.id").
		Join("roles ON roles.id = role_filters.role").
		Where(sq.Eq{"roles.role_nick": ticker}).
		Where(sq.Eq{"roles.sig": sig})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return nil, err
	} else {
		sp.With(
			zap.String("query", sqlStr),
			zap.Any("args", args),
		)
		sp.Debug("GetTickerFilters(): sql query")
	}

	rows, err := query.QueryContext(ctx)
	if err != nil {
		sp.Error("error fetching filters", zap.Error(err))
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			sp.Error("error closing database", zap.Error(err))
		}
	}()

	var filters []payloads.Filter

	for rows.Next() {
		var filter payloads.Filter

		err = rows.Scan(&filter.Name)
		if err != nil {
			sp.Error("error scanning filters", zap.Error(err))
			return nil, err
		}

		filters = append(filters, filter)
	}

	return filters, nil
}

func (s Storage) InsertFilter(ctx context.Context, name, description string) (int, error) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	query := s.DB.Insert("filters").
		Columns("name", "description").
		Values(name, description).
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
		sp.Debug("InsertFilter(): sql query")
	}

	var id int
	err = query.Scan(&id)
	if err != nil {
		// I don't love this, but I can't find a better way right now
		if err.(*pq.Error).Code == "23505" {
			return -1, fmt.Errorf("filter `%s` already exists", name)
		}

		sp.Error("error inserting filter", zap.Error(err))
		return -1, err
	}

	return id, nil
}

func (s Storage) DeleteFilter(ctx context.Context, name string) error {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	filter, err := s.GetFilter(ctx, name)
	if err != nil {
		sp.Error("Error getting filter")
		return err
	}

	err = s.DeleteFilterMembership(ctx, filter.ID, "")
	if err != nil {
		sp.Error("Error deleting filter membership", zap.Error(err))
		return err
	}

	err = s.DeleteRoleFilter(ctx, filter.ID)
	if err != nil {
		sp.Error("Error deleting role filter", zap.Error(err))
		return err
	}

	query := s.DB.Delete("filters").
		Where(sq.Eq{"id": filter.ID})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return err
	} else {
		sp.With(
			zap.String("query", sqlStr),
			zap.Any("args", args),
		)
		sp.Debug("DeleteFilter(): sql query")
	}

	_, err = query.QueryContext(ctx)
	if err != nil {
		sp.Error("error deleting filter", zap.Error(err))
		return err
	}

	sp.Info("deleted filter", zap.Any("filter", filter))

	return nil
}

func (s Storage) DeleteFilterByID(ctx context.Context, id int) error {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	err := s.DeleteFilterMembership(ctx, id, "")
	if err != nil {
		sp.Error("Error deleting filter membership", zap.Error(err))
		return err
	}

	err = s.DeleteRoleFilter(ctx, id)
	if err != nil {
		sp.Error("Error deleting role filter", zap.Error(err))
		return err
	}

	query := s.DB.Delete("filters").
		Where(sq.Eq{"id": id})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return err
	} else {
		sp.With(
			zap.String("query", sqlStr),
			zap.Any("args", args),
		)
		sp.Debug("DeleteFilterByID(): sql query")
	}

	_, err = query.QueryContext(ctx)
	if err != nil {
		sp.Error("error deleting filter", zap.Error(err))
		return err
	}

	sp.Info("deleted filter by id", zap.Any("id", id))

	return nil
}

func (s Storage) DeleteFilterMembership(ctx context.Context, filterID int, userID string) error {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := s.DB.Delete("filter_membership").
		Where(sq.Eq{"filter": filterID})

	if userID != "" {
		query = query.Where(sq.Eq{"user_id": userID})
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
		sp.Debug("DeleteFilterMembership(): sql query")
	}

	_, err = query.QueryContext(ctx)
	if err != nil {
		sp.Error("error deleting filter membership", zap.Error(err))
		return err
	}

	return nil
}

func (s Storage) ListFilterMembers(ctx context.Context, filter string) ([]int64, error) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := s.DB.Select("user_id").
		From("filters").
		Join("filter_membership ON filters.id = filter_membership.filter").
		Where(sq.Eq{"filters.name": filter})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return nil, err
	} else {
		sp.With(
			zap.String("query", sqlStr),
			zap.Any("args", args),
		)
		sp.Debug("ListFilterMembers(): sql query")
	}

	rows, err := query.QueryContext(ctx)
	if err != nil {
		sp.Error("error getting filter membership list", zap.Error(err))
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			sp.Error("error closing database", zap.Error(err))
		}
	}()

	var userIDs []int64

	for rows.Next() {
		var userID int64
		err = rows.Scan(&userID)
		if err != nil {
			sp.Error("error scanning filter_membership userID", zap.Error(err))
			return nil, err
		}
		userIDs = append(userIDs, userID)
	}

	return userIDs, nil
}

func (s Storage) AddFilterMembership(ctx context.Context, filterID int, userID string) error {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := s.DB.Insert("filter_membership").
		Columns("filter", "user_id").
		Values(filterID, userID)

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return err
	} else {
		sp.With(
			zap.String("query", sqlStr),
			zap.Any("args", args),
		)
		sp.Debug("AddFilterMembership(): sql query")
	}

	rows, err := query.QueryContext(ctx)
	if err != nil {
		// I don't love this, but I can't find a better way right now
		if err.(*pq.Error).Code == "23505" {
			sp.Warn("already a member", zap.Bool("maybe", false))
			return err
		}
		sp.Error("error inserting filter", zap.Error(err))
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

// GetRoleMembers I need to re-evaluate this query. I don't really remember why it needed to be so complex.
func (s Storage) GetRoleMembers(ctx context.Context, filterList []int64) ([]int64, error) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := s.DB.Select("user_id").
		From("filter_membership").
		Where(sq.Eq{"filter": filterList}).
		GroupBy("user_id").
		Having("count(*) = ?", len(filterList))

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return nil, err
	} else {
		sp.With(
			zap.String("query", sqlStr),
			zap.Any("args", args),
		)
		sp.Debug("GetRoleMembers(): sql query")
	}

	rows, err := query.QueryContext(ctx)
	if err != nil {
		sp.Error("error getting filter membership", zap.Error(err))
		return nil, err
	}
	defer func() {
		if err = rows.Close(); err != nil {
			sp.Error("error closing row", zap.Error(err))
		}
	}()

	var members []int64

	for rows.Next() {
		var member int64
		err = rows.Scan(&member)
		if err != nil {
			sp.Error("error scanning filter's userID", zap.Error(err))
			return nil, err
		}

		members = append(members, member)
	}

	return members, nil
}
