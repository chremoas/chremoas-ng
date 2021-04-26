package roles

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/bwmarrin/discordgo"
	"github.com/chremoas/chremoas-ng/internal/filters"
	"github.com/chremoas/chremoas-ng/internal/perms"
	"github.com/lib/pq"
	"github.com/nsqio/go-nsq"
	"go.uber.org/zap"

	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/payloads"
)

var (
	// Role keys are database columns we're allowed up update
	roleKeys   = []string{"Name", "Color", "Hoist", "Position", "Permissions", "Joinable", "Managed", "Mentionable", "Sync"}
	roleTypes  = []string{"internal", "discord"}
	clientType = map[bool]string{true: "SIG", false: "Role"}
	adminType  = map[bool]string{true: "sig_admins", false: "role_admins"}
)

const (
	Role       = false
	Sig        = true
	PollerUser = "esi-poller"
)

var roleType = map[bool]string{Role: "role", Sig: "sig"}

func List(sig, all bool, logger *zap.SugaredLogger, db *sq.StatementBuilderType) string {
	var buffer bytes.Buffer
	var roleList = make(map[string]string)

	roles, err := GetRoles(sig, nil, logger, db)
	if err != nil {
		return common.SendFatal(err.Error())
	}

	for _, role := range roles {
		if sig && !role.Joinable && !all {
			continue
		}
		roleList[role.ShortName] = role.Name
	}

	if len(roleList) == 0 {
		return common.SendError(fmt.Sprintf("No %ss\n", clientType[sig]))
	}

	buffer.WriteString(fmt.Sprintf("%ss:\n", clientType[sig]))
	for role := range roleList {
		if sig {
			buffer.WriteString(fmt.Sprintf("\t%s: %s\n", role, roleList[role]))
		} else {
			buffer.WriteString(fmt.Sprintf("\t%s\n", role))
		}
	}

	return fmt.Sprintf("```%s```", buffer.String())
}

func Keys() string {
	var buffer bytes.Buffer
	buffer.WriteString("```")

	for _, key := range roleKeys {
		buffer.WriteString(fmt.Sprintf("%s\n", key))
	}

	buffer.WriteString("```")
	return buffer.String()
}

func Types() string {
	var buffer bytes.Buffer
	buffer.WriteString("```")

	for _, key := range roleTypes {
		buffer.WriteString(fmt.Sprintf("%s\n", key))
	}

	buffer.WriteString("```")
	return buffer.String()
}

// Members lists all userIDs that match all the filters for a role.
func Members(sig bool, name string, logger *zap.SugaredLogger, db *sq.StatementBuilderType) string {
	var (
		buffer bytes.Buffer
	)

	members, err := GetRoleMembers(sig, name, logger, db)
	if err != nil {
		return common.SendError(fmt.Sprintf("error getting member list: %s", err))
	}

	if len(members) == 0 {
		return common.SendError(fmt.Sprintf("No members in: %s", name))
	}

	for _, userID := range members {
		buffer.WriteString(fmt.Sprintf("\t%d\n", userID))
	}

	return fmt.Sprintf("```%d member(s) in %s:\n%s```", len(members), name, buffer.String())
}

func ListUserRoles(sig bool, userID string, logger *zap.SugaredLogger, db *sq.StatementBuilderType) string {
	var (
		buffer bytes.Buffer
	)

	roles, err := GetUserRoles(sig, userID, logger, db)
	if err != nil {
		return common.SendError(fmt.Sprintf("error getting user roles: %s", err))
	}

	if len(roles) == 0 {
		return common.SendError(fmt.Sprintf("User has no %ss: <@%s>", roleType[sig], userID))
	}

	for _, role := range roles {
		buffer.WriteString(fmt.Sprintf("%s - %s\n", role.ShortName, role.Name))
	}

	return fmt.Sprintf("```%s```", buffer.String())
}

