package storage

import (
	"context"
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	sl "github.com/bhechinger/spiffylogger"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	"go.uber.org/zap"
)

func (s Storage) GetCorporationCount(ctx context.Context, corporationID int32) (int, error) {
	ctx, sp := sl.OpenCorrelatedSpan(ctx, sl.NewID())
	defer sp.Close()

	query := s.DB.Select("COUNT(*)").
		From("corporations").
		Where(sq.Eq{"id": corporationID})

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
		sp.Error("error getting corporation count", zap.Error(err))
		return -1, err
	}

	return count, nil
}

func (s Storage) GetCorporation(ctx context.Context, corporationID int32) (payloads.Corporation, error) {
	ctx, sp := sl.OpenCorrelatedSpan(ctx, sl.NewID())
	defer sp.Close()

	query := s.DB.Select("ticker", "alliance_id").
		From("corporations").
		Where(sq.Eq{"id": corporationID})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return payloads.Corporation{}, err
	} else {
		sp.With(
			zap.String("query", sqlStr),
			zap.Any("args", args),
		)
		sp.Debug("sql query")
	}

	var corporation payloads.Corporation

	err = query.Scan(&corporation.Ticker, &corporation.AllianceID)
	if err != nil {
		sp.Error(
			"error getting corporation info",
			zap.Error(err),
		)
		return payloads.Corporation{}, err
	}

	return corporation, nil
}

func (s Storage) GetCorporations(ctx context.Context) ([]payloads.Corporation, error) {
	ctx, sp := sl.OpenCorrelatedSpan(ctx, sl.NewID())
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := s.DB.Select("id", "name", "ticker", "alliance_id").
		From("corporations")

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
		sp.Error("error getting corporation list from db", zap.Error(err))
		return nil, err
	}

	defer func() {
		if err = rows.Close(); err != nil {
			sp.Error("error closing row", zap.Error(err))
		}
	}()

	var corporations []payloads.Corporation

	for rows.Next() {
		var corporation payloads.Corporation

		err = rows.Scan(&corporation.ID, &corporation.Name, &corporation.Ticker, &corporation.AllianceID)
		if err != nil {
			sp.Error("error scanning corporation values", zap.Error(err))
			continue
		}

		corporations = append(corporations, corporation)
	}

	return corporations, nil
}

func (s Storage) UpsertCorporation(ctx context.Context, corporationID, allianceID int32, name, ticker string) error {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var allianceNullableID sql.NullInt32

	if allianceID != 0 {
		allianceNullableID.Int32 = allianceID
		allianceNullableID.Valid = true
	}

	sp.With(
		zap.String("sub-component", "corporation"),
		zap.Int32("corporation_id", corporationID),
		zap.Any("alliance_id", allianceNullableID),
		zap.String("name", name),
		zap.String("ticker", ticker),
	)

	sp.Debug("Upserting corporation")

	insert := s.DB.Insert("corporations").
		Columns("id", "name", "ticker", "alliance_id").
		Values(corporationID, name, ticker, allianceNullableID).
		Suffix("ON CONFLICT (id) DO UPDATE SET name=?, ticker=?, alliance_id=?", name, ticker, allianceNullableID)

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
		sp.Error("Error upserting corporation %d alliance: %v: %s", zap.Error(err))
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
