package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/bwmarrin/discordgo"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/filters"
	"github.com/chremoas/chremoas-ng/internal/roles"
	"github.com/lib/pq"
	"go.uber.org/zap"
)

func Create(_ context.Context, request *CreateRequest, deps common.Dependencies) (*string, error) {
	var (
		err   error
		count int
	)

	ctx, cancel := context.WithCancel(deps.Context)
	defer cancel()

	// ===========================================================================================
	// Get alliance

	// We MIGHT NOT have any kind of alliance information
	if request.Alliance != nil {
		deps.Logger.Info("Checking alliance",
			zap.Int32("id", request.Alliance.ID),
			zap.String("name", request.Alliance.Name),
			zap.String("ticker", request.Alliance.Ticker),
		)
		err = deps.DB.Select("COUNT(*)").
			From("alliances").
			Where(sq.Eq{"id": request.Alliance.ID}).
			Scan(&count)
		if err != nil {
			deps.Logger.Error("error getting alliance count", zap.Error(err))
			return nil, err
		}

		if count == 0 {
			deps.Logger.Info("Alliance not found, adding to db",
				zap.Int32("id", request.Alliance.ID),
				zap.String("name", request.Alliance.Name),
				zap.String("ticker", request.Alliance.Ticker),
			)
			_, err = deps.DB.Insert("alliances").
				Columns("id", "name", "ticker").
				Values(request.Alliance.ID, request.Alliance.Name, request.Alliance.Ticker).
				QueryContext(ctx)
			if err != nil {
				deps.Logger.Error("error inserting into alliances table", zap.Error(err))
				return nil, err
			}
		}

		role, err := roles.GetRoles(false, &request.Alliance.Ticker, deps)
		if err != nil {
			deps.Logger.Error("error getting roles", zap.Error(err))
			return nil, err
		}

		if len(role) == 0 {
			roles.Add(false, false, request.Alliance.Ticker, request.Alliance.Name, "discord", deps)
		}
	}

	// ===========================================================================================
	// Get Corporation

	err = deps.DB.Select("COUNT(*)").
		From("corporations").
		Where(sq.Eq{"id": request.Corporation.ID}).
		Scan(&count)
	if err != nil {
		deps.Logger.Error("error getting corporation count", zap.Error(err))
		return nil, err
	}

	if count == 0 {
		deps.Logger.Info("Corporation not found, adding to db",
			zap.Int32("id", request.Corporation.ID),
			zap.String("name", request.Corporation.Name),
			zap.String("ticker", request.Corporation.Ticker),
		)
		if request.Alliance == nil {
			_, err = deps.DB.Insert("corporations").
				Columns("id", "name", "ticker").
				Values(request.Corporation.ID, request.Corporation.Name, request.Corporation.Ticker).
				QueryContext(ctx)
		} else {
			_, err = deps.DB.Insert("corporations").
				Columns("id", "name", "ticker", "alliance_id").
				Values(request.Corporation.ID, request.Corporation.Name, request.Corporation.Ticker, request.Alliance.ID).
				QueryContext(ctx)
		}
		if err != nil {
			deps.Logger.Error("error inserting corporation", zap.Error(err))
			return nil, err
		}
	}

	role, err := roles.GetRoles(false, &request.Corporation.Ticker, deps)
	if err != nil {
		deps.Logger.Error("error getting roles", zap.Error(err))
		return nil, err
	}

	if len(role) == 0 {
		roles.Add(false, false, request.Corporation.Ticker, request.Corporation.Name, "discord", deps)
	}

	// ===========================================================================================
	// Get character

	err = deps.DB.Select("count(*)").
		From("characters").
		Where(sq.Eq{"id": request.Character.ID}).
		Scan(&count)
	if err != nil {
		deps.Logger.Error("error getting character count", zap.Error(err))
		return nil, err
	}

	if count == 0 {
		deps.Logger.Info("Character not found, adding to db",
			zap.Int32("id", request.Character.ID),
			zap.Int32("corporation id", request.Character.CorporationID),
			zap.String("name", request.Character.Name),
		)
		_, err = deps.DB.Insert("characters").
			Columns("id", "name", "token", "corporation_id").
			Values(request.Character.ID, request.Character.Name, request.Token, request.Corporation.ID).
			QueryContext(ctx)
		if err != nil {
			deps.Logger.Error("error inserting character", zap.Error(err))
			return nil, err
		}
	}

	// Now... make an auth string... hopefully this isn't too ugly
	b := make([]byte, 6)
	rand.Read(b) // TODO: Fix this?
	authCode := hex.EncodeToString(b)

	_, err = deps.DB.Insert("authentication_codes").
		Columns("character_id", "code").
		Values(request.Character.ID, authCode).
		QueryContext(ctx)
	if err != nil {
		deps.Logger.Error("error inserting authentication code", zap.Error(err))
		return nil, err
	}

	return &authCode, nil
}

