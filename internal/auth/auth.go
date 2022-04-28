package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	sl "github.com/bhechinger/spiffylogger"
	"github.com/bwmarrin/discordgo"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/filters"
	"github.com/chremoas/chremoas-ng/internal/roles"
	"github.com/lib/pq"
	"go.uber.org/zap"
)

func Create(ctx context.Context, request *CreateRequest, deps common.Dependencies) (*string, error) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.Any("request", request),
	)

	var (
		err   error
		count int
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// ===========================================================================================
	// Get alliance

	// We MIGHT NOT have any kind of alliance information
	if request.Alliance != nil {
		sp.Info("Checking alliance")
		query := deps.DB.Select("COUNT(*)").
			From("alliances").
			Where(sq.Eq{"id": request.Alliance.ID})

		sqlStr, args, err := query.ToSql()
		if err != nil {
			sp.Error("error getting sql", zap.Error(err))
			return nil, err
		} else {
			sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
		}

		err = query.Scan(&count)
		if err != nil {
			sp.Error("error getting alliance count", zap.Error(err))
			return nil, err
		}

		if count == 0 {
			sp.Info("Alliance not found, adding to db")
			insert := deps.DB.Insert("alliances").
				Columns("id", "name", "ticker").
				Values(request.Alliance.ID, request.Alliance.Name, request.Alliance.Ticker)

			sqlStr, args, err = insert.ToSql()
			if err != nil {
				sp.Error("error getting sql", zap.Error(err))
				return nil, err
			} else {
				sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
			}

			_, err = insert.QueryContext(ctx)
			if err != nil {
				sp.Error("error inserting into alliances table", zap.Error(err))
				return nil, err
			}
		}

		role, err := roles.GetRoles(ctx, false, &request.Alliance.Ticker, deps)
		if err != nil {
			sp.Error("error getting roles", zap.Error(err))
			return nil, err
		}

		if len(role) == 0 {
			roles.Add(ctx, false, false, request.Alliance.Ticker, request.Alliance.Name, "discord", deps)
		}
	}

	// ===========================================================================================
	// Get Corporation

	query := deps.DB.Select("COUNT(*)").
		From("corporations").
		Where(sq.Eq{"id": request.Corporation.ID})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return nil, err
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	err = query.Scan(&count)
	if err != nil {
		sp.Error("error getting corporation count", zap.Error(err))
		return nil, err
	}

	if count == 0 {
		sp.Info("Corporation not found, adding to db")
		if request.Alliance == nil {
			insert := deps.DB.Insert("corporations").
				Columns("id", "name", "ticker").
				Values(request.Corporation.ID, request.Corporation.Name, request.Corporation.Ticker)

			sqlStr, args, err = insert.ToSql()
			if err != nil {
				sp.Error("error getting sql", zap.Error(err))
				return nil, err
			} else {
				sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
			}

			_, err = insert.QueryContext(ctx)
		} else {
			insert := deps.DB.Insert("corporations").
				Columns("id", "name", "ticker", "alliance_id").
				Values(request.Corporation.ID, request.Corporation.Name, request.Corporation.Ticker, request.Alliance.ID)

			sqlStr, args, err = insert.ToSql()
			if err != nil {
				sp.Error("error getting sql", zap.Error(err))
				return nil, err
			} else {
				sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
			}

			_, err = insert.QueryContext(ctx)
		}
		if err != nil {
			sp.Error("error inserting corporation", zap.Error(err))
			return nil, err
		}
	}

	role, err := roles.GetRoles(ctx, false, &request.Corporation.Ticker, deps)
	if err != nil {
		sp.Error("error getting roles", zap.Error(err))
		return nil, err
	}

	if len(role) == 0 {
		roles.Add(ctx, false, false, request.Corporation.Ticker, request.Corporation.Name, "discord", deps)
	}

	// ===========================================================================================
	// Get character

	query = deps.DB.Select("count(*)").
		From("characters").
		Where(sq.Eq{"id": request.Character.ID})

	sqlStr, args, err = query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return nil, err
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	err = query.Scan(&count)
	if err != nil {
		sp.Error("error getting character count", zap.Error(err))
		return nil, err
	}

	if count == 0 {
		sp.Info("Character not found, adding to db")
		insert := deps.DB.Insert("characters").
			Columns("id", "name", "token", "corporation_id").
			Values(request.Character.ID, request.Character.Name, request.Token, request.Corporation.ID)

		sqlStr, args, err = insert.ToSql()
		if err != nil {
			sp.Error("error getting sql", zap.Error(err))
			return nil, err
		} else {
			sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
		}

		_, err = insert.QueryContext(ctx)
		if err != nil {
			sp.Error("error inserting character", zap.Error(err))
			return nil, err
		}
	}

	// Now... make an auth string... hopefully this isn't too ugly
	b := make([]byte, 6)
	_, err = rand.Read(b)
	if err != nil {
		sp.Error("error creating random string", zap.Error(err))
		return nil, err
	}
	authCode := hex.EncodeToString(b)

	insert := deps.DB.Insert("authentication_codes").
		Columns("character_id", "code").
		Values(request.Character.ID, authCode)

	sqlStr, args, err = insert.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return nil, err
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	_, err = insert.QueryContext(ctx)
	if err != nil {
		sp.Error("error inserting authentication code", zap.Error(err))
		return nil, err
	}

	return &authCode, nil
}

