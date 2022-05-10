package roles

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"

	sq "github.com/Masterminds/squirrel"
	sl "github.com/bhechinger/spiffylogger"
	"github.com/bwmarrin/discordgo"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/filters"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	"github.com/chremoas/chremoas-ng/internal/perms"
	"github.com/lib/pq"
	"go.uber.org/zap"
)

var (
	// Role keys are database columns we're allowed up update
	roleKeys   = []string{"Name", "Color", "Hoist", "Position", "Permissions", "Joinable", "Managed", "Mentionable", "Sync"}
	roleTypes  = []string{"internal", "discord"}
	clientType = map[bool]string{true: "SIG", false: "Role"}
	adminType  = map[bool]string{true: "sig_admins", false: "role_admins"}
)

const (
	Role = false
	Sig  = true
)

var roleType = map[bool]string{Role: "role", Sig: "sig"}

func List(ctx context.Context, sig, all bool, channelID string, deps common.Dependencies) []*discordgo.MessageSend {
	_, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("command", "list"),
		zap.String("role_type", roleType[sig]),
		zap.Bool("all", all),
	)

	roles, err := GetRoles(ctx, sig, nil, deps)
	if err != nil {
		sp.Error("error getting roles", zap.Error(err))
		return common.SendFatal(err.Error())
	}

	roleList := make([]string, 0, len(roles))

	for _, role := range roles {
		if sig && !role.Joinable && !all {
			continue
		}
		roleList = append(roleList, fmt.Sprintf("%s: %s", role.ShortName, role.Name))
	}
	sort.Strings(roleList)

	if len(roleList) == 0 {
		return common.SendError(fmt.Sprintf("No %ss\n", clientType[sig]))
	}

	err = common.SendChunkedMessage(ctx, channelID, fmt.Sprintf("%s List", clientType[sig]), roleList, deps)
	if err != nil {
		sp.Error("Error sending message")
		return common.SendError("Error sending message")
	}

	return nil
}

func Keys() []*discordgo.MessageSend {
	var (
		buffer   bytes.Buffer
		messages []*discordgo.MessageSend
	)

	for _, key := range roleKeys {
		buffer.WriteString(fmt.Sprintf("%s\n", key))
	}

	embed := common.NewEmbed()
	embed.SetTitle("Keys")
	embed.SetDescription(buffer.String())

	return append(messages, &discordgo.MessageSend{Embed: embed.GetMessageEmbed()})
}

func Types() []*discordgo.MessageSend {
	var (
		buffer   bytes.Buffer
		messages []*discordgo.MessageSend
	)

	for _, key := range roleTypes {
		buffer.WriteString(fmt.Sprintf("%s\n", key))
	}

	embed := common.NewEmbed()
	embed.SetTitle("Types")
	embed.SetDescription(buffer.String())

	return append(messages, &discordgo.MessageSend{Embed: embed.GetMessageEmbed()})
}

// ListMembers lists all userIDs that match all the filters for a role.
func ListMembers(ctx context.Context, sig bool, name string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("role_type", roleType[sig]),
		zap.String("name", name),
	)

	var (
		buffer   bytes.Buffer
		messages []*discordgo.MessageSend
	)

	sp.Debug("Listing members")

	members, err := GetRoleMembers(ctx, sig, name, deps)
	if err != nil {
		sp.Error("error getting member list", zap.Error(err))
		return common.SendError(fmt.Sprintf("error getting member list: %s", err))
	}

	if len(members) == 0 {
		sp.Info("no members in role")
		return common.SendError(fmt.Sprintf("No members in: %s", name))
	}

	for _, userID := range members {
		buffer.WriteString(fmt.Sprintf("%s\n", common.GetUsername(userID, deps.Session)))
	}

	embed := common.NewEmbed()
	embed.SetTitle(fmt.Sprintf("%d member(s) in %s", len(members), name))
	embed.SetDescription(buffer.String()) // TODO check if description is really what I want here.
	return append(messages, &discordgo.MessageSend{Embed: embed.GetMessageEmbed()})
}

