package perms

import (
	"bytes"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	"github.com/lib/pq"
)

func List(deps common.Dependencies) string {
	var (
		count  int
		buffer bytes.Buffer
		filter payloads.Filter
	)

	rows, err := deps.DB.Select("name", "description").
		From("permissions").
		Query()
	if err != nil {
		deps.Logger.Error(err)
		return common.SendFatal(err.Error())
	}

	buffer.WriteString("Permissions:\n")
	for rows.Next() {
		err = rows.Scan(&filter.Name, &filter.Description)
		if err != nil {
			newErr := fmt.Errorf("error scanning permissions row: %s", err)
			deps.Logger.Error(newErr)
			return common.SendFatal(newErr.Error())
		}

		buffer.WriteString(fmt.Sprintf("\t%s: %s\n", filter.Name, filter.Description))
		count += 1
	}

	if count == 0 {
		return common.SendError("No permissions")
	}

	return fmt.Sprintf("```%s```", buffer.String())
}

func Add(name, description, author string, deps common.Dependencies) string {
	if name == "server_admins" {
		return common.SendError("User doesn't have permission to this command")
	}

	if !CanPerform(author, "server_admins", deps) {
		return common.SendError("User doesn't have permission to this command")
	}

	_, err := deps.DB.Insert("permissions").
		Columns("name", "description").
		Values(name, description).
		Query()
	if err != nil {
		// I don't love this but I can't find a better way right now
		if err.(*pq.Error).Code == "23505" {
			return common.SendError(fmt.Sprintf("permission `%s` already exists", name))
		}
		newErr := fmt.Errorf("error inserting permission: %s", err)
		deps.Logger.Error(newErr)
		return common.SendFatal(newErr.Error())
	}

	return common.SendSuccess(fmt.Sprintf("Created permission `%s`", name))
}

func Delete(name, author string, deps common.Dependencies) string {
	if name == "server_admins" {
		return common.SendError("User doesn't have permission to this command")
	}

	if !CanPerform(author, "server_admins", deps) {
		return common.SendError("User doesn't have permission to this command")
	}

	_, err := deps.DB.Delete("permissions").
		Where(sq.Eq{"name": name}).
		Query()
	if err != nil {
		newErr := fmt.Errorf("error deleting permission: %s", err)
		deps.Logger.Error(newErr)
		return common.SendFatal(newErr.Error())
	}

	return common.SendSuccess(fmt.Sprintf("Deleted permission `%s`", name))
}

func ListMembers(name string, deps common.Dependencies) string {
	var (
		count, userID int
		buffer        bytes.Buffer
	)

	rows, err := deps.DB.Select("user_id").
		From("permission_membership").
		Join("permissions ON permission_membership.permission = permissions.id").
		Where(sq.Eq{"permissions.name": name}).
		Query()
	if err != nil {
		newErr := fmt.Errorf("error getting permission membership list: %s", err)
		deps.Logger.Error(newErr)
		return common.SendFatal(newErr.Error())
	}

	buffer.WriteString(fmt.Sprintf("Permission membership (%s):\n", name))
	for rows.Next() {
		err = rows.Scan(&userID)
		if err != nil {
			newErr := fmt.Errorf("error scanning permission_membership userID: %s", err)
			deps.Logger.Error(newErr)
			return common.SendFatal(newErr.Error())
		}
		buffer.WriteString(fmt.Sprintf("\t<@%s>\n", common.GetUsername(userID, deps.Session)))
		count += 1
	}

	if count == 0 {
		return common.SendError(fmt.Sprintf("Permission has no members: %s", name))
	}

	return buffer.String()
}

