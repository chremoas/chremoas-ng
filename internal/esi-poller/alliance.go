package esi_poller

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	sl "github.com/bhechinger/spiffylogger"
	"github.com/chremoas/chremoas-ng/internal/auth"
	"github.com/chremoas/chremoas-ng/internal/roles"
	"go.uber.org/zap"
)

func (aep *authEsiPoller) updateAlliances(ctx context.Context) (int, int, error) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(zap.String("sub-component", "alliance"))

	var (
		count      int
		errorCount int
		err        error
		alliance   auth.Alliance
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := aep.dependencies.DB.Select("id", "name", "ticker").
		From("alliances")

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	rows, err := query.QueryContext(ctx)
	if err != nil {
		return -1, -1, fmt.Errorf("error getting alliance list from db: %w", err)
	}

	defer func() {
		if err = rows.Close(); err != nil {
			sp.Error("error closing row", zap.Error(err))
		}
	}()

	for rows.Next() {
		err = rows.Scan(&alliance.ID, &alliance.Name, &alliance.Ticker)
		if err != nil {
			sp.Error("error scanning alliance values", zap.Error(err))
			errorCount += 1
			continue
		}

		err = aep.updateAlliance(ctx, alliance)
		if err != nil {
			sp.Error("error updating alliance", zap.Error(err),
				zap.String("name", alliance.Name), zap.Int32("id", alliance.ID))
			errorCount += 1
			continue
		}

		count += 1
	}

	return count, errorCount, nil
}

func (aep *authEsiPoller) updateAlliance(ctx context.Context, alliance auth.Alliance) error {
	ctx, sp := sl.OpenCorrelatedSpan(ctx, sl.NewID())
	defer sp.Close()

	sp.With(zap.String("sub-component", "alliance"))

	response, _, err := aep.esiClient.ESI.AllianceApi.GetAlliancesAllianceId(ctx, alliance.ID, nil)
	if err != nil {
		if aep.notFound(ctx, err) == nil {
			sp.Info("Alliance not found", zap.Int32("id", alliance.ID))
			roles.Destroy(ctx, roles.Role, response.Ticker, aep.dependencies)

			return fmt.Errorf("alliance not found: %d", alliance.ID)
		}

		sp.Error("Error calling GetAlliancesAllianceId", zap.Error(err))
		return err
	}

	if alliance.Name != response.Name || alliance.Ticker != response.Ticker {
		sp.Info("Updating alliance", zap.Int32("id", alliance.ID),
			zap.String("name", response.Name), zap.String("ticker", response.Ticker))
		err = aep.upsertAlliance(ctx, alliance.ID, response.Name, response.Ticker)
		if err != nil {
			sp.Error("Error upserting alliance", zap.Error(err), zap.Int32("id", alliance.ID),
				zap.String("name", response.Name), zap.String("ticker", response.Ticker))
			return err
		}
	}

	var count int
	query := aep.dependencies.DB.Select("count(id)").
		From("roles").
		Where(sq.Eq{"role_nick": response.Ticker}).
		Where(sq.Eq{"sig": roles.Role})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	err = query.Scan(&count)
	if err != nil {
		sp.Error("error getting count of alliances by name", zap.Error(err),
			zap.String("ticker", response.Ticker))
		return err
	}

	if count == 0 {
		sp.Debug("Adding Alliance", zap.String("ticker", response.Ticker),
			zap.String("name", response.Name))
		roles.Add(ctx, roles.Role, false, response.Ticker, response.Name, "discord", aep.dependencies)
	} else {
		sp.Debug("Updating Alliance", zap.String("ticker", response.Ticker),
			zap.String("name", response.Name))
		values := map[string]string{
			"role_nick": response.Ticker,
			"name":      response.Name,
		}
		roles.Update(ctx, roles.Role, alliance.Ticker, values, aep.dependencies)
	}

	return nil
}

func (aep *authEsiPoller) upsertAlliance(ctx context.Context, allianceID int32, name, ticker string) error {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(zap.String("sub-component", "alliance"))

	var err error

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	insert := aep.dependencies.DB.Insert("alliances").
		Columns("id", "name", "ticker").
		Values(allianceID, name, ticker).
		Suffix("ON CONFLICT (id) DO UPDATE SET name=?, ticker=?", name, ticker)

	sqlStr, args, err := insert.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	rows, err := insert.QueryContext(ctx)
	if err != nil {
		sp.Error("Error updating alliance", zap.Error(err), zap.Int32("id", allianceID))
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
