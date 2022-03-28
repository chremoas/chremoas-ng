package perms

import (
	"bytes"
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	sl "github.com/bhechinger/spiffylogger"
	"github.com/bwmarrin/discordgo"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	"github.com/lib/pq"
	"go.uber.org/zap"
)

func List(ctx context.Context, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	var (
		count    int
		buffer   bytes.Buffer
		filter   payloads.Filter
		messages []*discordgo.MessageSend
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := deps.DB.Select("name", "description").
		From("permissions")

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	rows, err := query.QueryContext(ctx)
	if err != nil {
		sp.Error("error getting permissions", zap.Error(err))
		return common.SendFatal(err.Error())
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			sp.Error("error closing database", zap.Error(err))
		}
	}()

	for rows.Next() {
		err = rows.Scan(&filter.Name, &filter.Description)
		if err != nil {
			newErr := fmt.Errorf("error scanning permissions row: %s", err)
			sp.Error("error scanning permissions", zap.Error(err))
			return common.SendFatal(newErr.Error())
		}

		buffer.WriteString(fmt.Sprintf("%s: %s\n", filter.Name, filter.Description))
		count += 1
	}

	if count == 0 {
		return common.SendError("No permissions")
	}

	embed := common.NewEmbed()
	embed.SetTitle("Permissions")
	embed.SetDescription(buffer.String())

	return append(messages, &discordgo.MessageSend{Embed: embed.GetMessageEmbed()})
}

func Add(ctx context.Context, name, description, author string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if name == "server_admins" {
		return common.SendError("User doesn't have permission to this command")
	}

	if !CanPerform(ctx, author, "server_admins", deps) {
		return common.SendError("User doesn't have permission to this command")
	}

	insert := deps.DB.Insert("permissions").
		Columns("name", "description").
		Values(name, description)

	sqlStr, args, err := insert.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	rows, err := insert.QueryContext(ctx)
	if err != nil {
		// I don't love this but I can't find a better way right now
		if err.(*pq.Error).Code == "23505" {
			return common.SendError(fmt.Sprintf("permission `%s` already exists", name))
		}
		newErr := fmt.Errorf("error inserting permission: %s", err)
		sp.Error("error inserting permissions", zap.Error(err))
		return common.SendFatal(newErr.Error())
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			sp.Error("error closing database", zap.Error(err))
		}
	}()

	return common.SendSuccess(fmt.Sprintf("Created permission `%s`", name))
}

func Delete(ctx context.Context, name, author string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if name == "server_admins" {
		return common.SendError("User doesn't have permission to this command")
	}

	if !CanPerform(ctx, author, "server_admins", deps) {
		return common.SendError("User doesn't have permission to this command")
	}

	query := deps.DB.Delete("permissions").
		Where(sq.Eq{"name": name})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	rows, err := query.QueryContext(ctx)
	if err != nil {
		newErr := fmt.Errorf("error deleting permission: %s", err)
		sp.Error("error deleting permissions", zap.Error(err))
		return common.SendFatal(newErr.Error())
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			sp.Error("error closing database", zap.Error(err))
		}
	}()

	return common.SendSuccess(fmt.Sprintf("Deleted permission `%s`", name))
}

func ListMembers(ctx context.Context, name string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	var (
		count, userID int
		buffer        bytes.Buffer
		messages      []*discordgo.MessageSend
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := deps.DB.Select("user_id").
		From("permission_membership").
		Join("permissions ON permission_membership.permission = permissions.id").
		Where(sq.Eq{"permissions.name": name})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	rows, err := query.QueryContext(ctx)
	if err != nil {
		newErr := fmt.Errorf("error getting permission membership list: %s", err)
		sp.Error("error getting permissions membership list", zap.Error(err))
		return common.SendFatal(newErr.Error())
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			sp.Error("error closing database", zap.Error(err))
		}
	}()

	for rows.Next() {
		err = rows.Scan(&userID)
		if err != nil {
			newErr := fmt.Errorf("error scanning permission_membership userID: %s", err)
			sp.Error("error scanning permission_membership userID", zap.Error(err))
			return common.SendFatal(newErr.Error())
		}
		buffer.WriteString(fmt.Sprintf("%s\n", common.GetUsername(userID, deps.Session)))
		count += 1
	}

	if count == 0 {
		return common.SendError(fmt.Sprintf("Permission has no members: %s", name))
	}

	embed := common.NewEmbed()
	embed.SetTitle(fmt.Sprintf("%s members", name))
	embed.SetDescription(buffer.String())

	return append(messages, &discordgo.MessageSend{Embed: embed.GetMessageEmbed()})
}

