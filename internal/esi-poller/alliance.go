package esi_poller

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/chremoas/chremoas-ng/internal/auth"
	"github.com/chremoas/chremoas-ng/internal/roles"
)

func (aep *authEsiPoller) updateAlliances() (int, int, error) {
	var (
		count      int
		errorCount int
		err        error
		alliance   auth.Alliance
	)

	rows, err := aep.dependencies.DB.Select("id", "name", "ticker").
		From("alliances").
		Query()
	if err != nil {
		return -1, -1, fmt.Errorf("error getting alliance list from db: %w", err)
	}

	defer func() {
		if err = rows.Close(); err != nil {
			aep.dependencies.Logger.Error(err)
		}
	}()

	for rows.Next() {
		err = rows.Scan(&alliance.ID, &alliance.Name, &alliance.Ticker)
		if err != nil {
			aep.dependencies.Logger.Errorf("error scanning alliance values: %s", err)
			errorCount += 1
			continue
		}

		err = aep.updateAlliance(alliance)
		if err != nil {
			aep.dependencies.Logger.Errorf("error updating alliance: %s", err)
			errorCount += 1
			continue
		}

		count += 1
	}

	return count, errorCount, nil
}

func (aep *authEsiPoller) updateAlliance(alliance auth.Alliance) error {
	response, _, err := aep.esiClient.ESI.AllianceApi.GetAlliancesAllianceId(context.Background(), alliance.ID, nil)
	if err != nil {
		if aep.notFound(err) == nil {
			aep.dependencies.Logger.Infof("Deleting alliance: %d", alliance.ID)
			deleteRows, deleteErr := aep.dependencies.DB.Delete("alliances").
				Where(sq.Eq{"id": alliance.ID}).
				Query()
			if deleteErr != nil {
				aep.dependencies.Logger.Errorf("Error deleting alliance: %s", err)
			}

			deleteErr = deleteRows.Close()
			if deleteErr != nil {
				aep.dependencies.Logger.Errorf("Error closing DB: %s", err)
			}

			roles.Destroy(roles.Role, response.Ticker, aep.dependencies)

			return deleteErr
		}

		aep.dependencies.Logger.Errorf("Error calling GetAlliancesAllianceId: %s", err)
		return err
	}

	if alliance.Name != response.Name || alliance.Ticker != response.Ticker {
		aep.dependencies.Logger.Infof("ESI Poller: Updating alliance: %d with name '%s' and ticker '%s'",
			alliance.ID, response.Name, response.Ticker)
		err = aep.upsertAlliance(alliance.ID, response.Name, response.Ticker)
		if err != nil {
			aep.dependencies.Logger.Errorf("Error upserting alliance: %s", err)
			return err
		}
	}

	var count int
	err = aep.dependencies.DB.Select("count(id)").
		From("roles").
		Where(sq.Eq{"role_nick": response.Ticker}).
		Where(sq.Eq{"sig": roles.Role}).
		Scan(&count)
	if err != nil {
		aep.dependencies.Logger.Errorf("error getting count of alliances by name: %s", err)
		return err
	}

	if count == 0 {
		aep.dependencies.Logger.Infof("Adding Alliance: %s", response.Ticker)
		roles.Add(roles.Role, false, response.Ticker, response.Name, "discord", aep.dependencies)
	} else {
		aep.dependencies.Logger.Infof("Updating Alliance: %s", response.Ticker)
		values := map[string]string{
			"role_nick": response.Ticker,
			"name":      response.Name,
		}
		roles.Update(roles.Role, alliance.Ticker, values, aep.dependencies)
	}

	return nil
}

func (aep *authEsiPoller) upsertAlliance(allianceID int32, name, ticker string) error {
	var err error

	rows, err := aep.dependencies.DB.Insert("alliances").
		Columns("id", "name", "ticker").
		Values(allianceID, name, ticker).
		Suffix("ON CONFLICT (id) DO UPDATE SET name=?, ticker=?", name, ticker).
		Query()
	if err != nil {
		aep.dependencies.Logger.Errorf("ESI Poller: Error updating alliance %d: %s", allianceID, err)
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
