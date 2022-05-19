package storage

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	sl "github.com/bhechinger/spiffylogger"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	"go.uber.org/zap"
)

func (s Storage) GetAllianceCount(ctx context.Context, allianceID int32) (int, error) {
	ctx, sp := sl.OpenCorrelatedSpan(ctx, sl.NewID())
	defer sp.Close()

	query := s.DB.Select("COUNT(*)").
		From("alliances").
		Where(sq.Eq{"id": allianceID})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return -1, err
	} else {
		sp.With(
			zap.String("query", sqlStr),
			zap.Any("args", args),
		)
		sp.Debug("sql query")
	}

	var count int

	err = query.Scan(&count)
	if err != nil {
		sp.Error("error getting alliance count", zap.Error(err))
		return -1, err
	}

	return count, nil
}

func (s Storage) GetAlliance(ctx context.Context, allianceID int32) (payloads.Alliance, error) {
	ctx, sp := sl.OpenCorrelatedSpan(ctx, sl.NewID())
	defer sp.Close()

	query := s.DB.Select("name", "ticker").
		From("alliances").
		Where(sq.Eq{"id": allianceID})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return payloads.Alliance{}, err
	} else {
		sp.With(
			zap.String("query", sqlStr),
			zap.Any("args", args),
		)
		sp.Debug("sql query")
	}

	alliance := payloads.Alliance{ID: allianceID}

	err = query.Scan(&alliance.Name, &alliance.Ticker)
	if err != nil {
		sp.Error("error getting alliance", zap.Error(err))
		return payloads.Alliance{}, err
	}

	return alliance, nil
}

func (s Storage) GetAlliances(ctx context.Context) ([]payloads.Alliance, error) {
	ctx, sp := sl.OpenCorrelatedSpan(ctx, sl.NewID())
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := s.DB.Select("id", "name", "ticker").
		From("alliances")

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return nil, err
	} else {
		sp.With(
			zap.String("query", sqlStr),
			zap.Any("args", args),
		)
		sp.Debug("sql query")
	}

	rows, err := query.QueryContext(ctx)
	if err != nil {
		sp.Error("error getting alliance list from db", zap.Error(err))
		return nil, err
	}

	defer func() {
		if err = rows.Close(); err != nil {
			sp.Error("error closing row", zap.Error(err))
		}
	}()

	var alliances []payloads.Alliance

	for rows.Next() {
		var alliance payloads.Alliance

		err = rows.Scan(&alliance.ID, &alliance.Name, &alliance.Ticker)
		if err != nil {
			sp.Error("error scanning alliance values", zap.Error(err))
			continue
		}

		alliances = append(alliances, alliance)
	}

	return alliances, nil
}

func (s Storage) UpsertAlliance(ctx context.Context, allianceID int32, name, ticker string) error {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	insert := s.DB.Insert("alliances").
		Columns("id", "name", "ticker").
		Values(allianceID, name, ticker).
		Suffix("ON CONFLICT (id) DO UPDATE SET name=?, ticker=?", name, ticker)

	sqlStr, args, err := insert.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return err
	} else {
		sp.With(
			zap.String("query", sqlStr),
			zap.Any("args", args),
		)
		sp.Debug("sql query")
	}

	rows, err := insert.QueryContext(ctx)
	if err != nil {
		sp.Error("Error updating alliance", zap.Error(err))
		return err
	}

	defer func() {
		if rows == nil {
			return
		}

		if err = rows.Close(); err != nil {
			sp.Error("error closing row", zap.Error(err))
		}
	}()

	return nil
}
