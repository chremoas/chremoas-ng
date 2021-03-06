package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	sl "github.com/bhechinger/spiffylogger"
	"go.uber.org/zap"
)

var ErrNoAuthCode = errors.New("no such auth code")

func (s Storage) GetAuthCode(ctx context.Context, authCode string) (int, bool, error) {
	ctx, sp := sl.OpenCorrelatedSpan(ctx, sl.NewID())
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := s.DB.Select("character_id", "used").
		From("authentication_codes").
		Where(sq.Eq{"code": authCode})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return -1, false, err
	} else {
		sp.With(
			zap.String("query", sqlStr),
			zap.Any("args", args),
		)
		sp.Debug("GetAuthCode(): sql query")
	}

	var (
		characterID int
		used        bool
	)
	err = query.Scan(&characterID, &used)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return -1, false, ErrNoAuthCode
		}

		sp.Error("error getting authentication code details", zap.Error(err))
		return -1, false, err
	}

	return characterID, used, nil
}

func (s Storage) DeleteAuthCodes(ctx context.Context, characterID int32) error {
	ctx, sp := sl.OpenCorrelatedSpan(ctx, sl.NewID())
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := s.DB.Delete("authentication_codes").
		Where(sq.Eq{"character_id": characterID})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return err
	} else {
		sp.With(
			zap.String("query", sqlStr),
			zap.Any("args", args),
		)
		sp.Debug("DeleteAuthCode(): sql query")
	}

	_, err = query.QueryContext(ctx)
	if err != nil {
		sp.Error("error deleting user's authentication codes from the db", zap.Error(err))
		return fmt.Errorf("error deleting user's auth codes: %w", err)
	}

	return nil
}

func (s Storage) InsertAuthCode(ctx context.Context, characterID int32, authCode string) error {
	ctx, sp := sl.OpenCorrelatedSpan(ctx, sl.NewID())
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	insert := s.DB.Insert("authentication_codes").
		Columns("character_id", "code").
		Values(characterID, authCode)

	sqlStr, args, err := insert.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return err
	} else {
		sp.With(
			zap.String("query", sqlStr),
			zap.Any("args", args),
		)
		sp.Debug("InsertAuthCode(): sql query")
	}

	_, err = insert.QueryContext(ctx)
	if err != nil {
		sp.Error("error inserting authentication code", zap.Error(err))
		return err
	}

	return nil
}

func (s Storage) UpdateAuthCode(ctx context.Context, authCode string) error {
	ctx, sp := sl.OpenCorrelatedSpan(ctx, sl.NewID())
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := s.DB.Update("authentication_codes").
		Set("used", true).
		Where(sq.Eq{"code": authCode})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return err
	} else {
		sp.With(
			zap.String("query", sqlStr),
			zap.Any("args", args),
		)
		sp.Debug("UpdateAuthCode(): sql query")
	}

	_, err = query.QueryContext(ctx)
	if err != nil {
		sp.Error("error updating authentication code", zap.Error(err))
		return err
	}

	return nil
}
