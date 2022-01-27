package esi_poller

import (
	"context"
	"errors"
	"time"

	"github.com/antihax/goesi"
	"github.com/antihax/goesi/esi"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/gregjones/httpcache"
	"go.uber.org/zap"
)

type AuthEsiPoller interface {
	Start()
	Poll()
	Stop()
}

type authEsiPoller struct {
	dependencies common.Dependencies
	logger       *zap.Logger
	tickTime     time.Duration
	ticker       *time.Ticker
	esiClient    *goesi.APIClient
	ctx          context.Context
}

func New(userAgent string, deps common.Dependencies) AuthEsiPoller {
	deps.Logger.Info("Setting up Auth ESI Poller", zap.String("component", "esi-poller"))
	httpClient := httpcache.NewMemoryCacheTransport().Client()

	return &authEsiPoller{
		dependencies: deps,
		logger:       deps.Logger.With(zap.String("component", "esi-poller")),
		tickTime:     time.Minute * 60,
		esiClient:    goesi.NewAPIClient(httpClient, userAgent),
		ctx:          deps.Context,
	}
}

func (aep *authEsiPoller) Start() {
	aep.ticker = time.NewTicker(aep.tickTime)

	aep.logger.Info("Starting polling loop")
	go func() {
		aep.Poll()
		for range aep.ticker.C {
			aep.Poll()
		}
	}()
}

func (aep *authEsiPoller) Stop() {
	aep.logger.Info("Stopping poller")
	aep.ticker.Stop()
}

// Poll currently starts at alliances and works it's way down to characters.  It then walks back up at the corporation
// level and character level if alliance/corporation membership has changed from the last poll.
func (aep *authEsiPoller) Poll() {
	var (
		count      int
		errorCount int
		err        error
	)

	aep.logger.Info("calling syncRoles()")
	count, errorCount, err = aep.syncRoles()
	if err == nil {
		aep.logger.Info("syncRoles() completed", zap.Int("count", count), zap.Int("errorCount", errorCount))
	} else {
		aep.logger.Error("error synchronizing discord roles", zap.Error(err))
	}

	aep.logger.Info("Calling updateAlliances()")
	count, errorCount, err = aep.updateAlliances()
	if err == nil {
		aep.logger.Info("updateAlliances() completed", zap.Int("count", count), zap.Int("errorCount", errorCount))
	} else {
		aep.logger.Error("error updating alliances", zap.Error(err))
	}

	aep.logger.Info("Calling updateCorporations()")
	count, errorCount, err = aep.updateCorporations()
	if err == nil {
		aep.logger.Info("updateCorporations() completed.", zap.Int("count", count), zap.Int("errorCount", errorCount))
	} else {
		aep.logger.Error("error updating corporations", zap.Error(err))
	}

	aep.logger.Info("Calling updateCharacters()")
	count, errorCount, err = aep.updateCharacters()
	if err == nil {
		aep.logger.Info("updateCharacters() completed", zap.Int("count", count), zap.Int("errorCount", errorCount))
	} else {
		aep.logger.Error("error updating characters", zap.Error(err))
	}
}

func (aep *authEsiPoller) notFound(err error) error {
	if err == nil {
		return errors.New("object found")
	}

	switch err.(type) {
	case esi.GenericSwaggerError:
		switch v := err.(esi.GenericSwaggerError).Model().(type) {
		case esi.GetAlliancesAllianceIdNotFound:
			aep.logger.Debug("Alliance not found")
			return nil
		case esi.GetCorporationsCorporationIdNotFound:
			aep.logger.Debug("Corporation not found")
			return nil
		case esi.GetCharactersCharacterIdNotFound:
			aep.logger.Debug("Character not found")
			return nil
		default:
			aep.logger.Error("API Error", zap.Error(err), zap.Any("errorType", v))
			return err
		}
	default:
		aep.logger.Error("Other Error", zap.Error(err))
		return err
	}
}
