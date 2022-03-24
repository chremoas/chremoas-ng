package filters

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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

func List(ctx context.Context, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	var (
		buffer   bytes.Buffer
		filter   payloads.Filter
		messages []*discordgo.MessageSend
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	rows, err := deps.DB.Select("name", "description").
		From("filters").
		QueryContext(ctx)
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
			newErr := fmt.Errorf("error scanning filter row: %s", err)
			sp.Error("error scanning filter row", zap.Error(err))
			return common.SendFatal(newErr.Error())
		}

		buffer.WriteString(fmt.Sprintf("%s: %s\n", filter.Name, filter.Description))
	}

	if buffer.Len() == 0 {
		return common.SendError("No filters")
	}

	if buffer.Len() > 2000 {
		return common.SendError("too many filters (exceeds Discord 2k character limit)")
	}

	embed := common.NewEmbed()
	embed.SetTitle("Filters")
	embed.SetDescription(buffer.String())
	return append(messages, &discordgo.MessageSend{Embed: embed.GetMessageEmbed()})
}

func AuthedAdd(ctx context.Context, name, description string, author string, deps common.Dependencies) ([]*discordgo.MessageSend, int) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	if !perms.CanPerform(ctx, author, "role_admins", deps) {
		return common.SendError("User doesn't have permission to this command"), -1
	}

	return Add(ctx, name, description, deps)
}

func Add(ctx context.Context, name, description string, deps common.Dependencies) ([]*discordgo.MessageSend, int) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

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
		sp.Error("error inserting filter", zap.Error(err))
		return common.SendFatal(newErr.Error()), -1
	}

	return common.SendSuccess(fmt.Sprintf("Created filter `%s`", name)), id
}

func AuthedDelete(ctx context.Context, name string, author string, deps common.Dependencies) ([]*discordgo.MessageSend, int) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	if !perms.CanPerform(ctx, author, "role_admins", deps) {
		return common.SendError("User doesn't have permission to this command"), -1
	}

	return Delete(ctx, name, deps)
}

func Delete(ctx context.Context, name string, deps common.Dependencies) ([]*discordgo.MessageSend, int) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	var id int

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	rows, err := deps.DB.Delete("filters").
		Where(sq.Eq{"name": name}).
		Suffix("RETURNING \"id\"").
		QueryContext(ctx)
	if err != nil {
		newErr := fmt.Errorf("error deleting filter: %s", err)
		sp.Error("error deleting filter", zap.Error(err))
		return common.SendFatal(newErr.Error()), -1
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
			newErr := fmt.Errorf("error scanning filters id: %s", err)
			sp.Error("error scanning filters", zap.Error(err))
			return common.SendFatal(newErr.Error()), -1
		}
	}

	return common.SendSuccess(fmt.Sprintf("Deleted filter `%s`", name)), id
}

func ListMembers(ctx context.Context, name string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	var (
		userID   int
		buffer   bytes.Buffer
		messages []*discordgo.MessageSend
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	rows, err := deps.DB.Select("user_id").
		From("filters").
		Join("filter_membership ON filters.id = filter_membership.filter").
		Where(sq.Eq{"filters.name": name}).
		QueryContext(ctx)
	if err != nil {
		newErr := fmt.Errorf("error getting filter membership list: %s", err)
		sp.Error("error getting filter membership list", zap.Error(err))
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
			newErr := fmt.Errorf("error scanning filter_membership userID: %s", err)
			sp.Error("error scanning filter_membership userID", zap.Error(err))
			return common.SendFatal(newErr.Error())
		}
		buffer.WriteString(fmt.Sprintf("%s\n", common.GetUsername(userID, deps.Session)))
	}

	if buffer.Len() == 0 {
		return common.SendError(fmt.Sprintf("Filter has no members: %s", name))
	}

	if buffer.Len() > 2000 {
		return common.SendError("too many filter members (exceeds Discord 2k character limit)")
	}

	embed := common.NewEmbed()
	embed.SetTitle(fmt.Sprintf("Filter membership (%s)", name))
	embed.SetDescription(buffer.String())
	return append(messages, &discordgo.MessageSend{Embed: embed.GetMessageEmbed()})
}

