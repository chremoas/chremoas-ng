package filters

import (
	"bytes"
	"encoding/json"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	"github.com/chremoas/chremoas-ng/internal/perms"
	"github.com/lib/pq"
	"github.com/nsqio/go-nsq"
	"go.uber.org/zap"
)

func List(logger *zap.SugaredLogger, db *sq.StatementBuilderType) string {
	var (
		count  int
		buffer bytes.Buffer
		filter payloads.Filter
	)

	rows, err := db.Select("name", "description").
		From("filters").
		Query()
	if err != nil {
		logger.Error(err)
		return common.SendFatal(err.Error())
	}

	buffer.WriteString("Filters:\n")
	for rows.Next() {
		err = rows.Scan(&filter.Name, &filter.Description)
		if err != nil {
			newErr := fmt.Errorf("error scanning filter row: %s", err)
			logger.Error(newErr)
			return common.SendFatal(newErr.Error())
		}

		buffer.WriteString(fmt.Sprintf("\t%s: %s\n", filter.Name, filter.Description))
		count += 1
	}

	if count == 0 {
		return common.SendError("No filters")
	}

	return fmt.Sprintf("```%s```", buffer.String())
}

func Add(name, description string, sig bool, author string, logger *zap.SugaredLogger, db *sq.StatementBuilderType) (string, int) {
	var id int

	if !perms.CanPerform(author, "role_admins", logger, db) {
		return common.SendError("User doesn't have permission to this command"), -1
	}

	err := db.Insert("filters").
		Columns("name", "description", "sig").
		Values(name, description, sig).
		Suffix("RETURNING \"id\"").
		QueryRow().Scan(&id)
	if err != nil {
		// I don't love this but I can't find a better way right now
		if err.(*pq.Error).Code == "23505" {
			return common.SendError(fmt.Sprintf("filter `%s` already exists", name)), -1
		}
		newErr := fmt.Errorf("error inserting filter: %s", err)
		logger.Error(newErr)
		return common.SendFatal(newErr.Error()), -1
	}

	return common.SendSuccess(fmt.Sprintf("Created filter `%s`", name)), id
}

func Delete(name string, sig bool, author string, logger *zap.SugaredLogger, db *sq.StatementBuilderType) (string, int) {
	var id int

	if !perms.CanPerform(author, "role_admins", logger, db) {
		return common.SendError("User doesn't have permission to this command"), -1
	}

	rows, err := db.Delete("filters").
		Where(sq.Eq{"name": name}).
		Where(sq.Eq{"sig": sig}).
		Suffix("RETURNING \"id\"").
		Query()
	if err != nil {
		newErr := fmt.Errorf("error deleting filter: %s", err)
		logger.Error(newErr)
		return common.SendFatal(newErr.Error()), -1
	}

	for rows.Next() {
		err = rows.Scan(&id)
		if err != nil {
			newErr := fmt.Errorf("error scanning filters id: %s", err)
			logger.Error(newErr)
			return common.SendFatal(newErr.Error()), -1
		}
	}

	return common.SendSuccess(fmt.Sprintf("Deleted filter `%s`", name)), id
}

func Members(name string, logger *zap.SugaredLogger, db *sq.StatementBuilderType) string {
	var (
		count, userID int
		buffer        bytes.Buffer
	)

	rows, err := db.Select("user_id").
		From("filters").
		Join("filter_membership ON filters.id = filter_membership.filter").
		Where(sq.Eq{"filters.name": name}).
		Query()
	if err != nil {
		newErr := fmt.Errorf("error getting filter membership list: %s", err)
		logger.Error(newErr)
		return common.SendFatal(newErr.Error())
	}

	buffer.WriteString(fmt.Sprintf("Filter membership (%s):\n", name))
	for rows.Next() {
		err = rows.Scan(&userID)
		if err != nil {
			newErr := fmt.Errorf("error scanning filter_membership userID: %s", err)
			logger.Error(newErr)
			return common.SendFatal(newErr.Error())
		}
		buffer.WriteString(fmt.Sprintf("\t<@%d>\n", userID))
		count += 1
	}

	if count == 0 {
		return common.SendError(fmt.Sprintf("Filter has no members: %s", name))
	}

	return buffer.String()
}

func AddMember(sig bool, userID, filter, author string, logger *zap.SugaredLogger, db *sq.StatementBuilderType, nsq *nsq.Producer) string {
	var filterID int

	if author != "sig-cmd" {
		if !perms.CanPerform(author, "role_admins", logger, db) {
			return common.SendError("User doesn't have permission to this command")
		}
	}

	err := db.Select("id").
		From("filters").
		Where(sq.Eq{"name": filter}).
		Where(sq.Eq{"sig": sig}).
		QueryRow().Scan(&filterID)
	if err != nil {
		newErr := fmt.Errorf("error scanning filterID: %s", err)
		logger.Error(newErr)
		return common.SendFatal(newErr.Error())
	}

	logger.Infof("Got userID:%s filterID:%d", userID, filterID)

	_, err = db.Insert("filter_membership").
		Columns("filter", "user_id").
		Values(filterID, userID).
		Query()
	if err != nil {
		// I don't love this but I can't find a better way right now
		if err.(*pq.Error).Code == "23505" {
			return common.SendError(fmt.Sprintf("<@%s> already a member of `%s`", userID, filter))
		}
		newErr := fmt.Errorf("error inserting filter: %s", err)
		logger.Error(newErr)
		return common.SendFatal(newErr.Error())
	}

	queueUpdate(userID, logger, nsq)

	return common.SendSuccess(fmt.Sprintf("Added <@%s> to `%s`", userID, filter))
}

func RemoveMember(sig bool, userID, filter, author string, logger *zap.SugaredLogger, db *sq.StatementBuilderType, nsq *nsq.Producer) string {
	var (
		filterID int
		deleted  bool
	)

	if author != "sig-cmd" {
		if !perms.CanPerform(author, "role_admins", logger, db) {
			return common.SendError("User doesn't have permission to this command")
		}
	}

	err := db.Select("id").
		From("filters").
		Where(sq.Eq{"name": filter}).
		Where(sq.Eq{"sig": sig}).
		QueryRow().Scan(&filterID)
	if err != nil {
		newErr := fmt.Errorf("error scanning filterID: %s", err)
		logger.Error(newErr)
		return common.SendFatal(newErr.Error())
	}

	rows, err := db.Delete("filter_membership").
		Where(sq.Eq{"filter": filterID}).
		Where(sq.Eq{"user_id": userID}).
		Suffix("RETURNING \"id\"").
		Query()
	if err != nil {
		newErr := fmt.Errorf("error deleting filter: %s", err)
		logger.Error(newErr)
		return common.SendFatal(newErr.Error())
	}

	for rows.Next() {
		deleted = true
	}

	if deleted {
		queueUpdate(userID, logger, nsq)
		return common.SendSuccess(fmt.Sprintf("Removed <@%s> from `%s`", userID, filter))
	}

	return common.SendError(fmt.Sprintf("<@%s> not a member of `%s`", userID, filter))
}

func queueUpdate(member string, logger *zap.SugaredLogger, nsq *nsq.Producer) {
	payload := payloads.Payload{
		Member: member,
	}

	b, err := json.Marshal(payload)
	if err != nil {
		logger.Errorf("error marshalling queue message: %s", err)
	}

	logger.Debug("Submitting member queue message")
	err = nsq.PublishAsync(common.GetTopic("member"), b, nil)
	if err != nil {
		logger.Errorf("error publishing message: %s", err)
	}
}
