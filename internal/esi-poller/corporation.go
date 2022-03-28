package esi_poller

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	sl "github.com/bhechinger/spiffylogger"
	"github.com/chremoas/chremoas-ng/internal/auth"
	"github.com/chremoas/chremoas-ng/internal/filters"
	"github.com/chremoas/chremoas-ng/internal/roles"
	"go.uber.org/zap"
)

func (aep *authEsiPoller) addCorpMembers(ctx context.Context, corpTicker string, allianceID int32) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(zap.String("sub-component", "corporation"))

	var allianceTicker string

	query := aep.dependencies.DB.Select("ticker").
		From("alliances").
		Where(sq.Eq{"id": allianceID})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	err = query.Scan(&allianceTicker)
	if err != nil {
		sp.Error("error getting alliance ticker", zap.Error(err), zap.Int32("id", allianceID))
		return
	}

	members, err := roles.GetRoleMembers(ctx, roles.Role, corpTicker, aep.dependencies)
	if err != nil {
		sp.Error("error getting corp member list to add to alliance",
			zap.Error(err), zap.String("ticker", corpTicker))
		return
	}

	for member := range members {
		filters.AddMember(ctx, fmt.Sprintf("%d", member), allianceTicker, aep.dependencies)
	}
}

func (aep *authEsiPoller) removeCorpMembers(ctx context.Context, corpTicker string, allianceID int32) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(zap.String("sub-component", "corporation"))

	var allianceTicker string

	query := aep.dependencies.DB.Select("ticker").
		From("alliances").
		Where(sq.Eq{"id": allianceID})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	err = query.Scan(&allianceTicker)
	if err != nil {
		sp.Error("error getting alliance ticker", zap.Error(err), zap.Int32("id", allianceID))
		return
	}

	members, err := roles.GetRoleMembers(ctx, roles.Role, corpTicker, aep.dependencies)
	if err != nil {
		sp.Error("error getting corp member list to remove from alliance",
			zap.Error(err), zap.String("ticker", corpTicker))
		return
	}

	for member := range members {
		filters.RemoveMember(ctx, fmt.Sprintf("%d", member), allianceTicker, aep.dependencies)
	}
}

func (aep *authEsiPoller) updateCorporations(ctx context.Context) (int, int, error) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(zap.String("sub-component", "corporation"))

	var (
		count       int
		errorCount  int
		err         error
		corporation auth.Corporation
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := aep.dependencies.DB.Select("id", "name", "ticker", "alliance_id").
		From("corporations")

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	rows, err := query.QueryContext(ctx)
	if err != nil {
		return -1, -1, fmt.Errorf("error getting corporation list from db: %w", err)
	}

	defer func() {
		if err = rows.Close(); err != nil {
			sp.Error("error closing row", zap.Error(err))
		}
	}()

	for rows.Next() {
		err = rows.Scan(&corporation.ID, &corporation.Name, &corporation.Ticker, &corporation.AllianceID)
		if err != nil {
			sp.Error("error scanning corporation values", zap.Error(err))
			errorCount += 1
			continue
		}

		err := aep.updateCorporation(ctx, corporation)
		if err != nil {
			sp.Error("error updating corporation",
				zap.Error(err), zap.String("name", corporation.Name), zap.Int32("id", corporation.ID))
			errorCount += 1
			continue
		}

		count += 1
	}

	return count, errorCount, nil
}

func (aep *authEsiPoller) updateCorporation(_ context.Context, corporation auth.Corporation) error {
	ctx, sp := sl.OpenSpan(context.Background())
	defer sp.Close()

	sp.With(zap.String("sub-component", "corporation"))

	response, _, err := aep.esiClient.ESI.CorporationApi.GetCorporationsCorporationId(ctx, corporation.ID, nil)
	if err != nil {
		if aep.notFound(ctx, err) == nil {
			sp.Info("Corporation not found", zap.Int32("id", corporation.ID))
			roles.Destroy(ctx, roles.Role, response.Ticker, aep.dependencies)

			return fmt.Errorf("corporation not found: %d", corporation.ID)
		}

		sp.Error("Error calling GetCorporationsCorporationId",
			zap.Error(err), zap.String("name", corporation.Name))
	}

	if response.AllianceId != 0 {
		// corp has joined or switched alliances so let's make sure the new alliance is up to date
		sp.Debug("Updating corporation's alliance",
			zap.String("corporation", response.Name), zap.Int32("alliance", response.AllianceId))

		alliance := auth.Alliance{ID: response.AllianceId}
		query := aep.dependencies.DB.Select("name", "ticker").
			From("alliances").
			Where(sq.Eq{"id": response.AllianceId})

		sqlStr, args, err := query.ToSql()
		if err != nil {
			sp.Error("error getting sql", zap.Error(err))
		} else {
			sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
		}

		err = query.Scan(&alliance.Name, &alliance.Ticker)
		if err != nil {
			sp.Error("Error fetching alliance", zap.Error(err), zap.Int32("alliance", response.AllianceId))
		}

		err = aep.updateAlliance(ctx, alliance)
		if err != nil {
			return err
		}
	}

	if corporation.Name != response.Name || corporation.Ticker != response.Ticker || corporation.AllianceID.Int32 != response.AllianceId {
		sp.Debug("Updating corporation", zap.Int32("id", corporation.ID),
			zap.String("name", response.Name), zap.String("ticker", response.Ticker))
		err = aep.upsertCorporation(ctx, corporation.ID, response.AllianceId, response.Name, response.Ticker)
		if err != nil {
			sp.Error("Error updating alliance", zap.Error(err),
				zap.Int32("alliance id", response.AllianceId), zap.String("corporation", corporation.Name))
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
		sp.Error("error getting count of corporations by name", zap.Error(err),
			zap.String("role_nick", response.Ticker))
		return err
	}

	if count == 0 {
		sp.Debug("Adding Corporation", zap.String("ticker", response.Ticker),
			zap.String("role_nick", response.Ticker), zap.String("name", response.Name))
		roles.Add(ctx, roles.Role, false, response.Ticker, response.Name, "discord", aep.dependencies)
	} else {
		sp.Debug("Updating Corporation", zap.String("ticker", response.Ticker))
		values := map[string]string{
			"role_nick": response.Ticker,
			"name":      response.Name,
		}
		roles.Update(ctx, roles.Role, corporation.Ticker, values, aep.dependencies)
	}

	return nil
}

func (aep *authEsiPoller) upsertCorporation(ctx context.Context, corporationID, allianceID int32, name, ticker string) error {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(zap.String("sub-component", "corporation"))

	var (
		err                error
		allianceNullableID sql.NullInt32
	)

	if allianceID != 0 {
		allianceNullableID.Int32 = allianceID
		allianceNullableID.Valid = true
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sp.Debug("Upserting corporation",
		zap.Int32("corporation id", corporationID),
		zap.Int32("alliance id", allianceID),
		zap.String("name", name),
		zap.String("ticker", ticker),
	)

	insert := aep.dependencies.DB.Insert("corporations").
		Columns("id", "name", "ticker", "alliance_id").
		Values(corporationID, name, ticker, allianceNullableID).
		Suffix("ON CONFLICT (id) DO UPDATE SET name=?, ticker=?, alliance_id=?", name, ticker, allianceNullableID)

	sqlStr, args, err := insert.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	rows, err := insert.QueryContext(ctx)
	if err != nil {
		sp.Error("Error upserting corporation %d alliance: %v: %s", zap.Error(err),
			zap.Int32("corporation", corporationID), zap.Any("alliance", allianceNullableID))
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
