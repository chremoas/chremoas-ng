package filters

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	sq "github.com/Masterminds/squirrel"
	sl "github.com/bhechinger/spiffylogger"
	"github.com/bwmarrin/discordgo"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	"github.com/chremoas/chremoas-ng/internal/perms"
	"github.com/lib/pq"
	"go.uber.org/zap"
)

func List(ctx context.Context, channelID string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	var (
		filter     payloads.Filter
		filterList []string
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := deps.DB.Select("name", "description").
		From("filters")

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return common.SendFatal(err.Error())
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	rows, err := query.QueryContext(ctx)
	if err != nil {
		sp.Error("error getting filter", zap.Error(err))
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
			sp.Error("error scanning filter row", zap.Error(err))
			return common.SendFatal(err.Error())
		}

		filterList = append(filterList, fmt.Sprintf("%s: %s", filter.Name, filter.Description))
	}
	sort.Strings(filterList)

	if len(filterList) == 0 {
		return common.SendError("No filters")
	}

	err = common.SendChunkedMessage(ctx, channelID, "Filter List", filterList, deps)
	if err != nil {
		sp.Error("Error sending message")
		return common.SendError("Error sending message")
	}

	return nil
}

func AuthedAdd(ctx context.Context, name, description, author string, deps common.Dependencies) ([]*discordgo.MessageSend, int) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("name", name),
		zap.String("description", description),
		zap.String("author", author),
	)

	if !perms.CanPerform(ctx, author, "role_admins", deps) {
		sp.Warn("user doesn't have permission to this command")
		return common.SendError("User doesn't have permission to this command"), -1
	}

	sp.Debug("adding filter")
	return Add(ctx, name, description, deps)
}

func Add(ctx context.Context, name, description string, deps common.Dependencies) ([]*discordgo.MessageSend, int) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("name", name),
		zap.String("description", description),
	)

	var id int
	insert := deps.DB.Insert("filters").
		Columns("name", "description").
		Values(name, description).
		Suffix("RETURNING \"id\"")

	sqlStr, args, err := insert.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return common.SendFatal(err.Error()), -1
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	err = insert.Scan(&id)
	if err != nil {
		// I don't love this, but I can't find a better way right now
		if err.(*pq.Error).Code == "23505" {
			return common.SendError(fmt.Sprintf("filter `%s` already exists", name)), -1
		}
		sp.Error("error inserting filter", zap.Error(err))
		return common.SendFatal(err.Error()), -1
	}
	sp.With(zap.Int("id", id))

	sp.Info("created filter")
	return common.SendSuccess(fmt.Sprintf("Created filter `%s`", name)), id
}

func AuthedDelete(ctx context.Context, name, author string, deps common.Dependencies) ([]*discordgo.MessageSend, int) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("name", name),
		zap.String("author", author),
	)

	if !perms.CanPerform(ctx, author, "role_admins", deps) {
		sp.Warn("user doesn't have permission to this command")
		return common.SendError("User doesn't have permission to this command"), -1
	}

	sp.Debug("deleting filter")
	return Delete(ctx, name, deps)
}

func Delete(ctx context.Context, name string, deps common.Dependencies) ([]*discordgo.MessageSend, int) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(zap.String("name", name))

	var id int

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := deps.DB.Delete("filters").
		Where(sq.Eq{"name": name}).
		Suffix("RETURNING \"id\"")

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return common.SendFatal(err.Error()), -1
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	rows, err := query.QueryContext(ctx)
	if err != nil {
		sp.Error("error deleting filter", zap.Error(err))
		return common.SendFatal(err.Error()), -1
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			sp.Error("error closing database", zap.Error(err))
		}
	}()

	for rows.Next() {
		err = rows.Scan(&id)
		if err != nil {
			sp.Error("error scanning filters", zap.Error(err))
			return common.SendFatal(err.Error()), -1
		}
		sp.Info("deleted filter", zap.Int("id", id))
	}

	return common.SendSuccess(fmt.Sprintf("Deleted filter `%s`", name)), id
}

func ListMembers(ctx context.Context, filter string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(zap.String("filter", filter))

	var (
		userID   int
		buffer   bytes.Buffer
		messages []*discordgo.MessageSend
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := deps.DB.Select("user_id").
		From("filters").
		Join("filter_membership ON filters.id = filter_membership.filter").
		Where(sq.Eq{"filters.name": filter})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return common.SendFatal(err.Error())
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	rows, err := query.QueryContext(ctx)
	if err != nil {
		sp.Error("error getting filter membership list", zap.Error(err))
		return common.SendFatal(err.Error())
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
			sp.Error("error scanning filter_membership userID", zap.Error(err))
			return common.SendFatal(err.Error())
		}
		buffer.WriteString(fmt.Sprintf("%s\n", common.GetUsername(userID, deps.Session)))
	}

	bufLen := buffer.Len()

	sp.With(zap.Int("char_total", bufLen))

	if bufLen == 0 {
		return common.SendError(fmt.Sprintf("Filter has no members: %s", filter))
	}

	if bufLen > 2000 {
		sp.Error("too many characters for response")
		return common.SendError("too many filter members (exceeds Discord 2k character limit)")
	}

	embed := common.NewEmbed()
	embed.SetTitle(fmt.Sprintf("Filter membership (%s)", filter))
	embed.SetDescription(buffer.String())
	return append(messages, &discordgo.MessageSend{Embed: embed.GetMessageEmbed()})
}

func AuthedAddMember(ctx context.Context, userID, filter, author string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("user_id", userID),
		zap.String("filter", filter),
		zap.String("author", author),
	)

	if !perms.CanPerform(ctx, author, "role_admins", deps) {
		sp.Warn("user doesn't have permission to this command")
		return common.SendError("User doesn't have permission to this command")
	}

	sp.Debug("adding filter member")
	return AddMember(ctx, userID, filter, deps)
}

