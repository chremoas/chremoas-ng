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
	"github.com/lib/pq"
	"github.com/nsqio/go-nsq"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/payloads"
)

var (
	// Role keys are database columns we're allowed up update
	roleKeys   = []string{"Name", "Color", "Hoist", "Position", "Permissions", "Joinable", "Managed", "Mentionable", "Sync"}
	roleTypes  = []string{"internal", "discord"}
	clientType = map[bool]string{true: "SIG", false: "Role"}
)

const (
	Role = false
	Sig  = true
)

var roleType = map[bool]string{Role: "role", Sig: "sig"}

func List(sig, all bool, logger *zap.SugaredLogger, db *sq.StatementBuilderType) string {
	var buffer bytes.Buffer
	var roleList = make(map[string]string)

	roles, err := getRoles(sig, nil, logger, db)
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

// Members lists all userIDs that match all the filters for a role. I'm not sure if we need to care about
// sig or not here just yet.
func Members(sig bool, name string, logger *zap.SugaredLogger, db *sq.StatementBuilderType) string {
	var (
		buffer bytes.Buffer
		err    error
		id     int
		userID int
		count  int
	)

	id, err = getRoleID(name, db)
	if err != nil {
		if err == sql.ErrNoRows {
			return common.SendError(fmt.Sprintf("No such %s: %s", roleType[sig], name))
		}
		logger.Error(err)
		return common.SendError(fmt.Sprintf("error getting filter ID (%s): %s", name, err.Error()))
	}

	rows, err := db.Select("user_id").
		From("filter_membership").
		InnerJoin("role_filters USING (filter)").
		Join("roles ON role_filters.role = roles.id").
		Where(sq.Eq{"role_filters.role": id}).
		Where(sq.Eq{"roles.sig": sig}).
		Where(sq.Eq{"filter_membership.namespace": viper.GetString("namespace")}).
		Query()
	if err != nil {
		logger.Error(err)
		return common.SendError(err.Error())
	}

	for rows.Next() {
		err = rows.Scan(&userID)
		if err != nil {
			return common.SendError(fmt.Sprintf("error scanning user_id (%s): %s", name, err.Error()))
		}

		buffer.WriteString(fmt.Sprintf("%d", userID))
		count += 1
	}

	if count == 0 {
		return common.SendError(fmt.Sprintf("No members in: %s", name))
	}

	return fmt.Sprintf("```%s```", buffer.String())
}

// getRoles goes and fetches all the roles of type sig/role. If shortname is set only one role is fetched.
func getRoles(sig bool, shortName *string, logger *zap.SugaredLogger, db *sq.StatementBuilderType) ([]payloads.Role, error) {
	var (
		rs        []payloads.Role
		charTotal int
	)

	q := db.Select("color", "hoist", "joinable", "managed", "mentionable", "name", "permissions",
		"position", "role_nick", "sig", "sync").
		Where(sq.Eq{"sig": sig}).
		From("roles").
		Where(sq.Eq{"namespace": viper.GetString("namespace")})

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

func getRoleID(name string, db *sq.StatementBuilderType) (int, error) {
	var (
		err error
		id  int
	)

	err = db.Select("id").
		From("roles").
		Where(sq.Eq{"role_nick": name}).
		Where(sq.Eq{"namespace": viper.GetString("namespace")}).
		QueryRow().Scan(&id)

	return id, err
}

func ListUserRoles(sig bool, userID string, logger *zap.SugaredLogger, db *sq.StatementBuilderType) string {
	var (
		name      string
		shortName string
		buffer    bytes.Buffer
		count     int
	)

	rows, err := db.Select("roles.role_nick", "roles.name").
		From("filters").
		Join("filter_membership ON filters.id = filter_membership.filter").
		Join("role_filters ON filters.id = role_filters.filter").
		Join("roles ON role_filters.role = roles.id").
		Where(sq.Eq{"filter_membership.user_id": userID}).
		Where(sq.Eq{"roles.sig": sig}).
		Where(sq.Eq{"filters.namespace": viper.GetString("namespace")}).
		Query()
	if err != nil {
		logger.Error(err)
		return common.SendError(fmt.Sprintf("error getting user %ss (%s): %s", roleType[sig], userID, err))
	}
	defer func() {
		if err = rows.Close(); err != nil {
			logger.Error(err)
		}
	}()

	for rows.Next() {
		err = rows.Scan(&shortName, &name)
		if err != nil {
			return common.SendError(fmt.Sprintf("error scanning %s for userID (%s): %s", roleType[sig], userID, err))
		}

		buffer.WriteString(fmt.Sprintf("%s - %s", shortName, name))
		count += 1
	}

	if count == 0 {
		return common.SendError(fmt.Sprintf("User has no %ss: <@%s>", roleType[sig], userID))
	}

	return fmt.Sprintf("```%s```", buffer.String())
}

func Info(sig bool, shortName string, logger *zap.SugaredLogger, db *sq.StatementBuilderType) string {
	var buffer bytes.Buffer

	// TODO: Wire up permissions
	//canPerform, err := r.Permissions.CanPerform(ctx, sender)
	//if err != nil {
	//	return common.SendFatal(err.Error())
	//}

	//if !canPerform {
	//	return common.SendError("User doesn't have permission to this command")
	//}

	roles, err := getRoles(sig, &shortName, logger, db)
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

func Add(sig, joinable bool, shortName, name, chatType string, logger *zap.SugaredLogger, db *sq.StatementBuilderType, nsq *nsq.Producer) string {
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
	_, err := db.Insert("roles").
		Columns("namespace", "sig", "joinable", "name", "role_nick", "chat_type").
		Values(viper.GetString("namespace"), sig, joinable, name, shortName, chatType).
		Query()
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

	payload := payloads.Payload{
		Action: payloads.Create,
		Role:   &role,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("error:", err)
	}

	topic := fmt.Sprintf("%s-discord.role", viper.GetString("namespace"))
	err = nsq.PublishAsync(topic, b, nil)
	if err != nil {
		fmt.Println("error:", err)
	}

	// We now need to create the default filter for this role
	filterResponse := filters.Add(
		shortName,
		fmt.Sprintf("Auto-created filter for %s %s", roleType[sig], shortName),
		sig,
		logger,
		db,
	)

	return fmt.Sprintf("%s%s", filterResponse, common.SendSuccess(fmt.Sprintf("Created %s `%s`", roleType[sig], shortName)))
}

func Destroy(sig bool, shortName string, logger *zap.SugaredLogger, db *sq.StatementBuilderType, nsq *nsq.Producer) string {
	var roleID int
	if len(shortName) == 0 {
		return common.SendError("short name is required")
	}

	err := db.Select("chat_id").
		From("roles").
		Where(sq.Eq{"role_nick": shortName}).
		Where(sq.Eq{"sig": sig}).
		Where(sq.Eq{"namespace": viper.GetString("namespace")}).
		QueryRow().Scan(&roleID)
	if err != nil {
		logger.Error(err)
		return common.SendFatal(fmt.Sprintf("error deleting %s: %s", roleType[sig], err))
	}

	_, err = db.Delete("roles").
		Where(sq.Eq{"role_nick": shortName}).
		Where(sq.Eq{"sig": sig}).
		Where(sq.Eq{"namespace": viper.GetString("namespace")}).
		Query()
	if err != nil {
		logger.Error(err)
		return common.SendFatal(fmt.Sprintf("error deleting %s: %s", roleType[sig], err))
	}

	payload := payloads.Payload{
		Action: payloads.Delete,
		Role: &discordgo.Role{
			ID: fmt.Sprintf("%d", roleID),
		},
	}

	b, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("error:", err)
	}

	topic := fmt.Sprintf("%s-discord.role", viper.GetString("namespace"))
	err = nsq.PublishAsync(topic, b, nil)
	if err != nil {
		fmt.Println("error:", err)
	}

	// We now need to create the default filter for this role
	filterResponse := filters.Delete(shortName, sig, logger, db)

	return fmt.Sprintf("%s%s", filterResponse, common.SendSuccess(fmt.Sprintf("Destroyed %s `%s`", roleType[sig], shortName)))
}

func validListItem(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func Update(sig bool, shortName, key, value string, logger *zap.SugaredLogger, db *sq.StatementBuilderType, nsq *nsq.Producer) string {
	var chatID int

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

	if !validListItem(key, roleKeys) {
		return common.SendError(fmt.Sprintf("`%s` isn't a valid Role Key", key))
	}

	err := db.Select("chat_id").
		From("roles").
		Where(sq.Eq{"role_nick": shortName}).
		Where(sq.Eq{"sig": sig}).
		Where(sq.Eq{"namespace": viper.GetString("namespace")}).
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
		Set(key, value).
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

	role.ID = fmt.Sprintf("%d", chatID)

	payload := payloads.Payload{
		Action: payloads.Update,
		Role:   &role,
	}

	b, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("error:", err)
	}

	topic := fmt.Sprintf("%s-discord.role", viper.GetString("namespace"))
	err = nsq.PublishAsync(topic, b, nil)
	if err != nil {
		fmt.Println("error:", err)
	}

	return common.SendSuccess(fmt.Sprintf("Updated %s `%s`", roleType[sig], shortName))
}

func SigAction(sender, sig string, join, joinable bool, logger *zap.SugaredLogger, db *sq.StatementBuilderType) string {
	s := strings.Split(sender, ":")

	roles, err := getRoles(Sig, &sig, logger, db)
	if err != nil {
		return common.SendError(err.Error())
	}
	if len(roles) == 0 {
		return common.SendError(fmt.Sprintf("No such sig: `%s`", sig))
	}
	role := roles[0]

	if !role.Sig {
		return common.SendError("Not a SIG")
	}

	// Is this a joinable role? Only check on Join/Leave not Add/Remove
	if joinable {
		if !role.Joinable {
			return common.SendError(fmt.Sprintf("'%s' is not a joinable SIG, talk to an admin", sig))
		}
	}

	// add member to role
	if join {
		filters.AddMember(Sig, s[1], sig, logger, db)
	} else {
		filters.RemoveMember(Sig, s[1], sig, logger, db)
	}

	// TODO: sync membership
	//_, err = r.RoleClient.SyncToChatService(ctx, r.GetSyncRequest(sender, false))
	//if err != nil {
	//	return common.SendError(err.Error())
	//}
	//
	//_, outputName, err := r.MapName(ctx, []string{s[1]})

	if join {
		return common.SendSuccess(fmt.Sprintf("Added %s to %s", outputName[0], sig))
	} else {
		return common.SendSuccess(fmt.Sprintf("Removed %s from %s", outputName[0], sig))
	}
}