func Confirm(ctx context.Context, authCode, sender string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("auth_code", authCode),
		zap.String("sender", sender),
	)

	var (
		err            error
		characterID    int
		corporationID  int
		allianceID     sql.NullInt64
		corpTicker     string
		allianceTicker string
		characterName  string
		used           bool
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := deps.DB.Select("character_id", "used").
		From("authentication_codes").
		Where(sq.Eq{"code": authCode})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return common.SendError(fmt.Sprintf("Error generating SQL string: %s", err))
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	err = query.Scan(&characterID, &used)
	if err != nil {
		// if err.(*pq.Error).Code == "23505" {
		// 	return common.SendError(fmt.Sprintf("%s `%s` (%s) already exists", roleType[sig], name, ticker))
		// }
		// deps.Logger.Debug("sql error", zap.Any("code", err), zap.String("type", fmt.Sprintf("%t", err)))
		// err is a string?
		sp.Error("error getting authentication code details", zap.Error(err))
		return common.SendError(fmt.Sprintf("Error getting authentication code: %s", authCode))
	}

	sp.With(
		zap.Int("character_id", characterID),
		zap.Bool("used", used),
	)

	if used {
		sp.Warn("auth code already used")
		return common.SendError(fmt.Sprintf("Auth code already used: %s", authCode))
	}

	query = deps.DB.Select("name", "corporation_id").
		From("characters").
		Where(sq.Eq{"id": characterID})

	sqlStr, args, err = query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return common.SendError(fmt.Sprintf("Error generating SQL string: %s", err))
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	err = query.Scan(&characterName, &corporationID)
	if err != nil {
		sp.Error("error getting character name and corporation", zap.Error(err))
		return common.SendError(fmt.Sprintf("Error getting character's name and corporation: %d", characterID))
	}

	sp.With(
		zap.String("character_name", characterName),
		zap.Int("corporation_id", corporationID),
	)

	update := deps.DB.Update("authentication_codes").Set("used", true)

	sqlStr, args, err = update.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return common.SendError(fmt.Sprintf("Error generating SQL string: %s", err))
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	_, err = update.QueryContext(ctx)
	if err != nil {
		sp.Error("error updating authentication code", zap.Error(err))
		return common.SendError("Error updating auth code used")
	}

	insert := deps.DB.Insert("user_character_map").Values(sender, characterID)

	sqlStr, args, err = insert.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return common.SendError(fmt.Sprintf("Error generating SQL string: %s", err))
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	_, err = insert.QueryContext(ctx)
	if err != nil {
		// I don't love this, but I can't find a better way right now
		if err.(*pq.Error).Code != "23505" {
			sp.Error("error linking user with character", zap.Error(err))
			return common.SendError("Error linking user with character")
		}
	}

	// get corp ticker
	query = deps.DB.Select("ticker", "alliance_id").
		From("corporations").
		Where(sq.Eq{"id": corporationID})

	sqlStr, args, err = query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return common.SendError(fmt.Sprintf("Error generating SQL string: %s", err))
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	err = query.Scan(&corpTicker, &allianceID)
	if err != nil {
		sp.Error("error getting ticker and alliance id", zap.Error(err))
		return common.SendError("Error updating auth code used")
	}

	sp.With(
		zap.String("corp_ticker", corpTicker),
		zap.Any("alliance_id", allianceID),
	)

	filters.AddMember(ctx, sender, corpTicker, deps)

	if allianceID.Valid {
		// get alliance ticker if there is an alliance
		query = deps.DB.Select("ticker").
			From("alliances").
			Where(sq.Eq{"id": allianceID})

		sqlStr, args, err = query.ToSql()
		if err != nil {
			sp.Error("error getting sql", zap.Error(err))
			return common.SendError(fmt.Sprintf("Error generating SQL string: %s", err))
		} else {
			sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
		}

		err = query.Scan(&allianceTicker)
		if err != nil {
			sp.Error("error getting alliance ticker", zap.Error(err))
			return common.SendError("Error updating auth code used")
		}

		sp.With(zap.String("alliance_ticker", allianceTicker))

		filters.AddMember(ctx, sender, allianceTicker, deps)
	}

	sp.Info("authed user")
	return common.SendSuccess(fmt.Sprintf("<@%s> **Success**: %s has been successfully authed.", sender, characterName))
}