func AddMember(ctx context.Context, userID, filter string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("user_id", userID),
		zap.String("filter", filter),
	)

	var filterID int

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	_, err := strconv.Atoi(userID)
	if err != nil {
		if !common.IsDiscordUser(userID) {
			sp.Warn("second argument must be a discord user")
			return common.SendError("second argument must be a discord user")
		}
		userID = common.ExtractUserId(userID)
	}

	before, err := common.GetMembership(ctx, userID, deps)
	if err != nil {
		sp.Error("error getting membership", zap.Error(err))
		return common.SendFatal(err.Error())
	}

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
		sp.Error("error scanning filterID", zap.Error(err))
		return common.SendFatal(err.Error())
	}

	sp.With(zap.Int("filter_id", filterID))

	sp.Info("Got member info")

	insert := deps.DB.Insert("filter_membership").
		Columns("filter", "user_id").
		Values(filterID, userID)

	sqlStr, args, err = insert.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return common.SendFatal(err.Error())
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	rows, err := insert.QueryContext(ctx)
	if err != nil {
		// I don't love this, but I can't find a better way right now
		if err.(*pq.Error).Code == "23505" {
			sp.Warn("already a member", zap.Bool("maybe", false))
			return common.SendError(fmt.Sprintf("<@%s> already a member of `%s`", userID, filter))
		}
		sp.Error("error inserting filter", zap.Error(err))
		return common.SendFatal(err.Error())
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			sp.Error("error closing database", zap.Error(err))
		}
	}()

	after, err := common.GetMembership(ctx, userID, deps)
	if err != nil {
		sp.Error("error getting membership")
		return common.SendFatal(err.Error())
	}

	addSet := after.Difference(before)

	if addSet.Len() == 0 {
		/* TODO: this error is not always correct. If someone joins a sig they aren't a member of all filters of
		 * it appears like they aren't in it.
		 */
		sp.Error("already a member", zap.Bool("maybe", true))
		return common.SendError(fmt.Sprintf("<@%s> already a member of: `%s` (maybe)", userID, filter))
	}

	for _, role := range addSet.ToSlice() {
		QueueUpdate(ctx, payloads.Upsert, userID, role, deps)
	}

	sp.Info("added user to filter")
	return common.SendSuccess(fmt.Sprintf("Added <@%s> to `%s`", userID, filter))
}

func AuthedRemoveMember(ctx context.Context, userID, filter, author string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("user_id", userID),
		zap.String("filter", filter),
		zap.String("author", author),
	)

	if !perms.CanPerform(ctx, author, "role_admins", deps) {
		sp.Warn("user doesn't have permission to this command")
		return common.SendError("User doesn't have permission to this command")
	}

	sp.Debug("removing filter member")
	return RemoveMember(ctx, userID, filter, deps)
}

func RemoveMember(ctx context.Context, userID, filter string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("user_id", userID),
		zap.String("filter", filter),
	)

	var filterID int

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	before, err := common.GetMembership(ctx, userID, deps)
	if err != nil {
		sp.Error("error getting membership")
		return common.SendFatal(err.Error())
	}

	_, err = strconv.Atoi(userID)
	if err != nil {
		if !common.IsDiscordUser(userID) {
			sp.Warn("second argument must be a discord user")
			return common.SendError("second argument must be a discord user")
		}
		userID = common.ExtractUserId(userID)
	}

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
		sp.Error("error scanning filterID", zap.Error(err))
		return common.SendFatal(err.Error())
	}

	sp.With(zap.Int("filter_id", filterID))

	delQuery := deps.DB.Delete("filter_membership").
		Where(sq.Eq{"filter": filterID}).
		Where(sq.Eq{"user_id": userID}).
		Suffix("RETURNING \"id\"")

	sqlStr, args, err = delQuery.ToSql()
	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
		return common.SendFatal(err.Error())
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}

	rows, err := delQuery.QueryContext(ctx)
	if err != nil {
		sp.Error("error deleting filter", zap.Error(err))
		return common.SendFatal(err.Error())
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			sp.Error("error closing database", zap.Error(err))
		}
	}()

	after, err := common.GetMembership(ctx, userID, deps)
	if err != nil {
		sp.Error("error getting membership")
		return common.SendFatal(err.Error())
	}

	removeSet := before.Difference(after)

	if removeSet.Len() == 0 {
		return common.SendError(fmt.Sprintf("<@%s> not a member of `%s`", userID, filter))
	}

	for _, role := range removeSet.ToSlice() {
		QueueUpdate(ctx, payloads.Delete, userID, role, deps)
	}

	sp.Info("removed user from filter")
	return common.SendSuccess(fmt.Sprintf("Removed <@%s> from `%s`", userID, filter))
}

func QueueUpdate(ctx context.Context, action payloads.Action, memberID, roleID string, deps common.Dependencies) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	payload := payloads.MemberPayload{
		Action:   action,
		GuildID:  deps.GuildID,
		MemberID: memberID,
		RoleID:   roleID,
	}

	sp.With(
		zap.Any("action", action),
		zap.String("member_id", memberID),
		zap.String("role_id", roleID),
		zap.Any("payload", payload),
	)

	b, err := json.Marshal(payload)
	if err != nil {
		sp.Error("error marshalling queue message", zap.Error(err))
		return
	}

	if roleID == "0" {
		// no point submitting a message as it'll be ignored anyway
		return
	}

	sp.Debug("Submitting member queue message")
	err = deps.MembersProducer.Publish(ctx, b)
	if err != nil {
		sp.Error("error publishing message", zap.Error(err))
	}
}
