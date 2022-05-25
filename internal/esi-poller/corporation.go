package esi_poller

import (
	"context"
	"database/sql"
	"fmt"

	sl "github.com/bhechinger/spiffylogger"
	"github.com/chremoas/chremoas-ng/internal/filters"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	"github.com/chremoas/chremoas-ng/internal/roles"
	"go.uber.org/zap"
)

func (aep *authEsiPoller) addCorpMembers(ctx context.Context, corpTicker string, allianceID int32) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("sub-component", "corporation"),
		zap.String("corp_ticker", corpTicker),
		zap.Int32("alliance_id", allianceID),
	)

	alliance, err := aep.dependencies.Storage.GetAlliance(ctx, allianceID)
	if err != nil {
		sp.Error("error getting alliance", zap.Error(err))
		return
	}

	sp.With(zap.String("alliance_ticker", alliance.Ticker))

	members, err := roles.GetRoleMembers(ctx, roles.Role, corpTicker, aep.dependencies)
	if err != nil {
		sp.Error("error getting corp member list to add to alliance", zap.Error(err))
		return
	}

	for member := range members {
		filters.AddMember(ctx, fmt.Sprintf("%d", member), alliance.Ticker, aep.dependencies)
	}
}

func (aep *authEsiPoller) removeCorpMembers(ctx context.Context, corpTicker string, allianceID int32) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("sub-component", "corporation"),
		zap.String("corp_ticker", corpTicker),
		zap.Int32("alliance_id", allianceID),
	)

	alliance, err := aep.dependencies.Storage.GetAlliance(ctx, allianceID)
	if err != nil {
		sp.Error("error getting alliance", zap.Error(err))
		return
	}

	sp.With(zap.String("alliance_ticker", alliance.Ticker))

	members, err := roles.GetRoleMembers(ctx, roles.Role, corpTicker, aep.dependencies)
	if err != nil {
		sp.Error("error getting corp member list to remove from alliance", zap.Error(err))
		return
	}

	for member := range members {
		filters.RemoveMember(ctx, fmt.Sprintf("%d", member), alliance.Ticker, aep.dependencies)
	}
}

func (aep *authEsiPoller) updateCorporations(ctx context.Context) (int, int, error) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(zap.String("sub-component", "corporation"))

	var (
		count      int
		errorCount int
		err        error
	)

	corporations, err := aep.dependencies.Storage.GetCorporations(ctx)
	if err != nil {
		return -1, -1, err
	}

	for c := range corporations {
		sp.With(zap.Any("corporation", corporations[c]))

		err := aep.updateCorporation(ctx, corporations[c])
		if err != nil {
			sp.Error("error updating corporation", zap.Error(err))
			errorCount += 1
			continue
		}

		count += 1
	}

	return count, errorCount, nil
}

func (aep *authEsiPoller) updateCorporation(ctx context.Context, corporation payloads.Corporation) error {
	ctx, sp := sl.OpenCorrelatedSpan(ctx, sl.NewID())
	defer sp.Close()

	sp.With(
		zap.String("sub-component", "corporation"),
		zap.Any("corporation", corporation),
	)

	response, _, err := aep.esiClient.ESI.CorporationApi.GetCorporationsCorporationId(ctx, corporation.ID, nil)
	if err != nil {
		if aep.notFound(ctx, err) == nil {
			sp.Info("Corporation not found")
			roles.Destroy(ctx, roles.Role, response.Ticker, aep.dependencies)

			return fmt.Errorf("corporation not found: %d", corporation.ID)
		}

		sp.Error("Error calling GetCorporationsCorporationId", zap.Error(err))
	}

	sp.With(zap.Any("esi_response", response))

	if response.AllianceId != 0 {
		// corp has joined or switched alliances so let's make sure the new alliance is up-to-date
		sp.Debug("Updating corporation's alliance")

		alliance, err := aep.dependencies.Storage.GetAlliance(ctx, response.AllianceId)
		if err != nil {
			sp.Warn("Error getting alliance", zap.String("sql.ErrNoRows", sql.ErrNoRows.Error()), zap.Error(err), zap.Any("error_type", fmt.Sprintf("%T", err)))

			if err != sql.ErrNoRows {
				alliance = payloads.Alliance{ID: response.AllianceId}
			} else {
				sp.Error("error getting alliance", zap.Error(err))
				return err
			}
		}

		sp.With(zap.Any("alliance", alliance))

		err = aep.updateAlliance(ctx, alliance)
		if err != nil {
			sp.Error("error updating alliance", zap.Error(err))
			return err
		}
	}

	if corporation.Name != response.Name || corporation.Ticker != response.Ticker || corporation.AllianceID.Int32 != response.AllianceId {
		sp.Debug("Updating corporation")
		err = aep.dependencies.Storage.UpsertCorporation(ctx, corporation.ID, response.AllianceId, response.Name, response.Ticker)
		if err != nil {
			sp.Error("Error updating alliance", zap.Error(err))
			return err
		}
	}

	// Corp has switched or left alliance
	if corporation.AllianceID.Int32 != response.AllianceId {
		// Alliance has changed. Need to remove all members from the old alliance and add them to the new alliance.
		// If there is an old alliance remove corp members from it
		if corporation.AllianceID.Int32 != 0 {
			aep.removeCorpMembers(ctx, response.Ticker, corporation.AllianceID.Int32)
		}

		// If there is a new alliance add corp members to it
		if response.AllianceId != 0 {
			aep.addCorpMembers(ctx, response.Ticker, response.AllianceId)
		}
	}

	count, err := aep.dependencies.Storage.GetRoleCount(ctx, roles.Role, response.Ticker)
	if err != nil {
		return err
	}

	sp.With(zap.Int("count", count))

	if count == 0 {
		sp.Debug("Adding Corporation")
		roles.Add(ctx, roles.Role, false, response.Ticker, response.Name, "discord", aep.dependencies)
	} else {
		sp.Debug("Updating Corporation")
		values := map[string]string{
			"role_nick": response.Ticker,
			"name":      response.Name,
		}
		roles.Update(ctx, roles.Role, corporation.Ticker, values, aep.dependencies)
	}

	return nil
}
