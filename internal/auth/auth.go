package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	sl "github.com/bhechinger/spiffylogger"
	"github.com/bwmarrin/discordgo"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/filters"
	"github.com/chremoas/chremoas-ng/internal/goof"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	"github.com/chremoas/chremoas-ng/internal/roles"
	"go.uber.org/zap"
)

func Create(ctx context.Context, request *payloads.CreateRequest, deps common.Dependencies) (*string, error) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.Any("request", request),
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// ===========================================================================================
	// Get alliance

	// We MIGHT NOT have any kind of alliance information
	if request.Alliance != nil {
		sp.Info("Checking alliance")

		count, err := deps.Storage.GetAllianceCount(ctx, request.Alliance.ID)
		if err != nil {
			sp.Error("Error getting alliance count", zap.Error(err))
			return nil, err
		}

		if count == 0 {
			sp.Info("Alliance not found, adding to db")
			err = deps.Storage.UpsertAlliance(ctx, request.Alliance.ID, request.Alliance.Name, request.Alliance.Ticker)
			if err != nil {
				sp.Error("Error upserting alliance", zap.Error(err))
				return nil, err
			}
		}

		_, err = deps.Storage.GetRoleByType(ctx, false, request.Alliance.Ticker)
		if err != nil {
			sp.Error("error getting roles", zap.Error(err))
			return nil, err
		}

		roles.Add(ctx, false, false, request.Alliance.Ticker, request.Alliance.Name, "discord", deps)
	}

	// ===========================================================================================
	// Get Corporation

	count, err := deps.Storage.GetCorporationCount(ctx, request.Corporation.ID)
	if err != nil {
		sp.Error("Error getting corporation count", zap.Error(err))
		return nil, err
	}

	if count == 0 {
		sp.Info("Corporation not found, adding to db")
		err = deps.Storage.UpsertCorporation(ctx, request.Corporation.ID, request.Alliance.ID, request.Corporation.Name, request.Corporation.Ticker)
		if err != nil {
			sp.Error("Error upserting corporation", zap.Error(err))
		}
	}

	_, err = deps.Storage.GetRoleByType(ctx, false, request.Corporation.Ticker)
	if err != nil {
		sp.Error("error getting roles", zap.Error(err))
		return nil, err
	}

	roles.Add(ctx, false, false, request.Corporation.Ticker, request.Corporation.Name, "discord", deps)

	// ===========================================================================================
	// Get character

	count, err = deps.Storage.GetCharacterCount(ctx, request.Character.ID)
	if err != nil {
		sp.Error("Error getting character count", zap.Error(err))
	}

	if count == 0 {
		sp.Info("Character not found, adding to db")
		err = deps.Storage.UpsertCharacter(ctx, request.Character.ID, request.Corporation.ID, request.Character.Name, request.Token)
		if err != nil {
			sp.Error("Error upserting character", zap.Error(err))
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

	err = deps.Storage.InsertAuthCode(ctx, request.Character.ID, authCode)
	if err != nil {
		sp.Error("Error inserting auth code", zap.Error(err))
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

	characterID, used, err := deps.Storage.GetAuthCode(ctx, authCode)
	if err != nil {
		sp.Error("Error getting auth code", zap.Error(err))
		return common.SendError(fmt.Sprintf("Error getting character's auth code from the DB: %d", characterID))
	}

	sp.With(
		zap.Int("character_id", characterID),
		zap.Bool("used", used),
	)

	if used {
		sp.Warn("auth code already used")
		return common.SendError(fmt.Sprintf("Auth code already used: %s", authCode))
	}

	character, err := deps.Storage.GetCharacter(ctx, characterID)
	if err != nil {
		sp.Error("Error getting character", zap.Error(err))
		return common.SendError(fmt.Sprintf("Error getting character from the DB: %d", characterID))
	}

	sp.With(
		zap.String("character_name", character.Name),
		zap.Int32("corporation_id", character.CorporationID),
	)

	err = deps.Storage.UpdateAuthCode(ctx, authCode)
	if err != nil {
		sp.Error("Error updating auth code", zap.Error(err))
		return common.SendError("Error updating auth code")
	}

	err = deps.Storage.InsertUserCharacterMap(ctx, sender, characterID)
	if err != nil {
		sp.Error("Error inserting user character map", zap.Error(err))
		return common.SendError("Error inserting user character map")
	}

	// get corp ticker
	corporation, err := deps.Storage.GetCorporation(ctx, character.CorporationID)
	if err != nil {
		sp.Error("Error getting corporation", zap.Error(err))
		return common.SendError("Error getting corporation")
	}

	sp.With(
		zap.String("corp_ticker", corporation.Ticker),
		zap.Any("alliance_id", corporation.AllianceID),
	)

	filters.AddMember(ctx, sender, corporation.Ticker, deps)

	if corporation.AllianceID.Valid {
		// get alliance ticker if there is an alliance
		alliance, err := deps.Storage.GetAlliance(ctx, corporation.AllianceID.Int32)
		if err != nil {
			if errors.Is(err, goof.NoSuchAlliance) {
				return common.SendError("No such alliance")
			}

			sp.Error("Error getting alliance", zap.Error(err))
			return common.SendError("Error getting alliance")
		}
		sp.With(zap.String("alliance_ticker", alliance.Ticker))

		filters.AddMember(ctx, sender, alliance.Ticker, deps)
	}

	sp.Info("authed user")
	return common.SendSuccess(fmt.Sprintf("<@%s> **Success**: %s has been successfully authed.", sender, character.Name))
}
