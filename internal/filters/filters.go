package filters

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"

	sq "github.com/Masterminds/squirrel"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	"github.com/chremoas/chremoas-ng/internal/perms"
	"github.com/lib/pq"
)

func List(deps common.Dependencies) string {
	var (
		buffer bytes.Buffer
		filter payloads.Filter
	)

	ctx, cancel := context.WithCancel(deps.Context)
	defer cancel()

	rows, err := deps.DB.Select("name", "description").
		From("filters").
		QueryContext(ctx)
	if err != nil {
		deps.Logger.Error(err)
		return common.SendFatal(err.Error())
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			deps.Logger.Errorf("error closing database: %s", err)
		}
	}()

	buffer.WriteString("Filters:\n")
	for rows.Next() {
		err = rows.Scan(&filter.Name, &filter.Description)
		if err != nil {
			newErr := fmt.Errorf("error scanning filter row: %s", err)
			deps.Logger.Error(newErr)
			return common.SendFatal(newErr.Error())
		}

		buffer.WriteString(fmt.Sprintf("\t%s: %s\n", filter.Name, filter.Description))
	}

	if buffer.Len() == 0 {
		return common.SendError("No filters")
	}

	if buffer.Len() > 2000 {
		return common.SendError("too many filters (exceeds Discord 2k character limit)")
	}

	return fmt.Sprintf("```%s```", buffer.String())
}

func AuthedAdd(name, description string, author string, deps common.Dependencies) (string, int) {
	if !perms.CanPerform(author, "role_admins", deps) {
		return common.SendError("User doesn't have permission to this command"), -1
	}

	return Add(name, description, deps)
}

func Add(name, description string, deps common.Dependencies) (string, int) {
	var id int
	err := deps.DB.Insert("filters").
		Columns("name", "description").
		Values(name, description).
		Suffix("RETURNING \"id\"").
		Scan(&id)
	if err != nil {
		// I don't love this but I can't find a better way right now
		if err.(*pq.Error).Code == "23505" {
			return common.SendError(fmt.Sprintf("filter `%s` already exists", name)), -1
		}
		newErr := fmt.Errorf("error inserting filter: %s", err)
		deps.Logger.Error(newErr)
		return common.SendFatal(newErr.Error()), -1
	}

	return common.SendSuccess(fmt.Sprintf("Created filter `%s`", name)), id
}

func AuthedDelete(name string, author string, deps common.Dependencies) (string, int) {
	if !perms.CanPerform(author, "role_admins", deps) {
		return common.SendError("User doesn't have permission to this command"), -1
	}

	return Delete(name, deps)
}

func Delete(name string, deps common.Dependencies) (string, int) {
	var id int

	ctx, cancel := context.WithCancel(deps.Context)
	defer cancel()

	rows, err := deps.DB.Delete("filters").
		Where(sq.Eq{"name": name}).
		Suffix("RETURNING \"id\"").
		QueryContext(ctx)
	if err != nil {
		newErr := fmt.Errorf("error deleting filter: %s", err)
		deps.Logger.Error(newErr)
		return common.SendFatal(newErr.Error()), -1
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			deps.Logger.Errorf("error closing database: %s", err)
		}
	}()

	for rows.Next() {
		err = rows.Scan(&id)
		if err != nil {
			newErr := fmt.Errorf("error scanning filters id: %s", err)
			deps.Logger.Error(newErr)
			return common.SendFatal(newErr.Error()), -1
		}
	}

	return common.SendSuccess(fmt.Sprintf("Deleted filter `%s`", name)), id
}

func ListMembers(name string, deps common.Dependencies) string {
	var (
		userID int
		buffer bytes.Buffer
	)

	ctx, cancel := context.WithCancel(deps.Context)
	defer cancel()

	rows, err := deps.DB.Select("user_id").
		From("filters").
		Join("filter_membership ON filters.id = filter_membership.filter").
		Where(sq.Eq{"filters.name": name}).
		QueryContext(ctx)
	if err != nil {
		newErr := fmt.Errorf("error getting filter membership list: %s", err)
		deps.Logger.Error(newErr)
		return common.SendFatal(newErr.Error())
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			deps.Logger.Errorf("error closing database: %s", err)
		}
	}()

	buffer.WriteString(fmt.Sprintf("Filter membership (%s):\n", name))
	for rows.Next() {
		err = rows.Scan(&userID)
		if err != nil {
			newErr := fmt.Errorf("error scanning filter_membership userID: %s", err)
			deps.Logger.Error(newErr)
			return common.SendFatal(newErr.Error())
		}
		buffer.WriteString(fmt.Sprintf("\t%s\n", common.GetUsername(userID, deps.Session)))
	}

	if buffer.Len() == 0 {
		return common.SendError(fmt.Sprintf("Filter has no members: %s", name))
	}

	if buffer.Len() > 2000 {
		return common.SendError("too many filter members (exceeds Discord 2k character limit)")
	}

	return buffer.String()
}

