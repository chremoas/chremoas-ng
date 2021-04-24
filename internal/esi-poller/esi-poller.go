package esi_poller

import (
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/antihax/goesi"
	"github.com/antihax/goesi/esi"
	"github.com/chremoas/auth-srv/proto"
	"github.com/chremoas/chremoas-ng/internal/auth"
	"github.com/chremoas/esi-srv/proto"
	"github.com/gregjones/httpcache"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

type AuthEsiPoller interface {
	Start()
	Poll() error
	Stop()
}

type authEsiPoller struct {
	tickTime  time.Duration
	ticker    *time.Ticker
	logger    *zap.SugaredLogger
	db        *sq.StatementBuilderType
	esiClient *goesi.APIClient
}

func New(userAgent string, logger *zap.SugaredLogger) AuthEsiPoller {
	logger.Info("ESI Poller: Setting up Auth ESI Poller")
	httpClient := httpcache.NewMemoryCacheTransport().Client()
	//goesi.NewAPIClient(httpClient, "chremoas-esi-srv Ramdar Chinken on TweetFleet Slack https://github.com/chremoas/esi-srv")

	return &authEsiPoller{
		tickTime:  time.Minute * 60,
		logger:    logger,
		esiClient: goesi.NewAPIClient(httpClient, userAgent),
	}
}

func (aep *authEsiPoller) Start() {
	aep.ticker = time.NewTicker(aep.tickTime)

	aep.logger.Info("ESI Poller: Starting polling loop")
	go func() {
		err := aep.Poll()
		if err != nil {
			//TODO: Replace with logger object
			aep.logger.Errorf("ESI Poller: Received an error while polling: %s", err)
		}
		for range aep.ticker.C {
			err := aep.Poll()
			if err != nil {
				//TODO: Replace with logger object
				aep.logger.Errorf("ESI Poller: Received an error while polling: %s", err)
			}
		}
	}()
}

// Poll currently starts at alliances and works it's way down to characters.  It then walks back up at the corporation
// level and character level if alliance/corporation membership has changed from the last poll.
func (aep *authEsiPoller) Poll() error {
	allErrors := ""

	aep.logger.Info("ESI Poller: Calling updateOrDeleteAlliances()")
	err := aep.updateOrDeleteAlliances()
	if err != nil {
		allErrors += err.Error() + "\n"
	}

	aep.logger.Info("ESI Poller: Calling updateOrDeleteCorporations()")
	err = aep.updateOrDeleteCorporations()
	if err != nil {
		allErrors += err.Error() + "\n"
	}

	aep.logger.Info("ESI Poller: Calling updateOrDeleteCharacters()")
	err = aep.updateOrDeleteCharacters()
	if err != nil {
		allErrors += err.Error() + "\n"
	}

	if len(allErrors) > 0 {
		return errors.New(allErrors)
	}

	return nil
}

func (aep *authEsiPoller) updateOrDeleteAlliances() error {
	var (
		err      error
		alliance auth.Alliance
		response esi.GetAlliancesAllianceIdOk
	)

	rows, err := aep.db.Select("id", "name", "ticker").
		From("alliances").
		Query()
	if err != nil {
		return err
	}
	defer func() {
		if err = rows.Close(); err != nil {
			aep.logger.Error(err)
		}
	}()

	for rows.Next() {
		err = rows.Scan(&alliance.ID, &alliance.Name, &alliance.Ticker)
		if err != nil {
			return err
		}

		response, _, err = aep.esiClient.ESI.AllianceApi.GetAlliancesAllianceId(context.Background(), int32(alliance.ID), nil)
		if err != nil {
			aep.logger.Errorf("Error calling GetAlliancesAllianceId: %s", err)
			aep.logger.Infof("response=%v error=%s", response, err)
			return err
		}

		// this may come back as an error now? Need to figure that out
		//if response. == nil {
		//	aep.logger.Infof("ESI Poller: Removing alliance: %s", alliance)
		//	aep.entityAdminClient.AllianceUpdate(context.Background(), &abaeve_auth.AllianceAdminRequest{
		//		Alliance:  alliance,
		//		Operation: abaeve_auth.EntityOperation_REMOVE,
		//	})
		if alliance.Name != response.Name || alliance.Ticker != response.Ticker {
			aep.logger.Infof("ESI Poller: Updating alliance: %d", alliance.ID)
			_, err = aep.db.Update("alliances").
				Set("name", response.Name).
				Set("ticket", response.Ticker).
				Query()
			if err != nil {
				aep.logger.Errorf("ESI Poller: Error updating alliance: %d", alliance.ID)
			}
		}
	}

	return nil
}

func (aep *authEsiPoller) updateOrDeleteCorporations() error {
	var (
		err         error
		corporation auth.Corporation
		response    esi.GetCorporationsCorporationIdOk
	)

	rows, err := aep.db.Select("id", "name", "ticker", "alliance_id").
		From("corporations").
		Query()
	if err != nil {
		return err
	}
	defer func() {
		if err = rows.Close(); err != nil {
			aep.logger.Error(err)
		}
	}()

	for rows.Next() {
		err = rows.Scan(&corporation.ID, &corporation.Name, &corporation.Ticker, &corporation.AllianceID)
		if err != nil {
			return err
		}

		response, _, err = aep.esiClient.ESI.CorporationApi.GetCorporationsCorporationId(context.Background(), int32(corporation.ID), nil)
		if err != nil {
			aep.logger.Errorf("Error calling GetCorporationsCorporationId: %s", err)
			aep.logger.Infof("response=%v error=%s", response, err)
			return err
		}

		// this may come back as an error now? Need to figure that out
		//if response. == nil {
		//	aep.logger.Infof("ESI Poller: Removing alliance: %s", alliance)
		//	aep.entityAdminClient.AllianceUpdate(context.Background(), &abaeve_auth.AllianceAdminRequest{
		//		Alliance:  alliance,
		//		Operation: abaeve_auth.EntityOperation_REMOVE,
		//	})
		if corporation.Name != response.Name || corporation.Ticker != response.Ticker || corporation.AllianceID != int64(response.AllianceId) {
			aep.logger.Infof("ESI Poller: Updating alliance: %d", corporation.ID)
			_, err = aep.db.Update("corporations").
				Set("name", response.Name).
				Set("ticket", response.Ticker).
				Set("alliance_id", response.AllianceId).
				Query()
			if err != nil {
				aep.logger.Errorf("ESI Poller: Error updating alliance: %d", corporation.ID)
			}
		}
		err = aep.checkAndUpdateCorpsAllianceIfNecessary(corporation, response)
		if err != nil {
			aep.logger.Errorf("Error updating %d's alliance: %s", corporation.ID, err)
		}
	}

	return nil
}

func (aep *authEsiPoller) updateOrDeleteCharacters() error {
	var (
		err       error
		character auth.Character
		response  esi.GetCharactersCharacterIdOk
	)

	rows, err := aep.db.Select("id", "name", "corporation_id").
		From("characters").
		Query()
	if err != nil {
		return err
	}
	defer func() {
		if err = rows.Close(); err != nil {
			aep.logger.Error(err)
		}
	}()

	for rows.Next() {
		err = rows.Scan(&character.ID, &character.Name, &character.CorporationID)
		if err != nil {
			return err
		}

		response, _, err = aep.esiClient.ESI.CharacterApi.GetCharactersCharacterId(context.Background(), int32(character.ID), nil)
		if err != nil {
			aep.logger.Errorf("Error calling GetCharactersCharacterId: %s", err)
			aep.logger.Infof("response=%v error=%s", response, err)
			return err
		}

		// this may come back as an error now? Need to figure that out
		//if response. == nil {
		//	aep.logger.Infof("ESI Poller: Removing alliance: %s", alliance)
		//	aep.entityAdminClient.AllianceUpdate(context.Background(), &abaeve_auth.AllianceAdminRequest{
		//		Alliance:  alliance,
		//		Operation: abaeve_auth.EntityOperation_REMOVE,
		//	})
		if character.Name != response.Name || character.CorporationID != int64(response.CorporationId) {
			aep.logger.Infof("ESI Poller: Updating alliance: %d", character.ID)
			_, err = aep.db.Update("corporations").
				Set("name", response.Name).
				Set("corporation_id", response.CorporationId).
				Query()
			if err != nil {
				aep.logger.Errorf("ESI Poller: Error updating alliance: %d", character.ID)
			}
		}
		// I don't think we really need to do this here, it'll get updated when we check corps
		//err = aep.checkAndUpdateCorpsAllianceIfNecessary(corporation, response)
		//if err != nil {
		//	aep.logger.Errorf("Error updating %d's alliance: %s", corporation.ID, err)
		//}
	}

	return nil
}

func (aep *authEsiPoller) checkAndUpdateCorpsAllianceIfNecessary(authCorporation auth.Corporation, esiCorporation esi.GetCorporationsCorporationIdOk) error {
	var (
		err      error
		response esi.GetAlliancesAllianceIdOk
	)
	if esiCorporation.AllianceId == 0 {
		return nil
	}

	aep.logger.Infof("ESI Poller: Updating corporations alliance for %s with allianceId %d\n", esiCorporation.Name, esiCorporation.AllianceId)
	allErrors := ""

	if authCorporation.AllianceID != int64(esiCorporation.AllianceId) {
		response, _, err = aep.esiClient.ESI.AllianceApi.GetAlliancesAllianceId(context.Background(), esiCorporation.AllianceId, nil)
		newAllianceResponse, err := aep.allianceClient.GetAllianceById(context.Background(), &chremoas_esi.GetAllianceByIdRequest{
			Id: esiCorporation.AllianceId,
		})
		if err != nil {
			allErrors += err.Error() + "\n"
		}

		aep.authAllianceMap[esiCorporation.AllianceId] = &abaeve_auth.Alliance{
			Id:     int64(esiCorporation.AllianceId),
			Name:   newAllianceResponse.Alliance.Name,
			Ticker: newAllianceResponse.Alliance.Ticker,
		}

		aep.esiAllianceMap[int64(esiCorporation.AllianceId)] = newAllianceResponse.Alliance

		aep.logger.Infof("ESI Poller: Updating alliance: %s", aep.authAllianceMap[esiCorporation.AllianceId])
		_, err = aep.entityAdminClient.AllianceUpdate(context.Background(), &abaeve_auth.AllianceAdminRequest{
			Alliance:  aep.authAllianceMap[esiCorporation.AllianceId],
			Operation: abaeve_auth.EntityOperation_ADD_OR_UPDATE,
		})
		if err != nil {
			allErrors += err.Error() + "\n"
		}
	}

	if len(allErrors) > 0 {
		return errors.New(allErrors)
	}

	return nil
}

func (aep *authEsiPoller) Stop() {
	aep.ticker.Stop()
}
