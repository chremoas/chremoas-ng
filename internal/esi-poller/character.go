package esi_poller

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/bhechinger/go-sets"
	sl "github.com/bhechinger/spiffylogger"
	"github.com/chremoas/chremoas-ng/internal/auth"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/filters"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	"go.uber.org/zap"
)

func (aep *authEsiPoller) updateCharacters(ctx context.Context) (int, int, error) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(zap.String("sub-component", "character"))

	var (
		count      int
		errorCount int
		err        error
		character  auth.Character
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	rows, err := aep.dependencies.DB.Select("id", "name", "corporation_id", "token").
		From("characters").
		QueryContext(ctx)
	if err != nil {
		return -1, -1, fmt.Errorf("error getting character list from the db: %w", err)
	}

	defer func() {
		if err = rows.Close(); err != nil {
			sp.Error("error closing role", zap.Error(err))
		}
	}()

	for rows.Next() {
		err = rows.Scan(&character.ID, &character.Name, &character.CorporationID, &character.Token)
		if err != nil {
			return -1, -1, fmt.Errorf("error scanning character values: %w", err)
		}

		err := aep.updateCharacter(ctx, character)
		if err != nil {
			sp.Error("error updating character", zap.Error(err),
				zap.Int32("id", character.ID), zap.String("name", character.Name))
			errorCount += 1
			continue
		}

		count += 1
	}

	return count, errorCount, nil
}

func (aep *authEsiPoller) updateCharacter(ctx context.Context, character auth.Character) error {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(zap.String("sub-component", "character"))

	response, _, err := aep.esiClient.ESI.CharacterApi.GetCharactersCharacterId(ctx, character.ID, nil)
	if err != nil {
		sp.Info("Character not found", zap.Int32("id", character.ID), zap.String("name", character.Name))

		return err
	}

	if response.CorporationId == 0 {
		return fmt.Errorf("CorpID is 0: ESI error most likely, probably transient")
	}

	if character.CorporationID != response.CorporationId {
		sp.Info("Updating character's corp",
			zap.String("character", character.Name), zap.Int32("corporation", response.CorporationId))

		// Check if corporation exists, if not, add it.
		corporation := auth.Corporation{ID: response.CorporationId}
		err = aep.updateCorporation(ctx, corporation)
		if err != nil {
			return fmt.Errorf("error updating corporation `%s`: %s", corporation.Name, err)
		}

		// We need the old corp Ticker to remove from the filter
		var (
			oldCorp       string
			oldAllianceID int
		)
		err = aep.dependencies.DB.Select("ticker", "alliance_id").
			From("corporations").
			Where(sq.Eq{"id": character.CorporationID}).
			Scan(&oldCorp, &oldAllianceID)
		if err != nil {
			sp.Error("error getting old corp info", zap.Error(err),
				zap.String("character", character.Name),
				zap.Int32("corporation", character.CorporationID))
			return err
		}

		err = aep.upsertCharacter(ctx, character.ID, response.CorporationId, response.Name, character.Token)
		if err != nil {
			return fmt.Errorf("error upserting character `%d` with corp '%d': %s", character.ID, response.CorporationId, err)
		}

		// We need the new corp Ticker to add to the filter
		var (
			newCorp       string
			newAllianceID int
		)
		err = aep.dependencies.DB.Select("ticker", "alliance_id").
			From("corporations").
			Where(sq.Eq{"id": response.CorporationId}).
			Scan(&newCorp, &newAllianceID)
		if err != nil {
			sp.Error("error getting new corp info", zap.Error(err),
				zap.String("character", character.Name),
				zap.Int32("corporation", character.CorporationID))
			return err
		}

		// We need the discord user ID
		var discordID string
		err = aep.dependencies.DB.Select("chat_id").
			From("user_character_map").
			Where(sq.Eq{"character_id": character.ID}).
			Scan(&discordID)
		if err != nil {
			sp.Error("error getting discord info", zap.Error(err),
				zap.String("character", character.Name),
				zap.Int32("corporation", character.CorporationID))
			return err
		}

		// I don't like this here because if it fails it will never get cleaned up
		// Change the filter they are in, requires discord ID
		sp.Debug("removing user from corp", zap.String("user", discordID), zap.String("corp", oldCorp))
		filters.RemoveMember(ctx, discordID, oldCorp, aep.dependencies)
		sp.Debug("adding user to corp", zap.String("user", discordID), zap.String("corp", newCorp))
		filters.AddMember(ctx, discordID, newCorp, aep.dependencies)

		if newAllianceID != oldAllianceID {
			// new corp is in a different alliance, gotta switch those.

			var oldAlliance, newAlliance string
			err = aep.dependencies.DB.Select("ticker").
				From("alliances").
				Where(sq.Eq{"id": oldAllianceID}).
				Scan(&oldAlliance)
			if err != nil {
				sp.Error("error getting old alliance ticker", zap.Error(err),
					zap.Int("allianceID", oldAllianceID))
				return err
			}

			err = aep.dependencies.DB.Select("ticker").
				From("alliances").
				Where(sq.Eq{"id": newAllianceID}).
				Scan(&newAlliance)
			if err != nil {
				sp.Error("error getting new alliance ticker", zap.Error(err),
					zap.Int("allianceID", newAllianceID))
				return err
			}

			sp.Debug("removing user from alliance", zap.String("user", discordID), zap.String("alliance", oldAlliance))
			filters.RemoveMember(ctx, discordID, oldAlliance, aep.dependencies)
			sp.Debug("adding user to alliance", zap.String("user", discordID), zap.String("alliance", newAlliance))
			filters.AddMember(ctx, discordID, newAlliance, aep.dependencies)
		}
	}

	// We need the chatID of the user, so let's get that.
	var chatID int
	err = aep.dependencies.DB.Select("chat_id").
		From("user_character_map").
		Where(sq.Eq{"character_id": character.ID}).
		Scan(&chatID)
	if err != nil {
		sp.Error("error getting chat_id for character", zap.Error(err), zap.Int32("id", character.ID))
		return err
	}

	strChatID := fmt.Sprintf("%d", chatID)

	member, err := aep.dependencies.Session.GuildMember(aep.dependencies.GuildID, strChatID)
	if err != nil {
		sp.Error("error getting guild member", zap.Error(err), zap.Int("id", chatID))
		return err
	}

	dRoles := sets.NewStringSet()
	dRoles.FromSlice(member.Roles)

	roles, err := common.GetMembership(ctx, strChatID, aep.dependencies)
	if err != nil {
		return err
	}

	addRoles := roles.Difference(dRoles)
	removeRoles := dRoles.Difference(roles)

	for _, r := range addRoles.ToSlice() {
		filters.QueueUpdate(ctx, payloads.Add, strChatID, r, aep.dependencies)
	}

	for _, r := range removeRoles.ToSlice() {
		filters.QueueUpdate(ctx, payloads.Delete, strChatID, r, aep.dependencies)
	}

	return nil
}

