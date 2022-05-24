package storage

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	sl "github.com/bhechinger/spiffylogger"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	"go.uber.org/zap"
)

func (s Storage) GetCharacterCount(ctx context.Context, characterID int32) (int, error) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	query := s.DB.Select("count(*)").
		From("characters").
		Where(sq.Eq{"id": characterID})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return -1, err
	} else {
		sp.With(
			zap.String("query", sqlStr),
			zap.Any("args", args),
		)
		sp.Debug("GetCharacterCount(): sql query")
	}

	var count int

	err = query.Scan(&count)
	if err != nil {
		sp.Error("error getting character count", zap.Error(err))
		return -1, err
	}

	return count, nil
}

func (s Storage) GetCharacter(ctx context.Context, characterID int) (payloads.Character, error) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	query := s.DB.Select("name", "corporation_id").
		From("characters").
		Where(sq.Eq{"id": characterID})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return payloads.Character{}, err
	} else {
		sp.With(
			zap.String("query", sqlStr),
			zap.Any("args", args),
		)
		sp.Debug("GetCharacter(): sql query")
	}

	var character payloads.Character

	err = query.Scan(&character.Name, &character.CorporationID)
	if err != nil {
		sp.Error("error getting character name and corporation", zap.Error(err))
		return payloads.Character{}, err
	}

	return character, nil
}

func (s Storage) GetCharacters(ctx context.Context) ([]payloads.Character, error) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := s.DB.Select("id", "name", "corporation_id", "token").
		From("characters")

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return nil, err
	} else {
		sp.With(
			zap.String("query", sqlStr),
			zap.Any("args", args),
		)
		sp.Debug("GetCharacters(): sql query")
	}

	rows, err := query.QueryContext(ctx)
	if err != nil {
		sp.Error("error getting character list from the db", zap.Error(err))
		return nil, err
	}

	defer func() {
		if err = rows.Close(); err != nil {
			sp.Error("error closing role", zap.Error(err))
		}
	}()

	var characters []payloads.Character

	for rows.Next() {
		var character payloads.Character

		err = rows.Scan(&character.ID, &character.Name, &character.CorporationID, &character.Token)
		if err != nil {
			sp.Error("error scanning character values", zap.Error(err))
			continue
		}

		characters = append(characters, character)
	}

	return characters, nil
}

func (s Storage) UpsertCharacter(ctx context.Context, characterID, corporationID int32, name, token string) error {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sp.With(
		zap.String("component", "storage"),
		zap.String("sub-component", "character"),
		zap.Int32("character_id", characterID),
		zap.Int32("corporation_id", corporationID),
		zap.String("name", name),
		zap.String("token", token),
	)

	var query sq.InsertBuilder

	if token != "" {
		query = s.DB.Insert("characters").
			Columns("id", "name", "token", "corporation_id").
			Values(characterID, name, token, corporationID).
			Suffix("ON CONFLICT (id) DO UPDATE SET name=?, token=?, corporation_id=?", name, token, corporationID)
	} else {
		query = s.DB.Insert("characters").
			Columns("id", "name", "corporation_id").
			Values(characterID, name, corporationID).
			Suffix("ON CONFLICT (id) DO UPDATE SET name=?, corporation_id=?", name, corporationID)
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
		sp.Debug("UpsertCharacter(): sql query")
	}

	rows, err := query.QueryContext(ctx)
	if err != nil {
		sp.Error("Error inserting character", zap.Error(err))
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

func (s Storage) DeleteCharacter(ctx context.Context, characterID int32) error {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := s.DB.Delete("characters").
		Where(sq.Eq{"id": characterID})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return err
	} else {
		sp.With(
			zap.String("query", sqlStr),
			zap.Any("args", args),
		)
		sp.Debug("DeleteCharacter(): sql query")
	}

	_, err = query.QueryContext(ctx)
	if err != nil {
		sp.Error("error deleting user's character from the db", zap.Error(err))
		return err
	}

	return nil
}
