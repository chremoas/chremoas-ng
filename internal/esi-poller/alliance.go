package esi_poller

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/chremoas/chremoas-ng/internal/auth"
	"github.com/chremoas/chremoas-ng/internal/roles"
	"go.uber.org/zap"
)

func (aep *authEsiPoller) updateAlliances() (int, int, error) {
	var (
		count      int
		errorCount int
		err        error
		alliance   auth.Alliance
		logger     = aep.logger.With(zap.String("sub-component", "alliance"))
	)

	ctx, cancel := context.WithCancel(aep.ctx)
	defer cancel()

	rows, err := aep.dependencies.DB.Select("id", "name", "ticker").
		From("alliances").
		QueryContext(ctx)
	if err != nil {
		return -1, -1, fmt.Errorf("error getting alliance list from db: %w", err)
	}

	defer func() {
		if err = rows.Close(); err != nil {
			logger.Error("error closing row", zap.Error(err))
		}
	}()

	for rows.Next() {
		err = rows.Scan(&alliance.ID, &alliance.Name, &alliance.Ticker)
		if err != nil {
			logger.Error("error scanning alliance values", zap.Error(err))
			errorCount += 1
			continue
		}

		err = aep.updateAlliance(alliance)
		if err != nil {
			logger.Error("error updating alliance", zap.Error(err),
				zap.String("name", alliance.Name), zap.Int32("id", alliance.ID))
			errorCount += 1
			continue
		}

		count += 1
	}

	return count, errorCount, nil
}

func (aep *authEsiPoller) updateAlliance(alliance auth.Alliance) error {
	logger := aep.logger.With(zap.String("sub-component", "alliance"))

	response, _, err := aep.esiClient.ESI.AllianceApi.GetAlliancesAllianceId(aep.ctx, alliance.ID, nil)
	if err != nil {
		if aep.notFound(err) == nil {
			logger.Info("Alliance not found", zap.Int32("id", alliance.ID))
			roles.Destroy(roles.Role, response.Ticker, aep.dependencies)

			return fmt.Errorf("alliance not found: %d", alliance.ID)
		}

		logger.Error("Error calling GetAlliancesAllianceId", zap.Error(err))
		return err
	}

	if alliance.Name != response.Name || alliance.Ticker != response.Ticker {
		logger.Info("Updating alliance", zap.Int32("id", alliance.ID),
			zap.String("name", response.Name), zap.String("ticker", response.Ticker))
		err = aep.upsertAlliance(alliance.ID, response.Name, response.Ticker)
		if err != nil {
			logger.Error("Error upserting alliance", zap.Error(err), zap.Int32("id", alliance.ID),
				zap.String("name", response.Name), zap.String("ticker", response.Ticker))
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
		logger.Error("error getting count of alliances by name", zap.Error(err),
			zap.String("ticker", response.Ticker))
		return err
	}

	if count == 0 {
		logger.Debug("Adding Alliance", zap.String("ticker", response.Ticker),
			zap.String("name", response.Name))
		roles.Add(roles.Role, false, response.Ticker, response.Name, "discord", aep.dependencies)
	} else {
		logger.Debug("Updating Alliance", zap.String("ticker", response.Ticker),
			zap.String("name", response.Name))
		values := map[string]string{
			"role_nick": response.Ticker,
			"name":      response.Name,
		}
		roles.Update(roles.Role, alliance.Ticker, values, aep.dependencies)
	}

	return nil
}

func (aep *authEsiPoller) upsertAlliance(allianceID int32, name, ticker string) error {
	var (
		err    error
		logger = aep.logger.With(zap.String("sub-component", "alliance"))
	)

	ctx, cancel := context.WithCancel(aep.ctx)
	defer cancel()

	rows, err := aep.dependencies.DB.Insert("alliances").
		Columns("id", "name", "ticker").
		Values(allianceID, name, ticker).
		Suffix("ON CONFLICT (id) DO UPDATE SET name=?, ticker=?", name, ticker).
		QueryContext(ctx)
	if err != nil {
		logger.Error("Error updating alliance", zap.Error(err), zap.Int32("id", allianceID))
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
