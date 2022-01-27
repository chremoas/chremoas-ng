package esi_poller

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/chremoas/chremoas-ng/internal/auth"
	"github.com/chremoas/chremoas-ng/internal/filters"
	"github.com/chremoas/chremoas-ng/internal/roles"
	"go.uber.org/zap"
)

func (aep *authEsiPoller) addCorpMembers(corpTicker string, allianceID int32) {
	var (
		allianceTicker string
		logger         = aep.logger.With(zap.String("sub-component", "corporation"))
	)

	err := aep.dependencies.DB.Select("ticker").
		From("alliances").
		Where(sq.Eq{"id": allianceID}).
		Scan(&allianceTicker)
	if err != nil {
		logger.Error("error getting alliance ticker", zap.Error(err), zap.Int32("id", allianceID))
		return
	}

	members, err := roles.GetRoleMembers(roles.Role, corpTicker, aep.dependencies)
	if err != nil {
		logger.Error("error getting corp member list to add to alliance",
			zap.Error(err), zap.String("ticker", corpTicker))
		return
	}

	for member := range members {
		filters.AddMember(fmt.Sprintf("%d", member), allianceTicker, aep.dependencies)
	}
}

func (aep *authEsiPoller) removeCorpMembers(corpTicker string, allianceID int32) {
	var (
		allianceTicker string
		logger         = aep.logger.With(zap.String("sub-component", "corporation"))
	)

	err := aep.dependencies.DB.Select("ticker").
		From("alliances").
		Where(sq.Eq{"id": allianceID}).
		Scan(&allianceTicker)
	if err != nil {
		logger.Error("error getting alliance ticker", zap.Error(err), zap.Int32("id", allianceID))
		return
	}

	members, err := roles.GetRoleMembers(roles.Role, corpTicker, aep.dependencies)
	if err != nil {
		logger.Error("error getting corp member list to remove from alliance",
			zap.Error(err), zap.String("ticker", corpTicker))
		return
	}

	for member := range members {
		filters.RemoveMember(fmt.Sprintf("%d", member), allianceTicker, aep.dependencies)
	}
}

func (aep *authEsiPoller) updateCorporations() (int, int, error) {
	var (
		count       int
		errorCount  int
		err         error
		corporation auth.Corporation
		logger      = aep.logger.With(zap.String("sub-component", "corporation"))
	)

	ctx, cancel := context.WithCancel(aep.ctx)
	defer cancel()

	rows, err := aep.dependencies.DB.Select("id", "name", "ticker", "alliance_id").
		From("corporations").
		QueryContext(ctx)
	if err != nil {
		return -1, -1, fmt.Errorf("error getting corporation list from db: %w", err)
	}

	defer func() {
		if err = rows.Close(); err != nil {
			logger.Error("error closing row", zap.Error(err))
		}
	}()

	for rows.Next() {
		err = rows.Scan(&corporation.ID, &corporation.Name, &corporation.Ticker, &corporation.AllianceID)
		if err != nil {
			logger.Error("error scanning corporation values", zap.Error(err))
			errorCount += 1
			continue
		}

		err := aep.updateCorporation(corporation)
		if err != nil {
			logger.Error("error updating corporation",
				zap.Error(err), zap.String("name", corporation.Name), zap.Int32("id", corporation.ID))
			errorCount += 1
			continue
		}

		count += 1
	}

	return count, errorCount, nil
}

