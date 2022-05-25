package esi_poller

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/bhechinger/go-sets"
	sl "github.com/bhechinger/spiffylogger"
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
	)

	characters, err := aep.dependencies.Storage.GetCharacters(ctx)
	if err != nil {
		return -1, -1, err
	}

	for c := range characters {
		sp.With(zap.Any("character", characters[c]))

		err = aep.updateCharacter(ctx, characters[c])
		if err != nil {
			discordID, err := aep.dependencies.Storage.GetDiscordUser(ctx, characters[c].ID)
			if err != nil {
				if err == sql.ErrNoRows {
					// character is no longer associated with a discord user so we're going do delete it.
					sp.Warn("Deleting character as they have no associated discord user")
					err = aep.dependencies.Storage.DeleteCharacter(ctx, characters[c].ID)
					if err != nil {
						sp.Error("Error deleting character", zap.Error(err))
						return -1, -1, err
					}
				}
				sp.Error("Error getting discord user", zap.Error(err))
				return -1, -1, err
			}

			handled, hErr := aep.cad.CheckAndDelete(ctx, discordID, err)
			if hErr != nil {
				sp.Error("Additional errors from checkAndDelete", zap.Error(hErr))
			}
			if handled {
				return -1, -1, err
			}

			sp.Error(
				"error updating character",
				zap.Error(err),
				zap.NamedError("hErr", hErr),
				zap.Int32("id", characters[c].ID),
				zap.String("name", characters[c].Name),
			)
			errorCount += 1
			continue
		}

		count += 1
	}

	return count, errorCount, nil
}

func (aep *authEsiPoller) updateCharacter(ctx context.Context, character payloads.Character) error {
	ctx, sp := sl.OpenCorrelatedSpan(ctx, sl.NewID())
	defer sp.Close()

	sp.With(
		zap.String("sub-component", "character"),
		zap.Any("character", character),
	)

	response, _, err := aep.esiClient.ESI.CharacterApi.GetCharactersCharacterId(ctx, character.ID, nil)
	if err != nil {
		sp.Error("Character not found")
		return err
	}

	if response.CorporationId == 0 {
		sp.Error("CorpID is 0: ESI Error most likely, probably transient")
		return fmt.Errorf("CorpID is 0: ESI error most likely, probably transient")
	}

	sp.With(zap.Any("esi_response", response))

	if character.CorporationID != response.CorporationId {
		sp.Info("Updating character's corp")

		// Check if corporation exists, if not, add it.
		corporation := payloads.Corporation{ID: response.CorporationId}
		err = aep.updateCorporation(ctx, corporation)
		if err != nil {
			sp.Error("error updating corporation", zap.Error(err))
			return err
		}

		// We need the old corp Ticker to remove from the filter
		oldCorp, err := aep.dependencies.Storage.GetCorporation(ctx, character.CorporationID)
		if err != nil {
			sp.Error(
				"error getting old corp info",
				zap.Error(err),
				zap.String("character", character.Name),
				zap.Int32("corporation", character.CorporationID),
			)
			return err
		}

		sp.With(
			zap.Any("old_corp", oldCorp),
		)

		err = aep.dependencies.Storage.UpsertCharacter(ctx, character.ID, response.CorporationId, response.Name, character.Token)
		if err != nil {
			sp.Error("error upserting character", zap.Error(err))
			return err
		}

		// We need the new corp Ticker to add to the filter
		newCorp, err := aep.dependencies.Storage.GetCorporation(ctx, response.CorporationId)
		if err != nil {
			sp.Error(
				"error getting new corp info",
				zap.Error(err),
				zap.String("character", character.Name),
				zap.Int32("corporation", character.CorporationID),
			)
			return err
		}

		sp.With(
			zap.Any("new_corp", newCorp),
		)

		// We need the discord user ID
		discordID, err := aep.dependencies.Storage.GetDiscordUser(ctx, character.ID)

		sp.With(zap.String("discord_id", discordID))

		// I don't like this here because if it fails it will never get cleaned up
		// Change the filter they are in, requires discord ID
		sp.Debug("removing user from corp")
		filters.RemoveMember(ctx, discordID, oldCorp.Ticker, aep.dependencies)
		sp.Debug("adding user to corp")
		filters.AddMember(ctx, discordID, newCorp.Ticker, aep.dependencies)

		if newCorp.AllianceID != oldCorp.AllianceID {
			// new corp is in a different alliance, gotta switch those.
			oldAlliance, err := aep.dependencies.Storage.GetAlliance(ctx, oldCorp.AllianceID.Int32)
			if err != nil {
				sp.Error("error getting old alliance ticker", zap.Error(err))
				return err
			}

			sp.With(zap.Any("old_alliance", oldAlliance))
			sp.Debug("removing user from alliance")
			filters.RemoveMember(ctx, discordID, oldAlliance.Ticker, aep.dependencies)

			newAlliance, err := aep.dependencies.Storage.GetAlliance(ctx, newCorp.AllianceID.Int32)
			if err != nil {
				sp.Error("error getting new alliance ticker", zap.Error(err))
				return err
			}

			sp.With(zap.Any("new_alliance", newAlliance))
			sp.Debug("adding user to alliance")
			filters.AddMember(ctx, discordID, newAlliance.Ticker, aep.dependencies)
		}
	}

	// We need the chatID of the user, so let's get that.
	chatID, err := aep.dependencies.Storage.GetDiscordUser(ctx, character.ID)

	sp.With(zap.String("chat_id", chatID))

	member, err := aep.dependencies.Session.GuildMember(aep.dependencies.GuildID, chatID)
	if err != nil {
		discordID, err := aep.dependencies.Storage.GetDiscordUser(ctx, character.ID)
		if err == nil {
			return nil
		}

		handled, hErr := aep.cad.CheckAndDelete(ctx, discordID, err)
		if hErr != nil {
			sp.Error("Additional errors from checkAndDelete", zap.Error(hErr))
		}
		if handled {
			return err
		}

		sp.Error("error getting guild member", zap.Error(err), zap.NamedError("hErr", hErr))
		return err
	}

	dRoles := sets.NewStringSet()
	dRoles.FromSlice(member.Roles)

	roles, err := common.GetMembership(ctx, chatID, aep.dependencies)
	if err != nil {
		sp.Error("error getting membership", zap.Error(err))
		return err
	}

	addRoles := roles.Difference(dRoles)
	removeRoles := dRoles.Difference(roles)

	for _, r := range addRoles.ToSlice() {
		filters.QueueUpdate(ctx, payloads.Add, chatID, r, aep.dependencies)
	}

	for _, r := range removeRoles.ToSlice() {
		filters.QueueUpdate(ctx, payloads.Delete, chatID, r, aep.dependencies)
	}

	return nil
}
