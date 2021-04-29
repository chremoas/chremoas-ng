package roles

import (
	"encoding/json"
	"fmt"
	"strconv"

	sq "github.com/Masterminds/squirrel"
	"github.com/bwmarrin/discordgo"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	"github.com/nsqio/go-nsq"
	"go.uber.org/zap"
)

func getRoleID(name string, db *sq.StatementBuilderType) (int, error) {
	var (
		err error
		id  int
	)

	err = db.Select("id").
		From("roles").
		Where(sq.Eq{"role_nick": name}).
		QueryRow().Scan(&id)

	return id, err
}

func validListItem(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func queueUpdate(chatID int, action payloads.Action, logger *zap.SugaredLogger, nsq *nsq.Producer) {
	payload := payloads.Payload{
		Action: action,
		Role: &discordgo.Role{
			ID: fmt.Sprintf("%d", chatID),
		},
	}

	b, err := json.Marshal(payload)
	if err != nil {
		logger.Errorf("error marshalling json for queue: %s", err)
	}

	logger.Debug("Submitting role queue message")
	err = nsq.PublishAsync(common.GetTopic("role"), b, nil)
	if err != nil {
		logger.Errorf("error publishing message: %s", err)
	}
}

// GetRoleMembers lists all userIDs that match all the filters for a role.
func GetRoleMembers(sig bool, name string, logger *zap.SugaredLogger, db *sq.StatementBuilderType) ([]int, error) {
	var (
		err        error
		id         int
		members    []int
		filterList []int
	)

	rows, err := db.Select("role_filters.id").
		From("role_filters").
		InnerJoin("roles ON role_filters.role = roles.id").
		Where(sq.Eq{"sig": sig}).
		Where(sq.Eq{"role_nick": name}).
		Query()
	if err != nil {
		logger.Error(err)
		return nil, err
	}
	defer func() {
		if err = rows.Close(); err != nil {
			logger.Error(err)
		}
	}()

	// add filters to the membership query
	for rows.Next() {
		err = rows.Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("error scanning role's id (%s): %s", name, err.Error())
		}

		filterList = append(filterList, id)
	}

	rows, err = db.Select("user_id").
		From("filter_membership").
		Where(sq.Eq{"filter": filterList}).
		GroupBy("user_id").
		Having("count(*) = ?", len(filterList)).
		Query()
	if err != nil {
		logger.Error(err)
		return nil, err
	}
	defer func() {
		if err = rows.Close(); err != nil {
			logger.Error(err)
		}
	}()

	// add filters to the membership query
	for rows.Next() {
		err = rows.Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("error scanning filter's userID (%s): %s", name, err.Error())
		}

		members = append(members, id)
	}

	return members, nil
}

func GetUserRoles(sig bool, userID string, logger *zap.SugaredLogger, db *sq.StatementBuilderType) ([]payloads.Role, error) {
	var (
		roles []payloads.Role
	)

	_, err := strconv.Atoi(userID)
	if err != nil {
		if !common.IsDiscordUser(userID) {
			return nil, fmt.Errorf("second argument must be a discord user")
		}
		userID = common.ExtractUserId(userID)
	}

	rows, err := db.Select("role_nick", "name").
		From("").
		Suffix("getMemberRoles(?, ?)", userID, strconv.FormatBool(sig)).
		Query()
	if err != nil {
		logger.Error(err)
		return nil, fmt.Errorf("error getting user %ss (%s): %s", roleType[sig], userID, err)
	}
	defer func() {
		if err = rows.Close(); err != nil {
			logger.Error(err)
		}
	}()

	for rows.Next() {
		var role payloads.Role

		err = rows.Scan(
			&role.ShortName,
			&role.Name,
		)
		if err != nil {
			newErr := fmt.Errorf("error scanning %s row: %s", roleType[sig], err)
			logger.Error(newErr)
			return nil, newErr
		}

		roles = append(roles, role)
	}

	return roles, nil
}

// GetRoles goes and fetches all the roles of type sig/role. If shortname is set only one role is fetched.
func GetRoles(sig bool, shortName *string, logger *zap.SugaredLogger, db *sq.StatementBuilderType) ([]payloads.Role, error) {
	var (
		rs        []payloads.Role
		charTotal int
	)

	q := db.Select("color", "hoist", "joinable", "managed", "mentionable", "name", "permissions",
		"position", "role_nick", "sig", "sync").
		Where(sq.Eq{"sig": sig}).
		From("roles")

	if shortName != nil {
		q = q.Where(sq.Eq{"role_nick": shortName})
	}

	rows, err := q.Query()
	if err != nil {
		newErr := fmt.Errorf("error getting %ss: %s", roleType[sig], err)
		logger.Error(newErr)
		return nil, newErr
	}
	defer func() {
		if err = rows.Close(); err != nil {
			logger.Error(err)
		}
	}()

	var role payloads.Role
	for rows.Next() {
		err = rows.Scan(
			&role.Color,
			&role.Hoist,
			&role.Joinable,
			&role.Managed,
			&role.Mentionable,
			&role.Name,
			&role.Permissions,
			&role.Position,
			&role.ShortName,
			&role.Sig,
			&role.Sync,
		)
		if err != nil {
			newErr := fmt.Errorf("error scanning %s row: %s", roleType[sig], err)
			logger.Error(newErr)
			return nil, newErr
		}
		charTotal += len(role.ShortName) + len(role.Name) + 15 // Guessing on bool excess
		rs = append(rs, role)
	}

	if charTotal >= 2000 {
		return nil, fmt.Errorf("too many %ss (exceeds Discord 2k character limit)", roleType[sig])
	}

	return rs, nil
}
