package filters

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"

	sl "github.com/bhechinger/spiffylogger"
	"github.com/bwmarrin/discordgo"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	"github.com/chremoas/chremoas-ng/internal/perms"
	"github.com/chremoas/chremoas-ng/internal/storage"
	"go.uber.org/zap"
)

func List(ctx context.Context, channelID string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	filters, err := deps.Storage.GetFilters(ctx)
	if err != nil {
		sp.Error("Error getting filters")
		return common.SendFatalf(nil, "Error getting filters: %s", err)
	}

	var filterList []string

	for f := range filters {
		filterList = append(filterList, fmt.Sprintf("%s: %s", filters[f].Name, filters[f].Description))
	}
	sort.Strings(filterList)

	if len(filterList) == 0 {
		return common.SendError(nil, "No filters")
	}

	err = common.SendChunkedMessage(ctx, channelID, "Filter List", filterList, deps)
	if err != nil {
		sp.Error("Error sending chunked message")
		return common.SendErrorf(nil, "Error sending chunked message: %s", err)
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

	if err := perms.CanPerform(ctx, author, "role_admins", deps); err != nil {
		sp.Warn("user doesn't have permission to this command", zap.Error(err))
		return common.SendError(&author, "User doesn't have permission to this command"), -1
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

	id, err := deps.Storage.InsertFilter(ctx, name, description)
	if err != nil {
		if errors.Is(err, storage.ErrFilterExists) {
			return common.SendErrorf(nil, "Filter already exists: %s", name), -1
		}

		return common.SendErrorf(nil, "Error inserting filter:%s", err), -1
	}

	sp.With(zap.Int("id", id))

	sp.Info("created filter")
	return common.SendSuccessf(nil, "Created filter `%s`", name), id
}

func AuthedDelete(ctx context.Context, name, author string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("name", name),
		zap.String("author", author),
	)

	if err := perms.CanPerform(ctx, author, "role_admins", deps); err != nil {
		sp.Warn("user doesn't have permission to this command", zap.Error(err))
		return common.SendError(&author, "User doesn't have permission to this command")
	}

	sp.Debug("deleting filter")
	return Delete(ctx, name, deps)
}

func Delete(ctx context.Context, name string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(zap.String("name", name))

	var id int
	err := deps.Storage.DeleteFilter(ctx, name)
	if err != nil {
		if errors.Is(err, storage.ErrNotFilterMember) {
			return common.SendErrorf(nil, "User not a member of filter: %s", name)
		}

		if errors.Is(err, storage.ErrNoFilter) {
			return common.SendErrorf(nil, "No such filter: %s", name)
		}

		sp.Error("Error deleting filter")
		return common.SendErrorf(nil, "Error deleting filter: %s", name)
	}

	sp.Info("deleted filter", zap.Int("id", id))

	return common.SendSuccessf(nil, "Deleted filter `%s`", name)
}

func ListMembers(ctx context.Context, filter, channelID string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(zap.String("filter", filter))

	userIDs, err := deps.Storage.ListFilterMembers(ctx, filter)
	if err != nil {
		sp.Error("Error listing filter members")
		return common.SendErrorf(nil, "Error getting filter member list for filter: %s", filter)
	}

	var memberList []string
	for u := range userIDs {
		memberList = append(memberList, fmt.Sprintf("%s", common.GetUsername(userIDs[u], deps.Session)))
	}
	sort.Strings(memberList)

	if len(memberList) == 0 {
		return common.SendErrorf(nil, "No members in filter: %s", filter)
	}

	err = common.SendChunkedMessage(ctx, channelID, "Filter List", memberList, deps)
	if err != nil {
		sp.Error("Error sending chunked message")
		return common.SendErrorf(nil, "Error sending chunked message: %s", err)
	}

	return nil
}

func AuthedAddMember(ctx context.Context, userID, filter, author string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("user_id", userID),
		zap.String("filter", filter),
		zap.String("author", author),
	)

	if err := perms.CanPerform(ctx, author, "role_admins", deps); err != nil {
		sp.Warn("user doesn't have permission to this command", zap.Error(err))
		return common.SendError(&author, "User doesn't have permission to this command")
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

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	_, err := strconv.Atoi(userID)
	if err != nil {
		if !common.IsDiscordUser(userID) {
			sp.Warn("second argument must be a discord user")
			return common.SendError(nil, "second argument must be a discord user")
		}
		userID = common.ExtractUserId(userID)
	}

	before, err := common.GetMembership(ctx, userID, deps)
	if err != nil {
		sp.Error("error getting membership", zap.Error(err))
		return common.SendFatalf(nil, "Error getting membership: %s", err)
	}

	filterData, err := deps.Storage.GetFilter(ctx, filter)
	if err != nil {
		if errors.Is(err, storage.ErrNoFilter) {
			return common.SendErrorf(nil, "No such filter: %s", filter)
		}

		sp.Error("Error getting filter", zap.Error(err))
		return common.SendErrorf(nil, "Error getting filter: %s", err)
	}
	sp.With(zap.Int("filter_id", filterData.ID))

	sp.Info("Got member info")

	err = deps.Storage.AddFilterMembership(ctx, filterData.ID, userID)
	if err != nil {
		if errors.Is(err, storage.ErrFilterMember) {
			return common.SendErrorf(
				nil,
				"%s Already member of %s",
				common.GetUsername(userID, deps.Session),
				filter,
			)
		}

		sp.Error("error getting membership")
		return common.SendFatalf(nil, "Error getting membership: %s", err)
	}

	after, err := common.GetMembership(ctx, userID, deps)
	if err != nil {
		sp.Error("error getting membership")
		return common.SendFatalf(nil, "Error getting membership: %s", err)
	}

	addSet := after.Difference(before)

	if addSet.Len() == 0 {
		/* TODO: this error is not always correct. If someone joins a sig they aren't a member of all filters of
		 * it appears like they aren't in it.
		 */
		return common.SendErrorf(
			nil,
			"%s already a member of %s (maybe)",
			common.GetUsername(userID, deps.Session),
			filter,
		)
	}

	for _, role := range addSet.ToSlice() {
		QueueUpdate(ctx, payloads.Upsert, userID, role, deps)
	}

	sp.Info("added user to filter")
	return common.SendSuccessf(nil, "Added <@%s> to `%s`", userID, filter)
}

func AuthedRemoveMember(ctx context.Context, userID, filter, author string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("user_id", userID),
		zap.String("filter", filter),
		zap.String("author", author),
	)

	if err := perms.CanPerform(ctx, author, "role_admins", deps); err != nil {
		sp.Warn("user doesn't have permission to this command", zap.Error(err))
		return common.SendError(&author, "User doesn't have permission to this command")
	}

	sp.Debug("removing filter member")
	return RemoveMember(ctx, userID, filter, deps)
}