func ListUserRoles(ctx context.Context, sig bool, userID string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("role_type", roleType[sig]),
		zap.String("user_id", userID),
	)

	var (
		buffer   bytes.Buffer
		messages []*discordgo.MessageSend
	)

	roles, err := common.GetUserRoles(ctx, sig, userID, deps)
	if err != nil {
		sp.Error("error getting user roles", zap.Error(err))
		return common.SendError(fmt.Sprintf("error getting user roles: %s", err))
	}

	if len(roles) == 0 {
		sp.Info("user has no roles")
		return common.SendError(fmt.Sprintf("User has no %ss: <@%s>", roleType[sig], userID))
	}

	for _, role := range roles {
		buffer.WriteString(fmt.Sprintf("%s - %s\n", role.ShortName, role.Name))
	}

	embed := common.NewEmbed()
	embed.SetTitle(fmt.Sprintf("Roles for %s", common.GetUsername(userID, deps.Session)))
	embed.SetDescription(buffer.String())

	return append(messages, &discordgo.MessageSend{Embed: embed.GetMessageEmbed()})
}

func Info(ctx context.Context, sig bool, ticker string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("role_type", roleType[sig]),
		zap.String("ticker", ticker),
	)

	var (
		buffer   bytes.Buffer
		messages []*discordgo.MessageSend
	)

	roles, err := GetRoles(ctx, sig, &ticker, deps)
	if err != nil {
		sp.Error("error getting roles", zap.Error(err))
		return common.SendFatal(err.Error())
	}

	if len(roles) == 0 {
		sp.Error("no such role")
		return common.SendError(fmt.Sprintf("no such %s: %s", roleType[sig], ticker))
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

	embed := common.NewEmbed()
	embed.SetTitle(fmt.Sprintf("Info for %s %s", roleType[sig], roles[0].Name))
	embed.SetDescription(buffer.String())

	return append(messages, &discordgo.MessageSend{Embed: embed.GetMessageEmbed()})
}

func AuthedAdd(ctx context.Context, sig, joinable bool, ticker, name, chatType, author string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("role_type", roleType[sig]),
		zap.Bool("joinable", joinable),
		zap.String("ticker", ticker),
		zap.String("name", name),
		zap.String("chat_type", chatType),
		zap.String("author", author),
	)

	if !perms.CanPerform(ctx, author, adminType[sig], deps) {
		sp.Warn("user doesn't have permission to this command")
		return common.SendError("User doesn't have permission to this command")
	}

	sp.Debug("adding role")
	return Add(ctx, sig, joinable, ticker, name, chatType, deps)
}

