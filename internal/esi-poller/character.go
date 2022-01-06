package esi_poller

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/bhechinger/go-sets"
	"github.com/chremoas/chremoas-ng/internal/auth"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/filters"
	"github.com/chremoas/chremoas-ng/internal/payloads"
)

func (aep *authEsiPoller) updateCharacters() (int, int, error) {
	var (
		count      int
		errorCount int
		err        error
		character  auth.Character
	)

	ctx, cancel := context.WithCancel(aep.ctx)
	defer cancel()

	rows, err := aep.dependencies.DB.Select("id", "name", "corporation_id", "token").
		From("characters").
		QueryContext(ctx)
	if err != nil {
		return -1, -1, fmt.Errorf("error getting character list from the db: %w", err)
	}

	defer func() {
		if err = rows.Close(); err != nil {
			aep.dependencies.Logger.Error(err)
		}
	}()

	for rows.Next() {
		err = rows.Scan(&character.ID, &character.Name, &character.CorporationID, &character.Token)
		if err != nil {
			return -1, -1, fmt.Errorf("error scanning character values: %w", err)
		}

		err := aep.updateCharacter(character)
		if err != nil {
			aep.dependencies.Logger.Errorf("error updating alliance: %s", err)
			errorCount += 1
			continue
		}

		count += 1
	}

	return count, errorCount, nil
}

func (aep *authEsiPoller) updateCharacter(character auth.Character) error {
	ctx, cancel := context.WithCancel(aep.ctx)
	defer cancel()

	response, _, err := aep.esiClient.ESI.CharacterApi.GetCharactersCharacterId(aep.ctx, character.ID, nil)
	if err != nil {
		aep.dependencies.Logger.Infof("Character not found: %d", character.ID)

		return err
	}

	if response.CorporationId == 0 {
		return fmt.Errorf("CorpID is 0: ESI error most likely, probably transient")
	}

	if character.Name != response.Name || character.CorporationID != response.CorporationId {
		aep.dependencies.Logger.Infof("ESI Poller: Updating character: %d with name '%s' and corporation '%d'", character.ID, response.Name, response.CorporationId)
		err = aep.upsertCharacter(character.ID, response.CorporationId, response.Name, character.Token)
		if err != nil {
			return fmt.Errorf("error upserting character `%d`: %s", character.ID, err)
		}
	}

	if character.CorporationID != response.CorporationId {
		aep.dependencies.Logger.Infof("Updating %s to corp %d", character.Name, response.CorporationId)

		// Check if corporation exists, if not, add it.
		corporation := auth.Corporation{ID: response.CorporationId}
		err := aep.dependencies.DB.Select("name, alliance_id", "ticker").
			From("corporations").
			Where(sq.Eq{"id": response.CorporationId}).
			Scan(&corporation.Name, &corporation.AllianceID, &corporation.Ticker)
		if err != nil {
			aep.dependencies.Logger.Errorf("error getting corp info for %d: %s", response.CorporationId, err)
			return err
		}

		err = aep.updateCorporation(corporation)

		updateRows, err := aep.dependencies.DB.Update("characters").
			Set("corporation_id", response.CorporationId).
			Where(sq.Eq{"id": character.ID}).
			QueryContext(ctx)
		if err != nil {
			aep.dependencies.Logger.Errorf("Error updating character: %s", err)
		}

		if updateRows == nil {
			aep.dependencies.Logger.Info("updateRows was nil")
		} else {
			err = updateRows.Close()
			if err != nil {
				aep.dependencies.Logger.Errorf("Error closing DB: %s", err)
			}
		}
	}

	// We need the chatID of the user, so let's get that.
	var chatID int
	err = aep.dependencies.DB.Select("chat_id").
		From("user_character_map").
		Where(sq.Eq{"character_id": character.ID}).
		Scan(&chatID)
	if err != nil {
		aep.dependencies.Logger.Errorf("error getting chat_id from %d: %s", character.ID, err)
		return err
	}

	strChatID := fmt.Sprintf("%d", chatID)

	member, err := aep.dependencies.Session.GuildMember(aep.dependencies.GuildID, strChatID)
	if err != nil {
		aep.dependencies.Logger.Errorf("error getting guild member '%d': %s", chatID, err)
		if err != nil {
			aep.dependencies.Logger.Errorf("Error Acking message: %s", err)
		}
		return err
	}

	dRoles := sets.NewStringSet()
	dRoles.FromSlice(member.Roles)

	roles, err := common.GetMembership(strChatID, aep.dependencies)
	if err != nil {
		return err
	}

	addRoles := roles.Difference(dRoles)
	removeRoles := dRoles.Difference(roles)

	for _, r := range addRoles.ToSlice() {
		filters.QueueUpdate(payloads.Add, strChatID, r, aep.dependencies)
	}

	for _, r := range removeRoles.ToSlice() {
		filters.QueueUpdate(payloads.Delete, strChatID, r, aep.dependencies)
	}

	return nil
}

func (aep *authEsiPoller) upsertCharacter(characterID, corporationID int32, name, token string) error {
	var (
		rows *sql.Rows
		err  error
	)

	ctx, cancel := context.WithCancel(aep.ctx)
	defer cancel()

	if token != "" {
		rows, err = aep.dependencies.DB.Insert("characters").
			Columns("id", "name", "token", "corporation_id").
			Values(characterID, name, token, corporationID).
			Suffix("ON CONFLICT (id) DO UPDATE SET name=?, token=?, corporation_id=?", name, token, corporationID).
			QueryContext(ctx)
		if err != nil {
			aep.dependencies.Logger.Errorf("ESI Poller: Error inserting character %d: %s", characterID, err)
		}
	} else {
		rows, err = aep.dependencies.DB.Insert("characters").
			Columns("id", "name", "corporation_id").
			Values(characterID, name, corporationID).
			Suffix("ON CONFLICT (id) DO UPDATE SET name=?, corporation_id=?", name, corporationID).
			QueryContext(ctx)
		if err != nil {
			aep.dependencies.Logger.Errorf("ESI Poller: Error inserting character %d: %s", characterID, err)
		}
	}

	defer func() {
		if rows == nil {
			return
		}

		if err = rows.Close(); err != nil {
			aep.dependencies.Logger.Error(err)
		}
	}()

	return err
}
