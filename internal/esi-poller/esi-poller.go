package esi_poller

import (
	"errors"
	"time"

	"github.com/antihax/goesi"
	"github.com/antihax/goesi/esi"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/gregjones/httpcache"
)

type AuthEsiPoller interface {
	Start()
	Poll()
	Stop()
}

type authEsiPoller struct {
	dependencies common.Dependencies
	tickTime     time.Duration
	ticker       *time.Ticker
	esiClient    *goesi.APIClient
}

func New(userAgent string, deps common.Dependencies) AuthEsiPoller {
	deps.Logger.Info("ESI Poller: Setting up Auth ESI Poller")
	httpClient := httpcache.NewMemoryCacheTransport().Client()

	return &authEsiPoller{
		dependencies: deps,
		tickTime:     time.Minute * 60,
		esiClient:    goesi.NewAPIClient(httpClient, userAgent),
	}
}

func (aep *authEsiPoller) Start() {
	aep.ticker = time.NewTicker(aep.tickTime)

	aep.dependencies.Logger.Info("ESI Poller: Starting polling loop")
	go func() {
		aep.Poll()
		for range aep.ticker.C {
			aep.Poll()
		}
	}()
}

func (aep *authEsiPoller) Stop() {
	aep.dependencies.Logger.Info("Stopping ESI Poller")
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

	aep.dependencies.Logger.Info("ESI Poller: calling syncRoles()")
	count, errorCount, err = aep.syncRoles()
	if err == nil {
		aep.dependencies.Logger.Infof("ESI Poller: syncRoles() processed %d entries (%d errors)", count, errorCount)
	} else {
		aep.dependencies.Logger.Error(err)
	}

	aep.dependencies.Logger.Info("ESI Poller: Calling updateAlliances()")
	count, errorCount, err = aep.updateAlliances()
	if err == nil {
		aep.dependencies.Logger.Infof("ESI Poller: updateAlliances() processed %d entries (%d errors)", count, errorCount)
	} else {
		aep.dependencies.Logger.Error(err)
	}

	aep.dependencies.Logger.Info("ESI Poller: Calling updateCorporations()")
	count, errorCount, err = aep.updateCorporations()
	if err == nil {
		aep.dependencies.Logger.Infof("ESI Poller: updateCorporations() processed %d entries (%d errors)", count, errorCount)
	} else {
		aep.dependencies.Logger.Error(err)
	}

	aep.dependencies.Logger.Info("ESI Poller: Calling updateCharacters()")
	count, errorCount, err = aep.updateCharacters()
	if err == nil {
		aep.dependencies.Logger.Infof("ESI Poller: updateCharacters() processed %d entries (%d errors)", count, errorCount)
	} else {
		aep.dependencies.Logger.Error(err)
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
			aep.dependencies.Logger.Debug("Alliance not found")
			return nil
		case esi.GetCorporationsCorporationIdNotFound:
			aep.dependencies.Logger.Debug("Corporation not found")
			return nil
		case esi.GetCharactersCharacterIdNotFound:
			aep.dependencies.Logger.Debug("Character not found")
			return nil
		default:
			aep.dependencies.Logger.Errorf("API Error: %s (%T)", err, v)
			return err
		}
	default:
		aep.dependencies.Logger.Errorf("Other Error: %s", err)
		return err
	}
}
