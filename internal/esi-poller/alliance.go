package esi_poller

import (
	"context"
	"fmt"

	sl "github.com/bhechinger/spiffylogger"
	"github.com/chremoas/chremoas-ng/internal/payloads"
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
	)

	alliances, err := aep.dependencies.Storage.GetAlliances(ctx)
	if err != nil {
		return -1, -1, err
	}

	for a := range alliances {
		sp.With(zap.Any("alliance", alliances[a]))

		err = aep.updateAlliance(ctx, alliances[a])
		if err != nil {
			sp.Error("error updating alliance", zap.Error(err))
			errorCount += 1
			continue
		}

		count += 1
	}

	return count, errorCount, nil
}

func (aep *authEsiPoller) updateAlliance(ctx context.Context, alliance payloads.Alliance) error {
	ctx, sp := sl.OpenCorrelatedSpan(ctx, sl.NewID())
	defer sp.Close()

	sp.With(
		zap.Any("alliance", alliance),
		zap.String("sub-component", "alliance"),
	)

	response, _, err := aep.esiClient.ESI.AllianceApi.GetAlliancesAllianceId(ctx, alliance.ID, nil)
	if err != nil {
		if aep.notFound(ctx, err) == nil {
			sp.Info("Alliance not found")
			roles.Destroy(ctx, roles.Role, response.Ticker, aep.dependencies)

			sp.Error("alliance not found")
			return fmt.Errorf("alliance not found: %d", alliance.ID)
		}

		sp.Error("Error calling GetAlliancesAllianceId", zap.Error(err))
		return err
	}

	sp.With(zap.Any("esi_response", response))

	if alliance.Name != response.Name || alliance.Ticker != response.Ticker {
		sp.Info("Updating alliance")
		err = aep.dependencies.Storage.UpsertAlliance(ctx, alliance.ID, response.Name, response.Ticker)
		if err != nil {
			sp.Error("Error upserting alliance", zap.Error(err))
			return err
		}
	}

	count, err := aep.dependencies.Storage.GetRoleCount(ctx, roles.Role, response.Ticker)
	if err != nil {
		return err
	}

	sp.With(zap.Int("count", count))

	if count == 0 {
		sp.Debug("Adding Alliance")
		roles.Add(ctx, roles.Role, false, response.Ticker, response.Name, "discord", aep.dependencies)
	} else {
		sp.Debug("Updating Alliance")
		values := map[string]string{
			"role_nick": response.Ticker,
			"name":      response.Name,
		}
		roles.Update(ctx, roles.Role, alliance.Ticker, values, aep.dependencies)
	}

	return nil
}