func Info(sig bool, shortName string, logger *zap.SugaredLogger, db *sq.StatementBuilderType) string {
	var buffer bytes.Buffer

	//if !canPerform {
	//	return common.SendError("User doesn't have permission to this command")
	//}

	roles, err := GetRoles(sig, &shortName, logger, db)
	if err != nil {
		return common.SendFatal(err.Error())
	}

	if len(roles) == 0 {
		return common.SendError(fmt.Sprintf("no such %s: %s", roleType[sig], shortName))
	}

	buffer.WriteString(fmt.Sprintf("ShortName: %s\n", roles[0].ShortName))
	buffer.WriteString(fmt.Sprintf("Name: %s\n", roles[0].Name))
	buffer.WriteString(fmt.Sprintf("Color: #%06x\n", roles[0].Color))
	buffer.WriteString(fmt.Sprintf("Hoist: %t\n", roles[0].Hoist))
	buffer.WriteString(fmt.Sprintf("Position: %d\n", roles[0].Position))
	buffer.WriteString(fmt.Sprintf("Permissions: %d\n", roles[0].Permissions))
	buffer.WriteString(fmt.Sprintf("Manged: %t\n", roles[0].Managed))
	buffer.WriteString(fmt.Sprintf("Mentionable: %t\n", roles[0].Mentionable))
	if sig {
		buffer.WriteString(fmt.Sprintf("Joinable: %t\n", roles[0].Joinable))
	}
	buffer.WriteString(fmt.Sprintf("Sync: %t\n", roles[0].Sync))

	return fmt.Sprintf("```%s```", buffer.String())
}

func Add(sig, joinable bool, shortName, name, chatType, author string, logger *zap.SugaredLogger, db *sq.StatementBuilderType, nsq *nsq.Producer) string {
	var roleID int

	if !perms.CanPerform(author, adminType[sig], logger, db) {
		return common.SendError("User doesn't have permission to this command")
	}

	// Type, Name and ShortName are required so let's check for those
	if len(chatType) == 0 {
		return common.SendError("type is required")
	}

	if len(shortName) == 0 {
		return common.SendError("short name is required")
	}

	if len(name) == 0 {
		return common.SendError("name is requred")
	}

	if !validListItem(chatType, roleTypes) {
		return common.SendError(fmt.Sprintf("`%s` isn't a valid Role Type", chatType))
	}

	// need to pass in joinable at some point
	err := db.Insert("roles").
		Columns("sig", "joinable", "name", "role_nick", "chat_type").
		Values(sig, joinable, name, shortName, chatType).
		Suffix("RETURNING \"id\"").
		QueryRow().Scan(&roleID)
	if err != nil {
		// I don't love this but I can't find a better way right now
		if err.(*pq.Error).Code == "23505" {
			return common.SendError(fmt.Sprintf("%s `%s` (%s) already exists", roleType[sig], name, shortName))
		}
		logger.Error(err)
		return common.SendFatal(fmt.Sprintf("error adding %s: %s", roleType[sig], err))
	}

	role := discordgo.Role{
		Name:        name,
		Managed:     false,
		Mentionable: false,
		Hoist:       false,
		Color:       0,
		Position:    0,
		Permissions: 0,
	}

	// We now need to create the default filter for this role
	filterResponse, filterID := filters.Add(
		shortName,
		fmt.Sprintf("Auto-created filter for %s %s", roleType[sig], shortName),
		author,
		logger,
		db,
	)

	// Associate new filter with new role
	_, err = db.Insert("role_filters").
		Columns("role", "filter").
		Values(roleID, filterID).
		Query()
	if err != nil {
		logger.Error(err)
		return common.SendFatal(fmt.Sprintf("error adding role_filter for %s: %s", roleType[sig], err))
	}

	payload := payloads.Payload{
		Action: payloads.Create,
		Role:   &role,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("error:", err)
	}

	err = nsq.PublishAsync(common.GetTopic("role"), b, nil)
	if err != nil {
		fmt.Println("error:", err)
	}

	return fmt.Sprintf("%s%s", filterResponse, common.SendSuccess(fmt.Sprintf("Created %s `%s`", roleType[sig], shortName)))
}

