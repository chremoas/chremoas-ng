package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/filters"
	"github.com/chremoas/chremoas-ng/internal/roles"
	"github.com/lib/pq"
	"github.com/nsqio/go-nsq"
	"go.uber.org/zap"
)

func Create(_ context.Context, request *CreateRequest, logger *zap.SugaredLogger, db *sq.StatementBuilderType, nsq *nsq.Producer) (*string, error) {
	var (
		err   error
		count int
	)

	// ===========================================================================================
	// Get alliance

	// We MIGHT NOT have any kind of alliance information
	if request.Alliance != nil {
		logger.Infof("Checking alliance: %d", request.Alliance.ID)
		err = db.Select("COUNT(*)").
			From("alliances").
			Where(sq.Eq{"id": request.Alliance.ID}).
			QueryRow().Scan(&count)
		if err != nil {
			logger.Error(err)
			return nil, err
		}

		if count == 0 {
			logger.Infof("Alliance not found, adding to db: %d", request.Alliance.ID)
			_, err = db.Insert("alliances").
				Columns("id", "name", "ticker").
				Values(request.Alliance.ID, request.Alliance.Name, request.Alliance.Ticker).
				Query()
			if err != nil {
				logger.Error(err)
				return nil, err
			}
		}

		role, err := roles.GetRoles(false, &request.Alliance.Ticker, logger, db)
		if err != nil {
			logger.Error(err)
			return nil, err
		}

		if len(role) == 0 {
			roles.Add(false, false, request.Alliance.Ticker, request.Alliance.Name, "discord", logger, db, nsq)
		}
	}

	// ===========================================================================================
	// Get Corporation

	err = db.Select("COUNT(*)").
		From("corporations").
		Where(sq.Eq{"id": request.Corporation.ID}).
		QueryRow().Scan(&count)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	if count == 0 {
		logger.Infof("Corporation not found, adding to db: %d", request.Corporation.ID)
		if request.Alliance == nil {
			_, err = db.Insert("corporations").
				Columns("id", "name", "ticker").
				Values(request.Corporation.ID, request.Corporation.Name, request.Corporation.Ticker).
				Query()
		} else {
			_, err = db.Insert("corporations").
				Columns("id", "name", "ticker", "alliance_id").
				Values(request.Corporation.ID, request.Corporation.Name, request.Corporation.Ticker, request.Alliance.ID).
				Query()
		}
		if err != nil {
			logger.Error(err)
			return nil, err
		}
	}

	role, err := roles.GetRoles(false, &request.Corporation.Ticker, logger, db)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	if len(role) == 0 {
		roles.Add(false, false, request.Corporation.Ticker, request.Corporation.Name, "discord", logger, db, nsq)
	}

	// ===========================================================================================
	// Get character

	err = db.Select("COUNT(*)").
		From("characters").
		Where(sq.Eq{"id": request.Character.ID}).
		QueryRow().Scan(&count)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	if count == 0 {
		logger.Infof("Character not found, adding to db: %d", request.Character.ID)
		_, err = db.Insert("characters").
			Columns("id", "name", "token", "corporation_id").
			Values(request.Character.ID, request.Character.Name, request.Token, request.Corporation.ID).
			Query()
		if err != nil {
			logger.Error(err)
			return nil, err
		}
	}

	//Now... make an auth string... hopefully this isn't too ugly
	b := make([]byte, 6)
	rand.Read(b)
	authCode := hex.EncodeToString(b)

	_, err = db.Insert("authentication_codes").
		Columns("character_id", "code").
		Values(request.Character.ID, authCode).
		Query()
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	return &authCode, nil
}

func Confirm(authCode, sender string, logger *zap.SugaredLogger, db *sq.StatementBuilderType, nsq *nsq.Producer) string {
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

	err = db.Select("character_id", "used").
		From("authentication_codes").
		Where(sq.Eq{"code": authCode}).
		QueryRow().Scan(&characterID, &used)
	if err != nil {
		logger.Error(err)
		return common.SendError(fmt.Sprintf("Error getting authentication code: %s", authCode))
	}

	if used {
		return common.SendError(fmt.Sprintf("Auth code already used: %s", authCode))
	}

	err = db.Select("name", "corporation_id").
		From("characters").
		Where(sq.Eq{"id": characterID}).
		QueryRow().Scan(&name, &corporationID)
	if err != nil {
		logger.Error(err)
		return common.SendError(fmt.Sprintf("Error getting character name: %d", characterID))
	}

	_, err = db.Update("authentication_codes").
		Set("used", true).
		Query()
	if err != nil {
		logger.Error(err)
		return common.SendError("Error updating auth code used")
	}

	_, err = db.Insert("user_character_map").
		Values(sender, characterID).
		Query()
	if err != nil {
		// I don't love this but I can't find a better way right now
		if err.(*pq.Error).Code == "23505" {
			return common.SendError("User already authenticated")
		}
		logger.Error(err)
		return common.SendError("Error linking user with character")
	}

	// get corp ticker
	err = db.Select("ticker", "alliance_id").
		From("corporations").
		Where(sq.Eq{"id": corporationID}).
		QueryRow().Scan(&corpTicker, &allianceID)
	if err != nil {
		logger.Error(err)
		return common.SendError("Error updating auth code used")
	}

	filters.AddMember(sender, corpTicker, logger, db, nsq)

	if allianceID.Valid {
		// get alliance ticker if there is an alliance
		err = db.Select("ticker").
			From("alliances").
			Where(sq.Eq{"id": allianceID}).
			QueryRow().Scan(&allianceTicker)
		if err != nil {
			logger.Error(err)
			return common.SendError("Error updating auth code used")
		}

		filters.AddMember(sender, allianceTicker, logger, db, nsq)
	}

	return common.SendSuccess(fmt.Sprintf("<@%s> **Success**: %s has been successfully authed.", sender, name))
}