func Add(ctx context.Context, sig, joinable bool, ticker, name, chatType string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("role_type", roleType[sig]),
		zap.Bool("joinable", joinable),
		zap.String("ticker", ticker),
		zap.String("name", name),
		zap.String("chatType", chatType),
	)

	var roleID int

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Type, Name and ShortName are required so let's check for those
	if len(chatType) == 0 {
		return common.SendError("type is required")
	}

	if len(ticker) == 0 {
		return common.SendError("short name is required")
	}

	if len(name) == 0 {
		return common.SendError("name is required")
	}

	if !validListItem(chatType, roleTypes) {
		return common.SendError(fmt.Sprintf("`%s` isn't a valid Role Type", chatType))
	}

	insert := deps.DB.Insert("roles").
		Columns("sig", "joinable", "name", "role_nick", "chat_type", "sync").
		// a sig is sync-ed by default, so we overload the sig bool because it does the right thing here.
		Values(sig, joinable, name, ticker, chatType, sig).
		Suffix("RETURNING \"id\"")

	sqlStr, args, err := insert.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return common.SendFatal(err.Error())
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	err = insert.Scan(&roleID)
	if err != nil {
		// I don't love this but I can't find a better way right now
		if err.(*pq.Error).Code == "23505" {
			sp.Info("role already exists")
			return common.SendError(fmt.Sprintf("%s `%s` (%s) already exists", roleType[sig], name, ticker))
		}
		sp.Error("error adding role", zap.Error(err))
		return common.SendFatal(fmt.Sprintf("error adding %s: %s", roleType[sig], err))
	}

	sp.With(zap.Int("role_id", roleID))

	role := payloads.Role{
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
		ctx,
		ticker,
		fmt.Sprintf("Auto-created filter for %s %s", roleType[sig], ticker),
		deps,
	)

	sp.With(zap.Int("filter_id", filterID))

	// Associate new filter with new role
	insert = deps.DB.Insert("role_filters").
		Columns("role", "filter").
		Values(roleID, filterID)

	sqlStr, args, err = insert.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return common.SendFatal(err.Error())
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	rows, err := insert.QueryContext(ctx)
	if err != nil {
		sp.Error("error adding role_filter", zap.Error(err))
		return common.SendFatal(fmt.Sprintf("error adding role_filter for %s: %s", roleType[sig], err))
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			sp.Error("error closing database", zap.Error(err))
		}
	}()

	err = queueUpdate(ctx, role, payloads.Upsert, deps)
	if err != nil {
		sp.Error("error adding role", zap.Error(err))
		return common.SendFatal(fmt.Sprintf("error adding role for %s: %s", roleType[sig], err))
	}

	sp.Info("created role")
	messages := common.SendSuccess(fmt.Sprintf("Created %s `%s`", roleType[sig], ticker))

	embed := common.NewEmbed()
	embed.SetTitle("filter response")
	messages = append(messages, &discordgo.MessageSend{Embed: embed.GetMessageEmbed()})
	messages = append(messages, filterResponse...)

	return messages
}

func AuthedDestroy(ctx context.Context, sig bool, ticker, author string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("role_type", roleType[sig]),
		zap.String("ticker", ticker),
		zap.String("author", author),
	)

	if !perms.CanPerform(ctx, author, adminType[sig], deps) {
		sp.Warn("user doesn't have permission to this command")
		return common.SendError("User doesn't have permission to this command")
	}

	sp.Debug("destroying role")
	return Destroy(ctx, sig, ticker, deps)
}

func Destroy(ctx context.Context, sig bool, ticker string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("role_type", roleType[sig]),
		zap.String("ticker", ticker),
	)

	var chatID, roleID int

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if len(ticker) == 0 {
		return common.SendError("short name is required")
	}

	query := deps.DB.Select("chat_id").
		From("roles").
		Where(sq.Eq{"role_nick": ticker}).
		Where(sq.Eq{"sig": sig})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return common.SendFatal(err.Error())
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	err = query.Scan(&chatID)
	if err != nil {
		if err == sql.ErrNoRows {
			sp.Error("no such role", zap.Error(err))
			return common.SendError(fmt.Sprintf("No such %s: %s", roleType[sig], ticker))
		}
		sp.Error("error deleting role", zap.Error(err))
		return common.SendFatal(fmt.Sprintf("error deleting %s: %s", roleType[sig], err))
	}

	sp.With(zap.Int("chat_id", chatID))

	delQuery := deps.DB.Delete("roles").
		Where(sq.Eq{"role_nick": ticker}).
		Where(sq.Eq{"sig": sig})

	sqlStr, args, err = delQuery.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return common.SendFatal(err.Error())
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	rows, err := delQuery.QueryContext(ctx)
	if err != nil {
		sp.Error("error deleting role", zap.Error(err))
		return common.SendFatal(fmt.Sprintf("error deleting %s: %s", roleType[sig], err))
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			sp.Error("error closing database", zap.Error(err))
		}
	}()

	for rows.Next() {
		err = rows.Scan(&roleID)
		if err != nil {
			sp.Error("error scanning role id", zap.Error(err))
			return common.SendFatal(err.Error())
		}
	}

	// We now need to create the default filter for this role
	filterResponse, filterID := filters.Delete(ctx, ticker, deps)

	sp.With(
		zap.Int("role_id", roleID),
		zap.Int("filter_id", filterID),
	)

	delQuery = deps.DB.Delete("filter_membership").
		Where(sq.Eq{"filter": filterID})

	sqlStr, args, err = delQuery.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return common.SendFatal(err.Error())
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	rows, err = delQuery.QueryContext(ctx)
	if err != nil {
		sp.Error("error deleting filter_membership", zap.Error(err))
		return common.SendFatal(fmt.Sprintf("error deleting filter_memberships for %s: %s", roleType[sig], err))
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			sp.Error("error closing database", zap.Error(err))
		}
	}()

	delQuery = deps.DB.Delete("role_filters").
		Where(sq.Eq{"role": roleID})

	sqlStr, args, err = delQuery.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return common.SendFatal(err.Error())
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	rows, err = delQuery.QueryContext(ctx)
	if err != nil {
		sp.Error("error deleting role_filters", zap.Error(err))
		return common.SendFatal(fmt.Sprintf("error deleting role_filters %s: %s", roleType[sig], err))
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			sp.Error("error closing database", zap.Error(err))
		}
	}()

	err = queueUpdate(ctx, payloads.Role{ID: fmt.Sprintf("%d", chatID)}, payloads.Delete, deps)
	if err != nil {
		sp.Error("error deleting role", zap.Error(err))
		return common.SendFatal(fmt.Sprintf("error deleting role for %s: %s", roleType[sig], err))
	}

	sp.Info("deleted role")
	messages := common.SendSuccess(fmt.Sprintf("Destroyed %s `%s`", roleType[sig], ticker))

	embed := common.NewEmbed()
	embed.SetTitle("filter response")
	messages = append(messages, &discordgo.MessageSend{Embed: embed.GetMessageEmbed()})
	messages = append(messages, filterResponse...)

	return messages
}