func Confirm(authCode, sender string, deps common.Dependencies) []*discordgo.MessageSend {
	var (
		err            error
		characterID    int
		corporationID  int
		allianceID     sql.NullInt64
		corpTicker     string
		allianceTicker string
		name           string
		used           bool
	)

	ctx, cancel := context.WithCancel(deps.Context)
	defer cancel()

	err = deps.DB.Select("character_id", "used").
		From("authentication_codes").
		Where(sq.Eq{"code": authCode}).
		Scan(&characterID, &used)
	if err != nil {
		// if err.(*pq.Error).Code == "23505" {
		// 	return common.SendError(fmt.Sprintf("%s `%s` (%s) already exists", roleType[sig], name, ticker))
		// }
		// deps.Logger.Debug("sql error", zap.Any("code", err), zap.String("type", fmt.Sprintf("%t", err)))
		// err is a string?
		deps.Logger.Error("error getting authentication code", zap.Error(err), zap.String("auth code", authCode))
		return common.SendError(fmt.Sprintf("Error getting authentication code: %s", authCode))
	}

	if used {
		return common.SendError(fmt.Sprintf("Auth code already used: %s", authCode))
	}

	err = deps.DB.Select("name", "corporation_id").
		From("characters").
		Where(sq.Eq{"id": characterID}).
		Scan(&name, &corporationID)
	if err != nil {
		deps.Logger.Error("error getting character name", zap.Error(err), zap.Int("character id", characterID))
		return common.SendError(fmt.Sprintf("Error getting character's name and corporation: %d", characterID))
	}

	_, err = deps.DB.Update("authentication_codes").
		Set("used", true).
		QueryContext(ctx)
	if err != nil {
		deps.Logger.Error("error updating authentication code", zap.Error(err))
		return common.SendError("Error updating auth code used")
	}

	_, err = deps.DB.Insert("user_character_map").
		Values(sender, characterID).
		QueryContext(ctx)
	if err != nil {
		// I don't love this but I can't find a better way right now
		if err.(*pq.Error).Code != "23505" {
			deps.Logger.Error("error linking user with character", zap.Error(err))
			return common.SendError("Error linking user with character")
		}
	}

	// get corp ticker
	err = deps.DB.Select("ticker", "alliance_id").
		From("corporations").
		Where(sq.Eq{"id": corporationID}).
		Scan(&corpTicker, &allianceID)
	if err != nil {
		deps.Logger.Error("error getting ticker and alliance id",
			zap.Error(err), zap.Int("corporation ID", corporationID))
		return common.SendError("Error updating auth code used")
	}

	filters.AddMember(sender, corpTicker, deps)

	if allianceID.Valid {
		// get alliance ticker if there is an alliance
		err = deps.DB.Select("ticker").
			From("alliances").
			Where(sq.Eq{"id": allianceID}).
			Scan(&allianceTicker)
		if err != nil {
			deps.Logger.Error("error getting alliance ticker",
				zap.Error(err), zap.Int64("alliance ID", allianceID.Int64))
			return common.SendError("Error updating auth code used")
		}

		filters.AddMember(sender, allianceTicker, deps)
	}

	return common.SendSuccess(fmt.Sprintf("<@%s> **Success**: %s has been successfully authed.", sender, name))
}