func AddMember(ctx context.Context, user, permission, author string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	var permissionID int

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if permission == "server_admins" {
		return common.SendError("User doesn't have permission to this command")
	}

	if !CanPerform(ctx, author, "server_admins", deps) {
		return common.SendError("User doesn't have permission to this command")
	}

	if !common.IsDiscordUser(user) {
		return common.SendError("second argument must be a discord user")
	}

	userID := common.ExtractUserId(user)

	query := deps.DB.Select("id").
		From("permissions").
		Where(sq.Eq{"name": permission})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	err = query.Scan(&permissionID)
	if err != nil {
		newErr := fmt.Errorf("error scanning permissionID: %s", err)
		sp.Error("error scanning permissionID", zap.Error(err))
		return common.SendFatal(newErr.Error())
	}

	insert := deps.DB.Insert("permission_membership").
		Columns("permission", "user_id").
		Values(permissionID, userID)

	sqlStr, args, err = insert.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	rows, err := insert.QueryContext(ctx)
	if err != nil {
		// I don't love this but I can't find a better way right now
		if err.(*pq.Error).Code == "23505" {
			return common.SendError(fmt.Sprintf("<@%s> already a member of `%s`", userID, permission))
		}
		newErr := fmt.Errorf("error inserting permission: %s", err)
		sp.Error("error inserting permission", zap.Error(err))
		return common.SendFatal(newErr.Error())
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			sp.Error("error closing database", zap.Error(err))
		}
	}()

	return common.SendSuccess(fmt.Sprintf("Added <@%s> to `%s`", userID, permission))
}

func RemoveMember(ctx context.Context, user, permission, author string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	var permissionID int

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if permission == "server_admins" {
		return common.SendError("User doesn't have permission to this command")
	}

	if !CanPerform(ctx, author, "server_admins", deps) {
		return common.SendError("User doesn't have permission to this command")
	}

	if !common.IsDiscordUser(user) {
		return common.SendError("second argument must be a discord user")
	}

	userID := common.ExtractUserId(user)

	query := deps.DB.Select("id").
		From("permissions").
		Where(sq.Eq{"name": permission})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	err = query.Scan(&permissionID)
	if err != nil {
		newErr := fmt.Errorf("error scanning permisionID: %s", err)
		sp.Error("error scanning permissionID", zap.Error(err))
		return common.SendFatal(newErr.Error())
	}

	delQuery := deps.DB.Delete("permission_membership").
		Where(sq.Eq{"permission": permissionID}).
		Where(sq.Eq{"user_id": userID})

	sqlStr, args, err = delQuery.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	rows, err := delQuery.QueryContext(ctx)
	if err != nil {
		newErr := fmt.Errorf("error deleting permission: %s", err)
		sp.Error("error deleting permission", zap.Error(err))
		return common.SendFatal(newErr.Error())
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			sp.Error("error closing database", zap.Error(err))
		}
	}()

	return common.SendSuccess(fmt.Sprintf("Removed <@%s> from `%s`", userID, permission))
}

func UserPerms(ctx context.Context, user string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	var (
		buffer     bytes.Buffer
		permission string
		messages   []*discordgo.MessageSend
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if !common.IsDiscordUser(user) {
		return common.SendError("second argument must be a discord user")
	}

	userID := common.ExtractUserId(user)

	query := deps.DB.Select("name").
		From("permissions").
		Join("permission_membership ON permission_membership.permission = permissions.id").
		Where(sq.Eq{"permission_membership.user_id": userID})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	rows, err := query.QueryContext(ctx)
	if err != nil {
		newErr := fmt.Errorf("error getting user perms: %s", err)
		sp.Error("error getting user perms", zap.Error(err))
		return common.SendFatal(newErr.Error())
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			sp.Error("error closing database", zap.Error(err))
		}
	}()

	for rows.Next() {
		err = rows.Scan(&permission)
		if err != nil {
			sp.Error("Error scanning permission id", zap.Error(err))
			return common.SendFatal(err.Error())
		}
		buffer.WriteString(fmt.Sprintf("\t%s\n", permission))
	}

	embed := common.NewEmbed()
	embed.SetTitle(fmt.Sprintf("%s's Permissions", common.GetUsername(userID, deps.Session)))
	embed.SetDescription(buffer.String())

	return append(messages, &discordgo.MessageSend{Embed: embed.GetMessageEmbed()})
}

func CanPerform(ctx context.Context, authorID, permission string, deps common.Dependencies) bool {
	_, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	var (
		count        int
		permissionID int
	)

	// This is super jank and I don't like it, need to come up with a better way.
	if authorID == "auth-web" {
		return true
	}

	query := deps.DB.Select("id").
		From("permissions").
		Where(sq.Eq{"name": permission})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	err = query.Scan(&permissionID)
	if err != nil {
		sp.Error("error scanning permissionID", zap.Error(err))
		return false
	}
	query = deps.DB.Select("COUNT(*)").
		From("permission_membership").
		Where(sq.Eq{"user_id": authorID}).
		Where(sq.Eq{"permission": permissionID})

	sqlStr, args, err = query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	err = query.Scan(&count)
	if err != nil {
		sp.Error("error scanning permission count", zap.Error(err))
		return false
	}

	if count == 0 {
		return false
	}

	return true
}