func AuthedUpdate(ctx context.Context, sig bool, ticker, key, value, author string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("role_type", roleType[sig]),
		zap.String("ticker", ticker),
		zap.String("key", key),
		zap.String("value", value),
		zap.String("author", author),
	)

	if !perms.CanPerform(ctx, author, adminType[sig], deps) {
		sp.Warn("user doesn't have permission to this command")
		return common.SendError("User doesn't have permission to this command")
	}

	if !validListItem(key, roleKeys) {
		return common.SendError(fmt.Sprintf("`%s` isn't a valid Role Key", key))
	}

	values := map[string]string{
		key: value,
	}

	sp.Debug("updating role")
	return Update(ctx, sig, ticker, values, deps)
}

func Update(ctx context.Context, sig bool, ticker string, values map[string]string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("role_type", roleType[sig]),
		zap.String("ticker", ticker),
		zap.Any("values", values),
	)

	var (
		name string
		sync bool
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// ShortName, Key and Value are required so let's check for those
	if len(ticker) == 0 {
		return common.SendError("ticker is required")
	}

	if len(values) == 0 {
		return common.SendError("values is required")
	}

	query := deps.DB.Select("name", "sync").
		From("roles").
		Where(sq.Eq{"role_nick": ticker}).
		Where(sq.Eq{"sig": sig})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return common.SendFatal(err.Error())
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	err = query.Scan(&name, &sync)
	if err != nil {
		if err == sql.ErrNoRows {
			sp.Error("no such role")
			return common.SendError(fmt.Sprintf("No such %s: %s", roleType[sig], ticker))
		}
		sp.Error("error scanning name and sync", zap.Error(err))
		return common.SendFatal(err.Error())
	}

	sp.With(zap.Bool("sync", sync))

	updateSQL := deps.DB.Update("roles")

	for k, v := range values {
		key := strings.ToLower(k)
		if key == "color" {
			if string(v[0]) == "#" {
				i, _ := strconv.ParseInt(v[1:], 16, 64)
				v = strconv.Itoa(int(i))
			}
		}

		if key == "sync" {
			sync, err = strconv.ParseBool(v)
			if err != nil {
				sp.Warn("error updating sync", zap.Error(err))
				return common.SendFatal(fmt.Sprintf("error updating sync for %s: %s", name, err))
			}
			sp.With(zap.Bool("sync", sync))
		}

		updateSQL = updateSQL.Set(key, v)
	}

	sqlStr, args, err = updateSQL.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return common.SendFatal(err.Error())
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	_, err = updateSQL.Where(sq.Eq{"name": name}).
		Where(sq.Eq{"sig": sig}).
		QueryContext(ctx)
	if err != nil {
		sp.Error("error adding role", zap.Error(err))
		return common.SendFatal(fmt.Sprintf("error adding %s: %s", roleType[sig], err))
	}

	role, err := GetChremoasRole(ctx, sig, ticker, deps)
	if err != nil {
		sp.Error("error fetching role", zap.Error(err))
		return common.SendFatal(fmt.Sprintf("error fetching %s from db: %s", roleType[sig], err))
	}

	sp.With(zap.Any("role", role))

	dRole, err := GetDiscordRole(ctx, role.Name, deps)
	if err != nil {
		sp.Warn("error getting discord role", zap.Error(err))
		// TODO: Figure out if there are errors we should really fail on
		// return common.SendFatal(fmt.Sprintf("error fetching roles from discord: %s", err))
		err = queueUpdate(ctx, role, payloads.Upsert, deps)
		if err != nil {
			sp.Error("error sending update", zap.Error(err))
			return common.SendFatal(fmt.Sprintf("error updating role for %s: %s", roleType[sig], err))
		}

		sp.Info("updated role")
		return common.SendSuccess(fmt.Sprintf("Updated %s `%s`", roleType[sig], ticker))
	}

	sp.With(zap.Any("discord_role", dRole))

	if role.ID != dRole.ID {
		// The role was probably created/recreated manually.
		update := deps.DB.Update("roles").
			Set("ID", dRole.ID).
			Where(sq.Eq{"name": role.Name})

		sqlStr, args, err = update.ToSql()
		if err != nil {
			sp.Error("error getting sql", zap.Error(err))
			return common.SendFatal(err.Error())
		} else {
			sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
		}

		rows, err := query.QueryContext(ctx)
		if err != nil {
			sp.Error("error updating role's ID", zap.Error(err))
		}
		defer func() {
			err := rows.Close()
			if err != nil {
				sp.Error("error closing database", zap.Error(err))
			}
		}()
	}

	if !sync {
		sp.Info("updated role but didn't sync to discord")
		return common.SendSuccess(fmt.Sprintf("Updated %s in db but not Discord (sync not set): %s", roleType[sig], ticker))
	}

	if role.Mentionable != dRole.Mentionable ||
		role.Hoist != dRole.Hoist ||
		role.Color != dRole.Color ||
		role.Permissions != dRole.Permissions {
		sp.Info("Roles differ")

		err = queueUpdate(ctx, role, payloads.Upsert, deps)
		if err != nil {
			sp.Error("error updating role", zap.Error(err))
			return common.SendFatal(fmt.Sprintf("error updating role for %s: %s", roleType[sig], err))
		}
	}

	sp.Info("updated role")
	return common.SendSuccess(fmt.Sprintf("Updated %s `%s`", roleType[sig], ticker))
}