func AuthedAddMember(userID, filter, author string, deps common.Dependencies) string {
	if !perms.CanPerform(author, "role_admins", deps) {
		return common.SendError("User doesn't have permission to this command")
	}

	return AddMember(userID, filter, deps)
}

func AddMember(userID, filter string, deps common.Dependencies) string {
	var filterID int

	ctx, cancel := context.WithCancel(deps.Context)
	defer cancel()

	_, err := strconv.Atoi(userID)
	if err != nil {
		if !common.IsDiscordUser(userID) {
			return common.SendError("second argument must be a discord user")
		}
		userID = common.ExtractUserId(userID)
	}

	before, err := common.GetMembership(userID, deps)
	if err != nil {
		return common.SendFatal(err.Error())
	}

	err = deps.DB.Select("id").
		From("filters").
		Where(sq.Eq{"name": filter}).
		Scan(&filterID)
	if err != nil {
		if err == sql.ErrNoRows {
			return common.SendError(fmt.Sprintf("No such filter: %s", filter))
		}
		newErr := fmt.Errorf("error scanning filterID: %s", err)
		deps.Logger.Error(newErr)
		return common.SendFatal(newErr.Error())
	}

	deps.Logger.Infof("Got userID:%s filterID:%d", userID, filterID)

	rows, err := deps.DB.Insert("filter_membership").
		Columns("filter", "user_id").
		Values(filterID, userID).
		QueryContext(ctx)
	if err != nil {
		// I don't love this but I can't find a better way right now
		if err.(*pq.Error).Code == "23505" {
			return common.SendError(fmt.Sprintf("<@%s> already a member of `%s`", userID, filter))
		}
		newErr := fmt.Errorf("error inserting filter: %s", err)
		deps.Logger.Error(newErr)
		return common.SendFatal(newErr.Error())
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			deps.Logger.Errorf("error closing database: %s", err)
		}
	}()

	after, err := common.GetMembership(userID, deps)
	if err != nil {
		return common.SendFatal(err.Error())
	}

	addSet := after.Difference(before)

	if addSet.Len() == 0 {
		return common.SendError(fmt.Sprintf("<@%s> already a member of: `%s`", userID, filter))
	}

	for _, role := range addSet.ToSlice() {
		QueueUpdate(payloads.Upsert, userID, role, deps)
	}

	return common.SendSuccess(fmt.Sprintf("Added <@%s> to `%s`", userID, filter))
}

func AuthedRemoveMember(userID, filter, author string, deps common.Dependencies) string {
	if !perms.CanPerform(author, "role_admins", deps) {
		return common.SendError("User doesn't have permission to this command")
	}

	return RemoveMember(userID, filter, deps)
}

func RemoveMember(userID, filter string, deps common.Dependencies) string {
	var filterID int

	ctx, cancel := context.WithCancel(deps.Context)
	defer cancel()

	before, err := common.GetMembership(userID, deps)
	if err != nil {
		return common.SendFatal(err.Error())
	}

	_, err = strconv.Atoi(userID)
	if err != nil {
		if !common.IsDiscordUser(userID) {
			return common.SendError("second argument must be a discord user")
		}
		userID = common.ExtractUserId(userID)
	}

	err = deps.DB.Select("id").
		From("filters").
		Where(sq.Eq{"name": filter}).
		Scan(&filterID)
	if err != nil {
		newErr := fmt.Errorf("error scanning filterID: %s", err)
		deps.Logger.Error(newErr)
		return common.SendFatal(newErr.Error())
	}

	rows, err := deps.DB.Delete("filter_membership").
		Where(sq.Eq{"filter": filterID}).
		Where(sq.Eq{"user_id": userID}).
		Suffix("RETURNING \"id\"").
		QueryContext(ctx)
	if err != nil {
		newErr := fmt.Errorf("error deleting filter: %s", err)
		deps.Logger.Error(newErr)
		return common.SendFatal(newErr.Error())
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			deps.Logger.Errorf("error closing database: %s", err)
		}
	}()

	after, err := common.GetMembership(userID, deps)
	if err != nil {
		return common.SendFatal(err.Error())
	}

	removeSet := before.Difference(after)

	if removeSet.Len() == 0 {
		return common.SendError(fmt.Sprintf("<@%s> not a member of `%s`", userID, filter))
	}

	for _, role := range removeSet.ToSlice() {
		QueueUpdate(payloads.Delete, userID, role, deps)
	}

	return common.SendSuccess(fmt.Sprintf("Removed <@%s> from `%s`", userID, filter))
}

func QueueUpdate(action payloads.Action, memberID, roleID string, deps common.Dependencies) {
	payload := payloads.MemberPayload{
		Action:   action,
		GuildID:  deps.GuildID,
		MemberID: memberID,
		RoleID:   roleID,
	}

	b, err := json.Marshal(payload)
	if err != nil {
		deps.Logger.Errorf("error marshalling queue message: %s", err)
	}

	deps.Logger.Debugf("Submitting member queue message: %+v", payload)
	err = deps.MembersProducer.Publish(b)
	if err != nil {
		deps.Logger.Errorf("error publishing message: %s", err)
	}
}