func AuthedAddMember(ctx context.Context, userID, filter, author string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	if !perms.CanPerform(ctx, author, "role_admins", deps) {
		return common.SendError("User doesn't have permission to this command")
	}

	return AddMember(ctx, userID, filter, deps)
}

func AddMember(ctx context.Context, userID, filter string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	var filterID int

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	_, err := strconv.Atoi(userID)
	if err != nil {
		if !common.IsDiscordUser(userID) {
			return common.SendError("second argument must be a discord user")
		}
		userID = common.ExtractUserId(userID)
	}

	before, err := common.GetMembership(ctx, userID, deps)
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
		sp.Error("error scanning filterID", zap.Error(err))
		return common.SendFatal(newErr.Error())
	}

	sp.Info("Got member info", zap.String("userID", userID), zap.Int("filterID", filterID))

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
		sp.Error("error inserting filter", zap.Error(err))
		return common.SendFatal(newErr.Error())
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			sp.Error("error closing database", zap.Error(err))
		}
	}()

	after, err := common.GetMembership(ctx, userID, deps)
	if err != nil {
		return common.SendFatal(err.Error())
	}

	addSet := after.Difference(before)

	if addSet.Len() == 0 {
		/* TODO: this error is not always correct. If someone joins a sig they aren't a member of all filters of
		 * it appears like they aren't in it.
		 */
		return common.SendError(fmt.Sprintf("<@%s> already a member of: `%s` (maybe)", userID, filter))
	}

	for _, role := range addSet.ToSlice() {
		QueueUpdate(ctx, payloads.Upsert, userID, role, deps)
	}

	return common.SendSuccess(fmt.Sprintf("Added <@%s> to `%s`", userID, filter))
}

func AuthedRemoveMember(ctx context.Context, userID, filter, author string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	if !perms.CanPerform(ctx, author, "role_admins", deps) {
		return common.SendError("User doesn't have permission to this command")
	}

	return RemoveMember(ctx, userID, filter, deps)
}

func RemoveMember(ctx context.Context, userID, filter string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	var filterID int

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	before, err := common.GetMembership(ctx, userID, deps)
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
		sp.Error("error scanning filterID", zap.Error(err))
		return common.SendFatal(newErr.Error())
	}

	rows, err := deps.DB.Delete("filter_membership").
		Where(sq.Eq{"filter": filterID}).
		Where(sq.Eq{"user_id": userID}).
		Suffix("RETURNING \"id\"").
		QueryContext(ctx)
	if err != nil {
		newErr := fmt.Errorf("error deleting filter: %s", err)
		sp.Error("error deleting filter", zap.Error(err))
		return common.SendFatal(newErr.Error())
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			sp.Error("error closing database", zap.Error(err))
		}
	}()

	after, err := common.GetMembership(ctx, userID, deps)
	if err != nil {
		return common.SendFatal(err.Error())
	}

	removeSet := before.Difference(after)

	if removeSet.Len() == 0 {
		return common.SendError(fmt.Sprintf("<@%s> not a member of `%s`", userID, filter))
	}

	for _, role := range removeSet.ToSlice() {
		QueueUpdate(ctx, payloads.Delete, userID, role, deps)
	}

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

	b, err := json.Marshal(payload)
	if err != nil {
		sp.Error("error marshalling queue message", zap.Error(err))
	}

	if roleID == "0" {
		// no point submitting a message as it'll be ignored anyway
		return
	}

	sp.Debug("Submitting member queue message", zap.Any("payload", payload))
	err = deps.MembersProducer.Publish(ctx, b)
	if err != nil {
		sp.Error("error publishing message", zap.Error(err))
	}
}