func GetChremoasRole(ctx context.Context, sig bool, ticker string, deps common.Dependencies) (payloads.Role, error) {
	_, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("role_type", roleType[sig]),
		zap.String("ticker", ticker),
	)

	var (
		role payloads.Role
		err  error
	)

	query := deps.DB.Select("chat_id", "name", "managed", "mentionable", "hoist", "color", "position", "permissions").
		From("roles").
		Where(sq.Eq{"role_nick": ticker}).
		Where(sq.Eq{"sig": sig})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return payloads.Role{}, err
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	err = query.Scan(
		&role.ID,
		&role.Name,
		&role.Managed,
		&role.Mentionable,
		&role.Hoist,
		&role.Color,
		&role.Position,
		&role.Permissions,
	)
	if err != nil {
		sp.Error("error fetching fole from db", zap.Error(err))
		return payloads.Role{}, fmt.Errorf("error fetching %s from db: %s", roleType[sig], err)
	}

	sp.With(zap.Any("role", role))

	sp.Info("Got role from db")
	return role, nil
}

func GetDiscordRole(ctx context.Context, name string, deps common.Dependencies) (*discordgo.Role, error) {
	_, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(zap.String("name", name))

	roles, err := deps.Session.GuildRoles(deps.GuildID)
	if err != nil {
		sp.Error("error getting roles from discord", zap.Error(err))
		return nil, err
	}

	for _, r := range roles {
		if r.Name == name {
			// something is different, let's push changes to discord
			sp.Info("found role", zap.String("discord_role", r.Name))
			return r, nil
		}
	}

	sp.Warn("no such role")
	return nil, fmt.Errorf("no such role: %s", name)
}

