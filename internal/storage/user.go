package storage

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	sl "github.com/bhechinger/spiffylogger"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	"github.com/lib/pq"
	"go.uber.org/zap"
)

func (s Storage) GetDiscordUser(ctx context.Context, characterID int32) (string, error) {
	ctx, sp := sl.OpenCorrelatedSpan(ctx, sl.NewID())
	defer sp.Close()

	var discordID string

	query := s.DB.Select("chat_id").
		From("user_character_map").
		Where(sq.Eq{"character_id": characterID})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return "", err
	} else {
		sp.With(
			zap.String("query", sqlStr),
			zap.Any("args", args),
		)
		sp.Debug("sql query")
	}

	err = query.Scan(&discordID)
	if err != nil {
		sp.Error("error getting discord info", zap.Error(err))
		return "", err
	}

	return discordID, nil
}

func (s Storage) GetDiscordCharacters(ctx context.Context, discordID string) ([]payloads.Character, error) {
	ctx, sp := sl.OpenCorrelatedSpan(ctx, sl.NewID())
	defer sp.Close()

	query := s.DB.Select("character_id").
		From("user_character_map").
		Where(sq.Eq{"chat_id": discordID})

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

		err = rows.Scan(&character.ID)
		if err != nil {
			sp.Error("error scanning character id", zap.Error(err))
			return nil, err
		}

		characters = append(characters, character)
	}

	return characters, nil
}

func (s Storage) InsertUserCharacterMap(ctx context.Context, sender string, characterID int) error {
	ctx, sp := sl.OpenCorrelatedSpan(ctx, sl.NewID())
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := s.DB.Insert("user_character_map").
		Values(sender, characterID)

	sqlStr, args, err := query.ToSql()
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

	_, err = query.QueryContext(ctx)
	if err != nil {
		if err.(*pq.Error).Code != "23505" {
			// Duplicate entry, which is fine, actually
			return nil
		}

		sp.Error("Error updating user character map", zap.Error(err))
		return err
	}

	return nil
}