func (aep *authEsiPoller) updateCorporation(corporation auth.Corporation) error {
	logger := aep.logger.With(zap.String("sub-component", "corporation"))

	response, _, err := aep.esiClient.ESI.CorporationApi.GetCorporationsCorporationId(aep.ctx, corporation.ID, nil)
	if err != nil {
		if aep.notFound(err) == nil {
			logger.Info("Corporation not found", zap.Int32("id", corporation.ID))
			roles.Destroy(roles.Role, response.Ticker, aep.dependencies)

			return fmt.Errorf("corporation not found: %d", corporation.ID)
		}

		logger.Error("Error calling GetCorporationsCorporationId",
			zap.Error(err), zap.String("name", corporation.Name))
	}

	if response.AllianceId != 0 {
		// corp has joined or switched alliances so let's make sure the new alliance is up to date
		logger.Debug("Updating corporation's alliance for %s with allianceId %d",
			zap.String("corporation", response.Name), zap.Int32("alliance", response.AllianceId))

		alliance := auth.Alliance{ID: response.AllianceId}
		err := aep.dependencies.DB.Select("name", "ticker").
			From("alliances").
			Where(sq.Eq{"id": response.AllianceId}).
			Scan(&alliance.Name, &alliance.Ticker)
		if err != nil {
			logger.Error("Error fetching alliance", zap.Error(err), zap.Int32("alliance", response.AllianceId))
		}

		err = aep.updateAlliance(alliance)
		if err != nil {
			return err
		}
	}

	if corporation.Name != response.Name || corporation.Ticker != response.Ticker || corporation.AllianceID.Int32 != response.AllianceId {
		logger.Debug("Updating corporation", zap.Int32("id", corporation.ID),
			zap.String("name", response.Name), zap.String("ticker", response.Ticker))
		err = aep.upsertCorporation(corporation.ID, response.AllianceId, response.Name, response.Ticker)
		if err != nil {
			logger.Error("Error updating alliance", zap.Error(err),
				zap.Int32("alliance id", response.AllianceId), zap.String("corporation", corporation.Name))
			return err
		}
	}

	// Corp has switched or left alliance
	if corporation.AllianceID.Int32 != response.AllianceId {
		// Alliance has changed. Need to remove all members from the old alliance and add them to the new alliance.
		// If there is an old alliance remove corp members from it
		if corporation.AllianceID.Int32 != 0 {
			aep.removeCorpMembers(response.Ticker, corporation.AllianceID.Int32)
		}

		// If there is a new alliance add corp members to it
		if response.AllianceId != 0 {
			aep.addCorpMembers(response.Ticker, response.AllianceId)
		}
	}

	var count int
	err = aep.dependencies.DB.Select("count(id)").
		From("roles").
		Where(sq.Eq{"role_nick": response.Ticker}).
		Where(sq.Eq{"sig": roles.Role}).
		Scan(&count)
	if err != nil {
		logger.Error("error getting count of corporations by name", zap.Error(err),
			zap.String("role_nick", response.Ticker))
		return err
	}

	if count == 0 {
		logger.Debug("Adding Corporation", zap.String("ticker", response.Ticker),
			zap.String("role_nick", response.Ticker), zap.String("name", response.Name))
		roles.Add(roles.Role, false, response.Ticker, response.Name, "discord", aep.dependencies)
	} else {
		logger.Debug("Updating Corporation", zap.String("ticker", response.Ticker))
		values := map[string]string{
			"role_nick": response.Ticker,
			"name":      response.Name,
		}
		roles.Update(roles.Role, corporation.Ticker, values, aep.dependencies)
	}

	return nil
}

func (aep *authEsiPoller) upsertCorporation(corporationID, allianceID int32, name, ticker string) error {
	var (
		err                error
		allianceNullableID sql.NullInt32
		logger             = aep.logger.With(zap.String("sub-component", "corporation"))
	)

	if allianceID != 0 {
		allianceNullableID.Int32 = allianceID
		allianceNullableID.Valid = true
	}

	ctx, cancel := context.WithCancel(aep.ctx)
	defer cancel()

	logger.Debug("Upserting corporation",
		zap.Int32("corporation id", corporationID),
		zap.Int32("alliance id", allianceID),
		zap.String("name", name),
		zap.String("ticker", ticker),
	)

	rows, err := aep.dependencies.DB.Insert("corporations").
		Columns("id", "name", "ticker", "alliance_id").
		Values(corporationID, name, ticker, allianceNullableID).
		Suffix("ON CONFLICT (id) DO UPDATE SET name=?, ticker=?, alliance_id=?", name, ticker, allianceNullableID).
		QueryContext(ctx)
	if err != nil {
		logger.Error("Error upserting corporation %d alliance: %v: %s", zap.Error(err),
			zap.Int32("corporation", corporationID), zap.Any("alliance", allianceNullableID))
	}

	defer func() {
		if rows == nil {
			return
		}

		if err = rows.Close(); err != nil {
			logger.Error("error closing row", zap.Error(err))
		}
	}()

	return err
}