func ListFilters(ctx context.Context, sig bool, ticker string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("role_type", roleType[sig]),
		zap.String("ticker", ticker),
	)

	var (
		buffer   bytes.Buffer
		filter   string
		results  bool
		messages []*discordgo.MessageSend
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := deps.DB.Select("filters.name").
		From("filters").
		Join("role_filters ON role_filters.filter = filters.id").
		Join("roles ON roles.id = role_filters.role").
		Where(sq.Eq{"roles.role_nick": ticker}).
		Where(sq.Eq{"roles.sig": sig})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return common.SendFatal(err.Error())
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	rows, err := query.QueryContext(ctx)
	if err != nil {
		sp.Error("error fetching filters", zap.Error(err))
		return common.SendFatal(fmt.Sprintf("error fetching filters: %s", err))
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			sp.Error("error closing database", zap.Error(err))
		}
	}()

	for rows.Next() {
		if !results {
			results = true
		}
		err = rows.Scan(&filter)
		if err != nil {
			sp.Error("error scanning filters", zap.Error(err))
			return common.SendFatal(fmt.Sprintf("error scanning row filters: %s", err))
		}

		buffer.WriteString(fmt.Sprintf("%s\n", filter))
	}

	if results {
		embed := common.NewEmbed()
		embed.SetTitle(fmt.Sprintf("Filters for %s", ticker))
		embed.SetDescription(buffer.String())
		return append(messages, &discordgo.MessageSend{Embed: embed.GetMessageEmbed()})
	} else {
		sp.Warn("no such role")
		return common.SendError(fmt.Sprintf("No such %s: %s", roleType[sig], ticker))
	}
}

func AuthedAddFilter(ctx context.Context, sig bool, filter, ticker, author string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("role_type", roleType[sig]),
		zap.String("filter", filter),
		zap.String("ticker", ticker),
		zap.String("author", author),
	)

	if !perms.CanPerform(ctx, author, adminType[sig], deps) {
		sp.Warn("user doesn't have permission to this command")
		return common.SendError("User doesn't have permission to this command")
	}

	sp.Debug("adding filter")
	return AddFilter(ctx, sig, filter, ticker, deps)
}

