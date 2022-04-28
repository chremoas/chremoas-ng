package esi_poller

import (
	"context"
	"errors"
	"time"

	"github.com/antihax/goesi"
	"github.com/antihax/goesi/esi"
	sl "github.com/bhechinger/spiffylogger"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/gregjones/httpcache"
	"go.uber.org/zap"
)

type AuthEsiPoller interface {
	Start(ctx context.Context)
	Poll(ctx context.Context)
	Stop(ctx context.Context)
}

type authEsiPoller struct {
	dependencies common.Dependencies
	tickTime     time.Duration
	ticker       *time.Ticker
	esiClient    *goesi.APIClient
}

func New(ctx context.Context, userAgent string, deps common.Dependencies) AuthEsiPoller {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.Info("Setting up Auth ESI Poller", zap.String("component", "esi-poller"))
	httpClient := httpcache.NewMemoryCacheTransport().Client()

	return &authEsiPoller{
		dependencies: deps,
		tickTime:     time.Minute * 60,
		esiClient:    goesi.NewAPIClient(httpClient, userAgent),
	}
}

func (aep *authEsiPoller) Start(ctx context.Context) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	aep.ticker = time.NewTicker(aep.tickTime)

	sp.Info("Starting polling loop")
	go func() {
		aep.Poll(ctx)
		for range aep.ticker.C {
			aep.Poll(ctx)
		}
	}()
}

func (aep *authEsiPoller) Stop(ctx context.Context) {
	_, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.Info("Stopping poller")
	aep.ticker.Stop()
}

// Poll currently starts at alliances and works it's way down to characters.  It then walks back up at the corporation
// level and character level if alliance/corporation membership has changed from the last poll.
func (aep *authEsiPoller) Poll(ctx context.Context) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	var (
		count      int
		errorCount int
		err        error
	)

	sp.Info("calling syncRoles()")
	count, errorCount, err = aep.syncRoles(ctx)
	if err == nil {
		sp.Info("syncRoles() completed", zap.Int("count", count), zap.Int("errorCount", errorCount))
		return
	} else {
		sp.Error("error synchronizing discord roles", zap.Error(err))
	}

	sp.Info("Calling updateAlliances()")
	count, errorCount, err = aep.updateAlliances(ctx)
	if err != nil {
		sp.Error("error updating alliances", zap.Error(err))
	} else {
		sp.Info("updateAlliances() completed", zap.Int("count", count), zap.Int("errorCount", errorCount))
	}

	sp.Info("Calling updateCorporations()")
	count, errorCount, err = aep.updateCorporations(ctx)
	if err != nil {
		sp.Error("error updating corporations", zap.Error(err))
	} else {
		sp.Info("updateCorporations() completed.", zap.Int("count", count), zap.Int("errorCount", errorCount))
	}

	sp.Info("Calling updateCharacters()")
	count, errorCount, err = aep.updateCharacters(ctx)
	if err != nil {
		sp.Error("error updating characters", zap.Error(err))
	} else {
		sp.Info("updateCharacters() completed", zap.Int("count", count), zap.Int("errorCount", errorCount))
	}
}

func (aep *authEsiPoller) notFound(ctx context.Context, err error) error {
	_, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	if err == nil {
		return errors.New("object found")
	}

	switch err.(type) {
	case esi.GenericSwaggerError:
		switch v := err.(esi.GenericSwaggerError).Model().(type) {
		case esi.GetAlliancesAllianceIdNotFound:
			sp.Debug("Alliance not found")
			return nil
		case esi.GetCorporationsCorporationIdNotFound:
			sp.Debug("Corporation not found")
			return nil
		case esi.GetCharactersCharacterIdNotFound:
			sp.Debug("Character not found")
			return nil
		default:
			sp.Error("API Error", zap.Error(err), zap.Any("errorType", v))
			return err
		}
	default:
		sp.Error("Other Error", zap.Error(err))
		return err
	}
}
