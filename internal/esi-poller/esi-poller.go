package esi_poller

import (
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/antihax/goesi"
	"github.com/antihax/goesi/esi"
	"github.com/chremoas/chremoas-ng/internal/auth"
	"github.com/chremoas/chremoas-ng/internal/filters"
	"github.com/chremoas/chremoas-ng/internal/roles"
	"github.com/gregjones/httpcache"
	"github.com/nsqio/go-nsq"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

type AuthEsiPoller interface {
	Start()
	Poll()
	Stop()
}

type authEsiPoller struct {
	tickTime  time.Duration
	ticker    *time.Ticker
	logger    *zap.SugaredLogger
	db        *sq.StatementBuilderType
	nsq       *nsq.Producer
	esiClient *goesi.APIClient
}

func New(userAgent string, logger *zap.SugaredLogger, db *sq.StatementBuilderType, nsq *nsq.Producer) AuthEsiPoller {
	logger.Info("ESI Poller: Setting up Auth ESI Poller")
	httpClient := httpcache.NewMemoryCacheTransport().Client()
	//goesi.NewAPIClient(httpClient, "chremoas-esi-srv Ramdar Chinken on TweetFleet Slack https://github.com/chremoas/esi-srv")

	return &authEsiPoller{
		tickTime:  time.Minute * 60,
		logger:    logger,
		db:        db,
		nsq:       nsq,
		esiClient: goesi.NewAPIClient(httpClient, userAgent),
	}
}

func (aep *authEsiPoller) Start() {
	aep.ticker = time.NewTicker(aep.tickTime)

	aep.logger.Info("ESI Poller: Starting polling loop")
	go func() {
		aep.Poll()
		for range aep.ticker.C {
			aep.Poll()
		}
	}()
}

func (aep *authEsiPoller) Stop() {
	aep.logger.Info("Stopping ESI Poller")
	aep.ticker.Stop()
}

// Poll currently starts at alliances and works it's way down to characters.  It then walks back up at the corporation
// level and character level if alliance/corporation membership has changed from the last poll.
func (aep *authEsiPoller) Poll() {
	aep.logger.Info("ESI Poller: Calling updateOrDeleteAlliances()")
	aep.updateOrDeleteAlliances()

	aep.logger.Info("ESI Poller: Calling updateOrDeleteCorporations()")
	aep.updateOrDeleteCorporations()

	aep.logger.Info("ESI Poller: Calling updateOrDeleteCharacters()")
	aep.updateOrDeleteCharacters()
}

func (aep *authEsiPoller) updateOrDeleteAlliances() {
	var (
		err      error
		alliance auth.Alliance
	)

	rows, err := aep.db.Select("id", "name", "ticker").
		From("alliances").
		Query()
	if err != nil {
		aep.logger.Errorf("Error getting alliance list from db: %s", err)
		return
	}
	defer func() {
		if err = rows.Close(); err != nil {
			aep.logger.Error(err)
		}
	}()

	for rows.Next() {
		err = rows.Scan(&alliance.ID, &alliance.Name, &alliance.Ticker)
		if err != nil {
			aep.logger.Errorf("Error scanning alliance values: %s", err)
			return
		}

		aep.logger.Debugf("Checking alliance: %s (%d)", alliance.Name, alliance.ID)

		_, _, err = aep.esiClient.ESI.AllianceApi.GetAlliancesAllianceId(context.Background(), alliance.ID, nil)
		if err != nil {
			if aep.notFound(err) == nil {
				aep.logger.Infof("Deleting alliance: %d", alliance.ID)
				_, err = aep.db.Delete("alliances").
					Where(sq.Eq{"id": alliance.ID}).
					Query()
				if err != nil {
					aep.logger.Errorf("Error deleting alliance: %s", err)
				}
				continue
			}

			aep.logger.Errorf("Error calling GetAlliancesAllianceId: %s", err)
		}
	}
}

func (aep *authEsiPoller) updateOrDeleteCorporations() {
	var (
		err         error
		corporation auth.Corporation
		response    esi.GetCorporationsCorporationIdOk
	)

	rows, err := aep.db.Select("id", "name", "ticker", "alliance_id").
		From("corporations").
		Query()
	if err != nil {
		aep.logger.Errorf("Error getting corporation list from db: %s", err)
		return
	}
	defer func() {
		if err = rows.Close(); err != nil {
			aep.logger.Error(err)
		}
	}()

	for rows.Next() {
		err = rows.Scan(&corporation.ID, &corporation.Name, &corporation.Ticker, &corporation.AllianceID)
		if err != nil {
			aep.logger.Errorf("Error scanning corporation values: %s", err)
			return
		}

		aep.logger.Debugf("Checking corporation: %s (%d)", corporation.Name, corporation.ID)

		response, _, err = aep.esiClient.ESI.CorporationApi.GetCorporationsCorporationId(context.Background(), corporation.ID, nil)
		if err != nil {
			if aep.notFound(err) == nil {
				aep.logger.Infof("Deleting corporation: %d", corporation.ID)
				_, err = aep.db.Delete("corporations").
					Where(sq.Eq{"id": corporation.ID}).
					Query()
				if err != nil {
					aep.logger.Errorf("Error deleting corporation: %s", err)
				}
				continue
			}

			aep.logger.Errorf("Error calling GetCorporationsCorporationId: %s", err)
		}

		err = aep.checkAndUpdateCorpsAllianceIfNecessary(corporation, response)
		if err != nil {
			aep.logger.Errorf("Error updating %d's alliance: %s", corporation.ID, err)
		}

		// Corp has switched alliance
		if corporation.AllianceID.Int32 != response.AllianceId {
			_, err = aep.db.Update("corporations").
				Set("alliance_id", response.AllianceId).
				Where(sq.Eq{"id": corporation.ID}).
				Query()
			if err != nil {
				aep.logger.Errorf("Error updating alliance '%d' for corp '%s': %s", response.AllianceId, corporation.Name, err)
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
	}
}

func (aep *authEsiPoller) updateOrDeleteCharacters() {
	var (
		err       error
		character auth.Character
		response  esi.GetCharactersCharacterIdOk
	)

	rows, err := aep.db.Select("id", "name", "corporation_id").
		From("characters").
		Query()
	if err != nil {
		aep.logger.Errorf("Error getting character list from the db: %s", err)
		return
	}
	defer func() {
		if err = rows.Close(); err != nil {
			aep.logger.Error(err)
		}
	}()

	for rows.Next() {
		err = rows.Scan(&character.ID, &character.Name, &character.CorporationID)
		if err != nil {
			aep.logger.Errorf("Error scanning character values: %s", err)
			return
		}

		aep.logger.Debugf("Checking character: %s (%d)", character.Name, character.ID)

		response, _, err = aep.esiClient.ESI.CharacterApi.GetCharactersCharacterId(context.Background(), character.ID, nil)
		if err != nil {
			if aep.notFound(err) == nil {
				aep.logger.Infof("Deleting character: %d", character.ID)
				_, err = aep.db.Delete("characters").
					Where(sq.Eq{"id": character.ID}).
					Query()
				if err != nil {
					aep.logger.Errorf("Error deleting character: %s", err)
				}
				continue
			}

			aep.logger.Errorf("Error calling GetCharactersCharacterId: %s", err)
		}

		if character.CorporationID != response.CorporationId {
			_, err = aep.db.Update("characters").
				Set("corporation_id", response.CorporationId).
				Query()
			if err != nil {
				aep.logger.Errorf("Error updating character: %s", err)
			}
		}
	}
}

func (aep *authEsiPoller) checkAndUpdateCorpsAllianceIfNecessary(authCorporation auth.Corporation, esiCorporation esi.GetCorporationsCorporationIdOk) error {
	var (
		err      error
		response esi.GetAlliancesAllianceIdOk
		count    int32
	)

	if esiCorporation.AllianceId == 0 {
		return nil
	}

	aep.logger.Infof("ESI Poller: Updating corporation's alliance for %s with allianceId %d\n", esiCorporation.Name, esiCorporation.AllianceId)

	if authCorporation.AllianceID.Int32 != esiCorporation.AllianceId {
		aep.logger.Debugf("Updating alliance (cascade): %s (%d)", authCorporation.Name, authCorporation.AllianceID)

		response, _, err = aep.esiClient.ESI.AllianceApi.GetAlliancesAllianceId(context.Background(), esiCorporation.AllianceId, nil)
		if err != nil {
			aep.logger.Errorf("Error calling GetAlliancesAllianceId: %s", err)
			return err
		}

		aep.upsertAlliance(esiCorporation.AllianceId, response.Name, response.Ticker)

		err := aep.db.Select("count(id)").
			From("roles").
			Where(sq.Eq{"role_nick": response.Ticker}).
			Where(sq.Eq{"sig": roles.Role}).
			QueryRow().Scan(&count)
		if err != nil {
			aep.logger.Errorf("error getting count of alliances by name: %s", err)
			return err
		}

		if count == 0 {
			roles.Add(roles.Role, false, response.Ticker, response.Name, "discord", aep.logger, aep.db, aep.nsq)
		} else {
			roles.Update(roles.Role, response.Ticker, "role_nick", response.Ticker, aep.logger, aep.db, aep.nsq)
		}
	}

	return nil
}

func (aep *authEsiPoller) addCorpMembers(corpTicker string, allianceID int32) {
	var allianceTicker string

	err := aep.db.Select("ticker").
		From("alliances").
		Where(sq.Eq{"id": allianceID}).
		QueryRow().Scan(&allianceTicker)
	if err != nil {
		aep.logger.Errorf("error getting alliance ticker for %d: %s", allianceID, err)
		return
	}

	members, err := roles.GetRoleMembers(roles.Role, corpTicker, aep.logger, aep.db)
	if err != nil {
		aep.logger.Errorf("error getting corp member list to add to alliance: %s", err)
		return
	}

	for member := range members {
		filters.AddMember(fmt.Sprintf("%d", member), allianceTicker, aep.logger, aep.db, aep.nsq)
	}
}

func (aep *authEsiPoller) removeCorpMembers(corpTicker string, allianceID int32) {
	var allianceTicker string

	err := aep.db.Select("ticker").
		From("alliances").
		Where(sq.Eq{"id": allianceID}).
		QueryRow().Scan(&allianceTicker)
	if err != nil {
		aep.logger.Errorf("error getting alliance ticker for %d: %s", allianceID, err)
		return
	}

	members, err := roles.GetRoleMembers(roles.Role, corpTicker, aep.logger, aep.db)
	if err != nil {
		aep.logger.Errorf("error getting corp member list to remove from alliance: %s", err)
		return
	}

	for member := range members {
		filters.RemoveMember(fmt.Sprintf("%d", member), allianceTicker, aep.logger, aep.db, aep.nsq)
	}
}

func (aep *authEsiPoller) upsertAlliance(allianceID int32, name, ticker string) {
	var err error

	aep.logger.Infof("ESI Poller: Updating alliance: %d", allianceID)
	_, err = aep.db.Insert("alliances").
		Columns("id", "name", "ticker").
		Values(allianceID, name, ticker).
		Suffix("ON CONFLICT (id) DO UPDATE SET name=?, ticker=?", name, ticker).
		Query()
	if err != nil {
		aep.logger.Errorf("ESI Poller: Error updating alliance: %d", allianceID)
	}
}

func (aep *authEsiPoller) notFound(err error) error {
	if err == nil {
		return errors.New("object found")
	}

	switch err.(type) {
	case esi.GenericSwaggerError:
		switch err.(esi.GenericSwaggerError).Model().(type) {
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
			aep.logger.Errorf("API Error: %s", err)
			return err
		}
	default:
		aep.logger.Errorf("Other Error: %s", err)
		return err
	}
}
