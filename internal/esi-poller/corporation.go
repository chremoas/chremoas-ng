package esi_poller

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/chremoas/chremoas-ng/internal/auth"
	"github.com/chremoas/chremoas-ng/internal/filters"
	"github.com/chremoas/chremoas-ng/internal/roles"
)

func (aep *authEsiPoller) addCorpMembers(corpTicker string, allianceID int32) {
	var allianceTicker string

	err := aep.dependencies.DB.Select("ticker").
		From("alliances").
		Where(sq.Eq{"id": allianceID}).
		Scan(&allianceTicker)
	if err != nil {
		aep.dependencies.Logger.Errorf("error getting alliance ticker for %d: %s", allianceID, err)
		return
	}

	members, err := roles.GetRoleMembers(roles.Role, corpTicker, aep.dependencies)
	if err != nil {
		aep.dependencies.Logger.Errorf("error getting corp member list to add to alliance: %s", err)
		return
	}

	for member := range members {
		filters.AddMember(fmt.Sprintf("%d", member), allianceTicker, aep.dependencies)
	}
}

func (aep *authEsiPoller) removeCorpMembers(corpTicker string, allianceID int32) {
	var allianceTicker string

	err := aep.dependencies.DB.Select("ticker").
		From("alliances").
		Where(sq.Eq{"id": allianceID}).
		Scan(&allianceTicker)
	if err != nil {
		aep.dependencies.Logger.Errorf("error getting alliance ticker for %d: %s", allianceID, err)
		return
	}

	members, err := roles.GetRoleMembers(roles.Role, corpTicker, aep.dependencies)
	if err != nil {
		aep.dependencies.Logger.Errorf("error getting corp member list to remove from alliance: %s", err)
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
			aep.dependencies.Logger.Error(err)
		}
	}()

	for rows.Next() {
		err = rows.Scan(&corporation.ID, &corporation.Name, &corporation.Ticker, &corporation.AllianceID)
		if err != nil {
			aep.dependencies.Logger.Errorf("error scanning corporation values: %s", err)
			errorCount += 1
			continue
		}

		err := aep.updateCorporation(corporation)
		if err != nil {
			aep.dependencies.Logger.Errorf("error scanning corporation values: %s", err)
			errorCount += 1
			continue
		}

		count += 1
	}

	return count, errorCount, nil
}

func (aep *authEsiPoller) updateCorporation(corporation auth.Corporation) error {
	response, _, err := aep.esiClient.ESI.CorporationApi.GetCorporationsCorporationId(aep.ctx, corporation.ID, nil)
	if err != nil {
		if aep.notFound(err) == nil {
			aep.dependencies.Logger.Infof("Corporation not found: %d", corporation.ID)
			roles.Destroy(roles.Role, response.Ticker, aep.dependencies)

			return fmt.Errorf("corporation not found: %d", corporation.ID)
		}

		aep.dependencies.Logger.Errorf("Error calling GetCorporationsCorporationId: %s", err)
	}

	if corporation.Name != response.Name || corporation.Ticker != response.Ticker || corporation.AllianceID.Int32 != response.AllianceId {
		aep.dependencies.Logger.Debugf("ESI Poller: Updating corporation: %d with name '%s' and ticker '%s'", corporation.ID, response.Name, response.Ticker)
		err = aep.upsertCorporation(corporation.ID, response.AllianceId, response.Name, response.Ticker)
		if err != nil {
			aep.dependencies.Logger.Errorf("Error updating alliance '%d' for corp '%s': %s", response.AllianceId, corporation.Name, err)
			return err
		}
	}

	// Corp has switched or left alliance
	if corporation.AllianceID.Int32 != response.AllianceId {
		if response.AllianceId != 0 {
			// corp has joined or switched alliances so let's make sure the new alliance is up to date
			aep.dependencies.Logger.Debugf("ESI Poller: Updating corporation's alliance for %s with allianceId %d\n", response.Name, response.AllianceId)

			alliance := auth.Alliance{ID: response.AllianceId}
			err := aep.dependencies.DB.Select("name", "ticker").
				From("alliances").
				Where(sq.Eq{"id": response.AllianceId}).
				Scan(&alliance.Name, &alliance.Ticker)
			if err != nil {
				aep.dependencies.Logger.Errorf("Error fetching alliance: %s", err)
			}

			err = aep.updateAlliance(alliance)
			if err != nil {
				return err
			}
		}

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
		aep.dependencies.Logger.Errorf("error getting count of corporations by name: %s", err)
		return err
	}

	if count == 0 {
		aep.dependencies.Logger.Debugf("Adding Corporation: %s", response.Ticker)
		roles.Add(roles.Role, false, response.Ticker, response.Name, "discord", aep.dependencies)
	} else {
		aep.dependencies.Logger.Debugf("Updating Corporation: %s", response.Ticker)
		values := map[string]string{
			"role_nick": response.Ticker,
			"name":      response.Name,
		}
		roles.Update(roles.Role, corporation.Ticker, values, aep.dependencies)
	}

	return nil
}

func (aep *authEsiPoller) upsertCorporation(corporationID, allianceID int32, name, ticker string) error {
	var err error

	ctx, cancel := context.WithCancel(aep.ctx)
	defer cancel()

	rows, err := aep.dependencies.DB.Insert("corporations").
		Columns("id", "name", "ticker", "alliance_id").
		Values(corporationID, name, ticker, allianceID).
		Suffix("ON CONFLICT (id) DO UPDATE SET name=?, ticker=?, alliance_id=?", name, ticker, allianceID).
		QueryContext(ctx)
	if err != nil {
		aep.dependencies.Logger.Errorf("ESI Poller: Error inserting corporation %d: %s", corporationID, err)
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
