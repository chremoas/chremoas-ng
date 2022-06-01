package storage

import (
	"context"
	"database/sql"
	"errors"

	sq "github.com/Masterminds/squirrel"
	sl "github.com/bhechinger/spiffylogger"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	"github.com/lib/pq"
	"go.uber.org/zap"
)

var ErrNoDiscordUser = errors.New("no such discord user")
var ErrUserMapped = errors.New("user already mapped to character")

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
		sp.Debug("GetDiscordUser(): sql query")
	}

	err = query.Scan(&discordID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNoDiscordUser
		}

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
		sp.Debug("GetDiscordCharacter(): sql query")
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
		sp.Debug("InsertUserCharacterMap(): sql query")
	}

	_, err = query.QueryContext(ctx)
	if err != nil {
		if err.(*pq.Error).Code == "23505" {
			// Duplicate entry, which is fine, actually
			return ErrUserMapped
		}

		sp.Error("Error updating user character map", zap.Error(err))
		return err
	}

	return nil
}

func (s Storage) DeleteDiscordUser(ctx context.Context, chatID string) error {
	ctx, sp := sl.OpenCorrelatedSpan(ctx, sl.NewID())
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sp.With(zap.String("chatID", chatID))

	sp.Warn("Deleting user from character map")
	query := s.DB.Delete("user_character_map").
		Where(sq.Eq{"chat_id": chatID})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return err
	} else {
		sp.With(
			zap.String("query", sqlStr),
			zap.Any("args", args),
		)
		sp.Debug("DeleteDiscordUser(): sql query")
	}

	_, err = query.QueryContext(ctx)
	if err != nil {
		sp.Error("error deleting role", zap.Error(err))
		return err
	}

	// Clean up dependencies
	characters, err := s.GetDiscordCharacters(ctx, chatID)
	if err != nil {
		sp.Error("Error getting discord user characters", zap.Error(err))
		return err
	}

	for c := range characters {
		sp.With(zap.Any("character", characters[c]))

		sp.Warn("Deleting user's authentication codes")
		err := s.DeleteAuthCodes(ctx, characters[c].ID)
		if err != nil {
			sp.Error("Error deleting auth codes", zap.Error(err))
			return err
		}

		sp.Warn("Deleting user's character")
		err = s.DeleteCharacter(ctx, characters[c].ID)
		if err != nil {
			sp.Error("Error deleting character", zap.Error(err))
			return err
		}
	}

	return nil
}