func (aep *authEsiPoller) upsertCharacter(ctx context.Context, characterID, corporationID int32, name, token string) error {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(zap.String("sub-component", "character"))

	var (
		rows *sql.Rows
		err  error
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if token != "" {
		sp.Debug("Updating character", zap.Int32("id", characterID), zap.String("name", name),
			zap.String("token", token), zap.Int32("corporation", corporationID))
		rows, err = aep.dependencies.DB.Insert("characters").
			Columns("id", "name", "token", "corporation_id").
			Values(characterID, name, token, corporationID).
			Suffix("ON CONFLICT (id) DO UPDATE SET name=?, token=?, corporation_id=?", name, token, corporationID).
			QueryContext(ctx)
		if err != nil {
			sp.Error("Error inserting character", zap.Error(err),
				zap.Int32("id", characterID), zap.String("name", name),
				zap.String("token", token), zap.Int32("corporation", corporationID))
		}
	} else {
		rows, err = aep.dependencies.DB.Insert("characters").
			Columns("id", "name", "corporation_id").
			Values(characterID, name, corporationID).
			Suffix("ON CONFLICT (id) DO UPDATE SET name=?, corporation_id=?", name, corporationID).
			QueryContext(ctx)
		if err != nil {
			sp.Error("Error inserting character", zap.Error(err),
				zap.Int32("id", characterID), zap.String("name", name),
				zap.String("token", token), zap.Int32("corporation", corporationID))
		}
	}

	defer func() {
		if rows == nil {
			return
		}

		if err = rows.Close(); err != nil {
			sp.Error("error closing row", zap.Error(err))
		}
	}()

	return err
}