func AddMember(user, permission, author string, deps common.Dependencies) string {
	var permissionID int

	if permission == "server_admins" {
		return common.SendError("User doesn't have permission to this command")
	}

	if !CanPerform(author, "server_admins", deps) {
		return common.SendError("User doesn't have permission to this command")
	}

	if !common.IsDiscordUser(user) {
		return common.SendError("second argument must be a discord user")
	}

	userID := common.ExtractUserId(user)

	err := deps.DB.Select("id").
		From("permissions").
		Where(sq.Eq{"name": permission}).
		QueryRow().Scan(&permissionID)
	if err != nil {
		newErr := fmt.Errorf("error scanning permissionID: %s", err)
		deps.Logger.Error(newErr)
		return common.SendFatal(newErr.Error())
	}

	_, err = deps.DB.Insert("permission_membership").
		Columns("permission", "user_id").
		Values(permissionID, userID).
		Query()
	if err != nil {
		// I don't love this but I can't find a better way right now
		if err.(*pq.Error).Code == "23505" {
			return common.SendError(fmt.Sprintf("<@%s> already a member of `%s`", userID, permission))
		}
		newErr := fmt.Errorf("error inserting permission: %s", err)
		deps.Logger.Error(newErr)
		return common.SendFatal(newErr.Error())
	}

	return common.SendSuccess(fmt.Sprintf("Added <@%s> to `%s`", userID, permission))
}

func RemoveMember(user, permission, author string, deps common.Dependencies) string {
	var permissionID int

	if permission == "server_admins" {
		return common.SendError("User doesn't have permission to this command")
	}

	if !CanPerform(author, "server_admins", deps) {
		return common.SendError("User doesn't have permission to this command")
	}

	if !common.IsDiscordUser(user) {
		return common.SendError("second argument must be a discord user")
	}

	userID := common.ExtractUserId(user)

	err := deps.DB.Select("id").
		From("permissions").
		Where(sq.Eq{"name": permission}).
		QueryRow().Scan(&permissionID)
	if err != nil {
		newErr := fmt.Errorf("error scanning permisionID: %s", err)
		deps.Logger.Error(newErr)
		return common.SendFatal(newErr.Error())
	}

	_, err = deps.DB.Delete("permission_membership").
		Where(sq.Eq{"permission": permissionID}).
		Where(sq.Eq{"user_id": userID}).
		Query()
	if err != nil {
		newErr := fmt.Errorf("error deleting permission: %s", err)
		deps.Logger.Error(newErr)
		return common.SendFatal(newErr.Error())
	}

	return common.SendSuccess(fmt.Sprintf("Removed <@%s> from `%s`", userID, permission))
}

func UserPerms(user string, deps common.Dependencies) string {
	var (
		buffer     bytes.Buffer
		permission string
	)

	if !common.IsDiscordUser(user) {
		return common.SendError("second argument must be a discord user")
	}

	userID := common.ExtractUserId(user)

	rows, err := deps.DB.Select("name").
		From("permissions").
		Join("permission_membership ON permission_membership.permission = permissions.id").
		Where(sq.Eq{"permission_membership.user_id": userID}).
		Query()
	if err != nil {
		newErr := fmt.Errorf("error getting user perms: %s", err)
		deps.Logger.Error(newErr)
		return common.SendFatal(newErr.Error())
	}

	buffer.WriteString(fmt.Sprintf("Permissions for <@%s>:\n", userID))
	for rows.Next() {
		err = rows.Scan(&permission)
		if err != nil {
			deps.Logger.Errorf("Error scanning permission id: %s", err)
			return common.SendFatal(err.Error())
		}
		buffer.WriteString(fmt.Sprintf("\t%s\n", permission))
	}

	return buffer.String()
}

func CanPerform(authorID, permission string, deps common.Dependencies) bool {
	var (
		count        int
		permissionID int
	)

	// This is super jank and I don't like it, need to come up with a better way.
	if authorID == "auth-web" {
		return true
	}

	err := deps.DB.Select("id").
		From("permissions").
		Where(sq.Eq{"name": permission}).
		QueryRow().Scan(&permissionID)
	if err != nil {
		newErr := fmt.Errorf("error scanning permisionID: %s", err)
		deps.Logger.Error(newErr)
		return false
	}
	err = deps.DB.Select("COUNT(*)").
		From("permission_membership").
		Where(sq.Eq{"user_id": authorID}).
		Where(sq.Eq{"permission": permissionID}).
		QueryRow().Scan(&count)
	if err != nil {
		newErr := fmt.Errorf("error scanning permission count: %s", err)
		deps.Logger.Error(newErr)
		return false
	}

	if count == 0 {
		return false
	}

	return true
}