func Destroy(sig bool, shortName, author string, logger *zap.SugaredLogger, db *sq.StatementBuilderType, nsq *nsq.Producer) string {
	var chatID, roleID int

	if !perms.CanPerform(author, adminType[sig], logger, db) {
		return common.SendError("User doesn't have permission to this command")
	}

	if len(shortName) == 0 {
		return common.SendError("short name is required")
	}

	err := db.Select("chat_id").
		From("roles").
		Where(sq.Eq{"role_nick": shortName}).
		Where(sq.Eq{"sig": sig}).
		QueryRow().Scan(&chatID)
	if err != nil {
		if err == sql.ErrNoRows {
			return common.SendError(fmt.Sprintf("No such %s: %s", roleType[sig], shortName))
		}
		logger.Error(err)
		return common.SendFatal(fmt.Sprintf("error deleting %s: %s", roleType[sig], err))
	}

	rows, err := db.Delete("roles").
		Where(sq.Eq{"role_nick": shortName}).
		Where(sq.Eq{"sig": sig}).
		Query()
	if err != nil {
		logger.Error(err)
		return common.SendFatal(fmt.Sprintf("error deleting %s: %s", roleType[sig], err))
	}

	for rows.Next() {
		err = rows.Scan(&roleID)
		if err != nil {
			newErr := fmt.Errorf("error scanning role id: %s", err)
			logger.Error(newErr)
			return common.SendFatal(newErr.Error())
		}
	}

	// We now need to create the default filter for this role
	filterResponse, filterID := filters.Delete(shortName, author, logger, db)

	_, err = db.Delete("filter_membership").
		Where(sq.Eq{"filter": filterID}).
		Query()
	if err != nil {
		logger.Error(err)
		return common.SendFatal(fmt.Sprintf("error deleting filter_membershipts for %s: %s", roleType[sig], err))
	}

	_, err = db.Delete("role_filters").
		Where(sq.Eq{"role": roleID}).
		Query()
	if err != nil {
		logger.Error(err)
		return common.SendFatal(fmt.Sprintf("error deleting role_filters %s: %s", roleType[sig], err))
	}

	queueUpdate(chatID, payloads.Delete, logger, nsq)

	return fmt.Sprintf("%s%s", filterResponse, common.SendSuccess(fmt.Sprintf("Destroyed %s `%s`", roleType[sig], shortName)))
}

func Update(sig bool, shortName, key, value, author string, logger *zap.SugaredLogger, db *sq.StatementBuilderType, nsq *nsq.Producer) string {
	var chatID int

	if author != PollerUser && !perms.CanPerform(author, adminType[sig], logger, db) {
		return common.SendError("User doesn't have permission to this command")
	}

	// ShortName, Key and Value are required so let's check for those
	if len(shortName) == 0 {
		return common.SendError("short name is required")
	}

	if len(key) == 0 {
		return common.SendError("type is required")
	}

	if len(value) == 0 {
		return common.SendError("name is requred")
	}

	if key == "Color" {
		if string(value[0]) == "#" {
			i, _ := strconv.ParseInt(value[1:], 16, 64)
			value = strconv.Itoa(int(i))
		}
	}

	if author != PollerUser && !validListItem(key, roleKeys) {
		return common.SendError(fmt.Sprintf("`%s` isn't a valid Role Key", key))
	}

	err := db.Select("chat_id").
		From("roles").
		Where(sq.Eq{"role_nick": shortName}).
		Where(sq.Eq{"sig": sig}).
		QueryRow().Scan(&chatID)
	if err != nil {
		if err == sql.ErrNoRows {
			return common.SendError(fmt.Sprintf("No such %s: %s", roleType[sig], shortName))
		}
		return common.SendFatal(err.Error())
	}

	if chatID == 0 {
		return common.SendError(fmt.Sprintf("%s `%s` doesn't have chatID set properly", roleType[sig], shortName))
	}

	_, err = db.Update("roles").
		Set(strings.ToLower(key), value).
		Where(sq.Eq{"chat_id": chatID}).
		Where(sq.Eq{"sig": sig}).
		Query()
	if err != nil {
		logger.Error(err)
		return common.SendFatal(fmt.Sprintf("error adding %s: %s", roleType[sig], err))
	}

	var role discordgo.Role

	err = db.Select("name", "managed", "mentionable", "hoist", "color", "position", "permissions").
		From("roles").
		Where(sq.Eq{"chat_id": chatID}).
		Where(sq.Eq{"sig": sig}).
		QueryRow().Scan(
		&role.Name,
		&role.Managed,
		&role.Mentionable,
		&role.Hoist,
		&role.Color,
		&role.Position,
		&role.Permissions,
	)
	if err != nil {
		logger.Error(err)
		return common.SendFatal(fmt.Sprintf("error fetching %s from db: %s", roleType[sig], err))
	}

	queueUpdate(chatID, payloads.Update, logger, nsq)

	return common.SendSuccess(fmt.Sprintf("Updated %s `%s`", roleType[sig], shortName))
}

