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
			aep.dependencies.Logger.Errorf("error updating character: %s", err)
			errorCount += 1
			continue
		}

		count += 1
	}

	return count, errorCount, nil
}

func (aep *authEsiPoller) updateCharacter(character auth.Character) error {
	response, _, err := aep.esiClient.ESI.CharacterApi.GetCharactersCharacterId(aep.ctx, character.ID, nil)
	if err != nil {
		aep.dependencies.Logger.Infof("Character not found: %d", character.ID)

		return err
	}

	if response.CorporationId == 0 {
		return fmt.Errorf("CorpID is 0: ESI error most likely, probably transient")
	}

	if character.CorporationID != response.CorporationId {
		aep.dependencies.Logger.Infof("Updating %s to corp %d", character.Name, response.CorporationId)

		// Check if corporation exists, if not, add it.
		corporation := auth.Corporation{ID: response.CorporationId}
		err = aep.updateCorporation(corporation)
		if err != nil {
			return fmt.Errorf("error updating corporation `%s`: %s", corporation.Name, err)
		}

		// We need the old corp Ticker to remove from the filter
		var oldCorp string
		err = aep.dependencies.DB.Select("ticker").
			From("corporations").
			Where(sq.Eq{"id": character.CorporationID}).
			Scan(&oldCorp)
		if err != nil {
			aep.dependencies.Logger.Errorf("error getting old corp info for %d: %s", character.CorporationID, err)
			return err
		}

		err = aep.upsertCharacter(character.ID, response.CorporationId, response.Name, character.Token)
		if err != nil {
			return fmt.Errorf("error upserting character `%d` with corp '%d': %s", character.ID, response.CorporationId, err)
		}

		// We need the new corp Ticker to add to the filter
		var newCorp string
		err = aep.dependencies.DB.Select("ticker").
			From("corporations").
			Where(sq.Eq{"id": response.CorporationId}).
			Scan(&newCorp)
		if err != nil {
			aep.dependencies.Logger.Errorf("error getting old corp info for %d: %s", character.CorporationID, err)
			return err
		}

		// We need the discord user ID
		var discordID string
		err = aep.dependencies.DB.Select("chat_id").
			From("user_character_map").
			Where(sq.Eq{"character_id": character.ID}).
			Scan(&discordID)
		if err != nil {
			aep.dependencies.Logger.Errorf("error getting old corp info for %d: %s", character.CorporationID, err)
			return err
		}

		// I don't like this here because if it fails it will never get cleaned up
		// Change the filter they are in, requires discord ID
		aep.dependencies.Logger.Debugf("removing %s from %s", discordID, oldCorp)
		filters.RemoveMember(discordID, oldCorp, aep.dependencies)
		aep.dependencies.Logger.Debugf("adding %s to %s", discordID, newCorp)
		filters.AddMember(discordID, newCorp, aep.dependencies)
	}

	// We need the chatID of the user, so let's get that.
	var chatID int
	err = aep.dependencies.DB.Select("chat_id").
		From("user_character_map").
		Where(sq.Eq{"character_id": character.ID}).
		Scan(&chatID)
	if err != nil {
		aep.dependencies.Logger.Errorf("error getting chat_id for character %d: %s", character.ID, err)
		return err
	}

	strChatID := fmt.Sprintf("%d", chatID)

	member, err := aep.dependencies.Session.GuildMember(aep.dependencies.GuildID, strChatID)
	if err != nil {
		aep.dependencies.Logger.Errorf("error getting guild member '%d': %s", chatID, err)
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
		aep.dependencies.Logger.Debugf("Updating character: %d, %s, %s, %d", characterID, name, token, corporationID)
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