func RemoveMember(ctx context.Context, userID, filterName string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("user_id", userID),
		zap.String("filter", filterName),
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	before, err := common.GetMembership(ctx, userID, deps)
	if err != nil {
		sp.Error("error getting membership")
		return common.SendFatalf(nil, "Error getting membership: %s", err)
	}

	_, err = strconv.Atoi(userID)
	if err != nil {
		if !common.IsDiscordUser(userID) {
			sp.Warn("second argument must be a discord user")
			return common.SendError(nil, "second argument must be a discord user")
		}
		userID = common.ExtractUserId(userID)
	}

	filter, err := deps.Storage.GetFilter(ctx, filterName)
	if err != nil {
		if errors.Is(err, storage.ErrNoFilter) {
			return common.SendErrorf(nil, "No such filter: %s", filter)
		}

		sp.Error("error getting filter")
		return common.SendErrorf(nil, "Error getting filter: %s", filter)
	}

	sp.With(zap.Int("filter_id", filter.ID))

	err = deps.Storage.DeleteFilterMembership(ctx, filter.ID, userID)
	if err != nil {
		sp.Error("Error deleting filter membership", zap.Error(err))
		return common.SendErrorf(
			nil,
			"Error removing %s from filter %s membership",
			common.GetUsername(userID, deps.Session),
			filter,
		)
	}

	after, err := common.GetMembership(ctx, userID, deps)
	if err != nil {
		sp.Error("error getting membership")
		return common.SendFatalf(nil, "Error getting membership: %s", err)
	}

	removeSet := before.Difference(after)

	if removeSet.Len() == 0 {
		return common.SendErrorf(nil, "<@%s> not a member of `%s`", userID, filter.Name)
	}

	for _, role := range removeSet.ToSlice() {
		QueueUpdate(ctx, payloads.Delete, userID, role, deps)
	}

	sp.Info("removed user from filter")
	return common.SendSuccessf(
		nil,
		"Removed %s from `%s`",
		common.GetUsername(userID, deps.Session),
		filter.Name,
	)
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