func ListFilters(sig bool, shortName string, logger *zap.SugaredLogger, db *sq.StatementBuilderType) string {
	var (
		buffer  bytes.Buffer
		filter  string
		results bool
	)

	rows, err := db.Select("filters.name").
		From("filters").
		Join("role_filters ON role_filters.filter = filters.id").
		Join("roles ON roles.id = role_filters.role").
		Where(sq.Eq{"roles.role_nick": shortName}).
		Where(sq.Eq{"roles.sig": sig}).
		Query()
	if err != nil {
		logger.Error(err)
		return common.SendFatal(fmt.Sprintf("error fetching filters: %s", err))
	}

	for rows.Next() {
		if !results {
			buffer.WriteString(fmt.Sprintf("Filters for %s\n", shortName))
			results = true
		}
		err = rows.Scan(&filter)
		if err != nil {
			logger.Error(err)
			return common.SendFatal(fmt.Sprintf("error scanning row filters: %s", err))
		}

		buffer.WriteString(fmt.Sprintf("\t%s\n", filter))
	}

	if results {
		return fmt.Sprintf("```%s```", buffer.String())
	} else {
		return common.SendError(fmt.Sprintf("No such %s: %s", roleType[sig], shortName))
	}
}

func AddFilter(sig bool, filter, shortName, author string, logger *zap.SugaredLogger, db *sq.StatementBuilderType, nsq *nsq.Producer) string {
	var (
		err      error
		filterID int
		roleID   int
	)

	if !perms.CanPerform(author, adminType[sig], logger, db) {
		return common.SendError("User doesn't have permission to this command")
	}

	err = db.Select("id").
		From("filters").
		Where(sq.Eq{"name": filter}).
		QueryRow().Scan(&filterID)
	if err != nil {
		if err == sql.ErrNoRows {
			return common.SendError(fmt.Sprintf("No such filter: %s", filter))
		}
		logger.Error(err)
		return common.SendFatal(fmt.Sprintf("error fetching filter id: %s", err))
	}

	err = db.Select("id").
		From("roles").
		Where(sq.Eq{"role_nick": shortName}).
		Where(sq.Eq{"sig": sig}).
		QueryRow().Scan(&roleID)
	if err != nil {
		if err == sql.ErrNoRows {
			return common.SendError(fmt.Sprintf("No such %s: %s", roleType[sig], filter))
		}
		logger.Error(err)
		return common.SendFatal(fmt.Sprintf("error fetching %s id: %s", roleType[sig], err))
	}

	_, err = db.Insert("role_filters").
		Columns("role", "filter").
		Values(roleID, filterID).
		Query()
	if err != nil {
		logger.Error(err)
		return common.SendFatal(fmt.Sprintf("error inserting role_filter: %s", err))
	}

	return common.SendSuccess(fmt.Sprintf("Added filter %s to role %s", filter, shortName))
}

func RemoveFilter(sig bool, filter, shortName, author string, logger *zap.SugaredLogger, db *sq.StatementBuilderType, nsq *nsq.Producer) string {
	var (
		err      error
		filterID int
		roleID   int
	)

	if !perms.CanPerform(author, adminType[sig], logger, db) {
		return common.SendError("User doesn't have permission to this command")
	}

	err = db.Select("id").
		From("filters").
		Where(sq.Eq{"name": filter}).
		QueryRow().Scan(&filterID)
	if err != nil {
		if err == sql.ErrNoRows {
			return common.SendError(fmt.Sprintf("No such filter: %s", filter))
		}
		logger.Error(err)
		return common.SendFatal(fmt.Sprintf("error fetching filter id: %s", err))
	}

	err = db.Select("id").
		From("roles").
		Where(sq.Eq{"role_nick": shortName}).
		Where(sq.Eq{"sig": sig}).
		QueryRow().Scan(&roleID)
	if err != nil {
		if err == sql.ErrNoRows {
			return common.SendError(fmt.Sprintf("No such %s: %s", roleType[sig], filter))
		}
		logger.Error(err)
		return common.SendFatal(fmt.Sprintf("error fetching %s id: %s", roleType[sig], err))
	}

	_, err = db.Delete("role_filters").
		Where(sq.Eq{"role": roleID}).
		Where(sq.Eq{"filter": filterID}).
		Query()
	if err != nil {
		logger.Error(err)
		return common.SendFatal(fmt.Sprintf("error deleting role_filter: %s", err))
	}

	return common.SendSuccess(fmt.Sprintf("Removed filter %s from role %s", filter, shortName))
}
