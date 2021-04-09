package filters

import (
	"bytes"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func List(logger *zap.SugaredLogger, db *sq.StatementBuilderType) string {
	var (
		count int
		buffer bytes.Buffer
		filter payloads.Filter
	)

	rows, err := db.Select("name", "description").
		From("filters").
		Where(sq.Eq{"namespace": viper.GetString("namespace")}).
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

func Add(name, description string, logger *zap.SugaredLogger, db *sq.StatementBuilderType) string {
	_, err := db.Insert("filters").
		Columns("namespace", "name", "description").
		Values(viper.GetString("namespace"), name, description).
		Query()
	if err != nil {
		newErr := fmt.Errorf("error inserting filter: %s", err)
		logger.Error(newErr)
		return common.SendFatal(newErr.Error())
	}

	return common.SendSuccess(fmt.Sprintf("Created filter `%s`", name))
}

func Delete(name string, logger *zap.SugaredLogger, db *sq.StatementBuilderType) string {
	_, err := db.Delete("filters").
		Where(sq.Eq{"name": name}).
		Where(sq.Eq{"namespace": viper.GetString("namespace")}).
		Query()
	if err != nil {
		newErr := fmt.Errorf("error deleting filter: %s", err)
		logger.Error(newErr)
		return common.SendFatal(newErr.Error())
	}

	return common.SendSuccess(fmt.Sprintf("Deleted filter `%s`", name))
}

func Members(name string, logger *zap.SugaredLogger, db *sq.StatementBuilderType) string {
	var (
		count, userID int
		buffer bytes.Buffer
	)

	rows, err := db.Select("user_id").
		From("filters").
		Join("filter_membership ON filters.id = filter_membership.filter").
		Where(sq.Eq{"filters.name": name}).
		Where(sq.Eq{"filters.namespace": viper.GetString("namespace")}).
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

func AddMember(member, filter string, logger *zap.SugaredLogger, db *sq.StatementBuilderType) string {
	var filterID int

	if !common.IsDiscordUser(member) {
		return common.SendError("Second argument must be a discord user")
	}

	err := db.Select("id").
		From("filters").
		Where(sq.Eq{"name": filter}).
		Where(sq.Eq{"namespace": viper.GetString("namespace")}).
		QueryRow().Scan(&filterID)
	if err != nil {
		newErr := fmt.Errorf("error scanning filterID: %s", err)
		logger.Error(newErr)
		return common.SendFatal(newErr.Error())
	}

	userID := common.ExtractUserId(member)

	_, err = db.Insert("filter_membership").
		Columns("namespace", "filter", "user_id").
		Values(viper.GetString("namespace"), filterID, userID).
		Query()
	if err != nil {
		newErr := fmt.Errorf("error inserting filter: %s", err)
		logger.Error(newErr)
		return common.SendFatal(newErr.Error())
	}

	return common.SendSuccess(fmt.Sprintf("Added <@%s> to `%s`", userID, filter))
}

func RemoveMember(member, filter string, logger *zap.SugaredLogger, db *sq.StatementBuilderType) string {
	var filterID int

	if !common.IsDiscordUser(member) {
		return common.SendError("Second argument must be a discord user")
	}

	err := db.Select("id").
		From("filters").
		Where(sq.Eq{"name": filter}).
		Where(sq.Eq{"namespace": viper.GetString("namespace")}).
		QueryRow().Scan(&filterID)
	if err != nil {
		newErr := fmt.Errorf("error scanning filterID: %s", err)
		logger.Error(newErr)
		return common.SendFatal(newErr.Error())
	}

	userID := common.ExtractUserId(member)

	_, err = db.Delete("filter_membership").
		Where(sq.Eq{"filter": filterID}).
		Where(sq.Eq{"user_id": userID}).
		Where(sq.Eq{"namespace": viper.GetString("namespace")}).
		Query()
	if err != nil {
		newErr := fmt.Errorf("error deleting filter: %s", err)
		logger.Error(newErr)
		return common.SendFatal(newErr.Error())
	}

	return common.SendSuccess(fmt.Sprintf("Removed <@%s> to `%s`", userID, filter))
}