func AddFilter(ctx context.Context, sig bool, filter, ticker string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()
	sp.With(
		zap.String("filter", filter),
		zap.String("ticker", ticker),
		zap.String("role_type", roleType[sig]),
	)

	sp.With(
		zap.String("role_type", roleType[sig]),
		zap.String("filter", filter),
		zap.String("ticker", ticker),
	)

	var (
		err      error
		filterID int
		roleID   int
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := deps.DB.Select("id").
		From("filters").
		Where(sq.Eq{"name": filter})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return common.SendFatal(err.Error())
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	err = query.Scan(&filterID)
	if err != nil {
		if err == sql.ErrNoRows {
			sp.Error("no such filter")
			return common.SendError(fmt.Sprintf("No such filter: %s", filter))
		}
		sp.Error("error fetching filter id", zap.Error(err))
		return common.SendFatal(fmt.Sprintf("error fetching filter id: %s", err))
	}

	sp.With(zap.Int("filter_id", filterID))

	query = deps.DB.Select("id").
		From("roles").
		Where(sq.Eq{"role_nick": ticker}).
		Where(sq.Eq{"sig": sig})

	sqlStr, args, err = query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return common.SendFatal(err.Error())
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	err = query.Scan(&roleID)
	if err != nil {
		if err == sql.ErrNoRows {
			sp.Error("no such filter")
			return common.SendError(fmt.Sprintf("No such %s: %s", roleType[sig], filter))
		}
		sp.Error("error fetching role", zap.Error(err))
		return common.SendFatal(fmt.Sprintf("error fetching %s id: %s", roleType[sig], err))
	}

	sp.With(zap.Int("role_id", roleID))

	insert := deps.DB.Insert("role_filters").
		Columns("role", "filter").
		Values(roleID, filterID)

	sqlStr, args, err = insert.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return common.SendFatal(err.Error())
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	rows, err := insert.QueryContext(ctx)
	if err != nil {
		sp.Error("error inserting role_filter", zap.Error(err))
		return common.SendFatal(fmt.Sprintf("error inserting role_filter: %s", err))
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			sp.Error("error closing database", zap.Error(err))
		}
	}()

	sp.Info("added filter")
	return common.SendSuccess(fmt.Sprintf("Added filter %s to role %s", filter, ticker))
}

func AuthedRemoveFilter(ctx context.Context, sig bool, filter, ticker, author string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("role_type", roleType[sig]),
		zap.String("filter", filter),
		zap.String("ticker", ticker),
		zap.String("author", author),
	)

	if !perms.CanPerform(ctx, author, adminType[sig], deps) {
		sp.Warn("user doesn't have permission to this command")
		return common.SendError("User doesn't have permission to this command")
	}

	sp.Debug("removing filter")
	return RemoveFilter(ctx, sig, filter, ticker, deps)
}

func RemoveFilter(ctx context.Context, sig bool, filter, ticker string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("role_type", roleType[sig]),
		zap.String("filter", filter),
		zap.String("ticker", ticker),
	)

	var (
		err      error
		filterID int
		roleID   int
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := deps.DB.Select("id").
		From("filters").
		Where(sq.Eq{"name": filter})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return common.SendFatal(err.Error())
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	err = query.Scan(&filterID)
	if err != nil {
		if err == sql.ErrNoRows {
			sp.Error("no such filter")
			return common.SendError(fmt.Sprintf("No such filter: %s", filter))
		}
		sp.Error("error fetching filter", zap.Error(err))
		return common.SendFatal(fmt.Sprintf("error fetching filter id: %s", err))
	}

	sp.With(zap.Int("filter_id", filterID))

	query = deps.DB.Select("id").
		From("roles").
		Where(sq.Eq{"role_nick": ticker}).
		Where(sq.Eq{"sig": sig})

	sqlStr, args, err = query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return common.SendFatal(err.Error())
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	err = query.Scan(&roleID)
	if err != nil {
		if err == sql.ErrNoRows {
			sp.Error("no such filter")
			return common.SendError(fmt.Sprintf("No such %s: %s", roleType[sig], filter))
		}
		sp.Error("error fetching filter", zap.Error(err))
		return common.SendFatal(fmt.Sprintf("error fetching %s id: %s", roleType[sig], err))
	}

	sp.With(zap.Int("role_id", roleID))

	delQuery := deps.DB.Delete("role_filters").
		Where(sq.Eq{"role": roleID}).
		Where(sq.Eq{"filter": filterID})

	sqlStr, args, err = delQuery.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return common.SendFatal(err.Error())
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	rows, err := delQuery.QueryContext(ctx)
	if err != nil {
		sp.Error("error deleting role_filter", zap.Error(err))
		return common.SendFatal(fmt.Sprintf("error deleting role_filter: %s", err))
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			sp.Error("error closing database", zap.Error(err))
		}
	}()

	sp.Info("removed filter")
	return common.SendSuccess(fmt.Sprintf("Removed filter %s from role %s", filter, ticker))
}
