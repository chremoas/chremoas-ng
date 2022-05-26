package roles

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"

	sl "github.com/bhechinger/spiffylogger"
	"github.com/bwmarrin/discordgo"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/filters"
	"github.com/chremoas/chremoas-ng/internal/goof"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	"github.com/chremoas/chremoas-ng/internal/perms"
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

	roles, err := deps.Storage.GetRolesByType(ctx, sig)
	if err != nil {
		sp.Error("error getting roles", zap.Error(err))
		return common.SendFatal(err.Error())
	}

	roleList := make([]string, 0, len(roles))

	for _, role := range roles {
		if sig && !role.Joinable && !all {
			continue
		}

		if !sig && !role.Sync && !all {
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

	role, err := deps.Storage.GetRoleByType(ctx, sig, ticker)
	if err != nil {
		sp.Error("error getting roles", zap.Error(err))
		return common.SendFatal(err.Error())
	}

	buffer.WriteString(fmt.Sprintf("ShortName: %s\n", role.ShortName))
	buffer.WriteString(fmt.Sprintf("Name: %s\n", role.Name))
	buffer.WriteString(fmt.Sprintf("Color: #%06x\n", role.Color))
	buffer.WriteString(fmt.Sprintf("Hoist: %t\n", role.Hoist))
	buffer.WriteString(fmt.Sprintf("Position: %d\n", role.Position))
	buffer.WriteString(fmt.Sprintf("Permissions: %d\n", role.Permissions))
	buffer.WriteString(fmt.Sprintf("Manged: %t\n", role.Managed))
	buffer.WriteString(fmt.Sprintf("Mentionable: %t\n", role.Mentionable))
	if sig {
		buffer.WriteString(fmt.Sprintf("Joinable: %t\n", role.Joinable))
	}
	buffer.WriteString(fmt.Sprintf("Sync: %t\n", role.Sync))

	embed := common.NewEmbed()
	embed.SetTitle(fmt.Sprintf("Info for %s %s", roleType[sig], role.Name))
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

	if err := perms.CanPerform(ctx, author, adminType[sig], deps); err != nil {
		sp.Warn("user doesn't have permission to this command", zap.Error(err))
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

	roleID, err := deps.Storage.InsertRole(ctx, name, ticker, chatType, sig, joinable)
	if err != nil {
		sp.Error("Error inserting role", zap.Error(err))
		return common.SendError("Error inserting role")
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

	err = deps.Storage.InsertRoleFilter(ctx, roleID, filterID)
	if err != nil {
		sp.Error("Error inserting role filter", zap.Error(err))
		return common.SendError("Error inserting role filter")
	}

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

	if err := perms.CanPerform(ctx, author, adminType[sig], deps); err != nil {
		sp.Warn("user doesn't have permission to this command", zap.Error(err))
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

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if len(ticker) == 0 {
		return common.SendError("short name is required")
	}

	role, err := deps.Storage.GetRole(ctx, "", ticker, &sig)
	if err != nil {
		sp.Error("Error getting role", zap.Error(err))
		return common.SendError("Error getting role")
	}

	sp.With(zap.Int64("chat_id", role.ChatID))

	err = deps.Storage.DeleteRole(ctx, ticker, sig)
	if err != nil {
		sp.Error("Error deleting role", zap.Error(err))
		return common.SendError("Error deleting role")
	}

	err = queueUpdate(ctx, payloads.Role{ID: fmt.Sprintf("%d", role.ChatID)}, payloads.Delete, deps)
	if err != nil {
		sp.Error("error deleting role", zap.Error(err))
		return common.SendFatal(fmt.Sprintf("error deleting role for %s: %s", roleType[sig], err))
	}

	sp.Info("deleted role")
	messages := common.SendSuccess(fmt.Sprintf("Destroyed %s `%s`", roleType[sig], ticker))

	embed := common.NewEmbed()
	embed.SetTitle("filter response")
	messages = append(messages, &discordgo.MessageSend{Embed: embed.GetMessageEmbed()})

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

	if err := perms.CanPerform(ctx, author, adminType[sig], deps); err != nil {
		sp.Warn("user doesn't have permission to this command", zap.Error(err))
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

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// ShortName, Key and Value are required so let's check for those
	if len(ticker) == 0 {
		return common.SendError("ticker is required")
	}

	if len(values) == 0 {
		return common.SendError("values is required")
	}

	roleData, err := deps.Storage.GetRoleByType(ctx, sig, ticker)

	sp.With(zap.Bool("sync", roleData.Sync))

	err = deps.Storage.UpdateRoleValues(ctx, sig, ticker, values)
	if err != nil {
		sp.Error("Error updating role", zap.Error(err))
		return common.SendError("Error updating role")
	}

	role, err := GetChremoasRole(ctx, sig, ticker, deps)
	if err != nil {
		sp.Error("error fetching role", zap.Error(err))
		return common.SendFatal(fmt.Sprintf("error fetching %s from db: %s", roleType[sig], err))
	}

	sp.With(zap.Any("role", role))

	dRole, err := GetDiscordRole(ctx, role.Name, deps)
	if err != nil {
		sp.Info("error getting discord role", zap.Error(err))
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
		err = deps.Storage.UpdateRole(ctx, dRole.ID, role.Name, "")
		if err != nil {
			sp.Error("Error updating Role", zap.Error(err))
		}
	}

	if !roleData.Sync {
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

	role, err := deps.Storage.GetRoleByType(ctx, sig, ticker)
	if err != nil {
		return payloads.Role{}, err
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

	sp.Info("no such role")
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
		results  bool
		messages []*discordgo.MessageSend
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	filterList, err := deps.Storage.GetTickerFilters(ctx, sig, ticker)
	if err != nil {
		sp.Error("Error getting ticker filters", zap.Error(err))
		return common.SendError("Error getting filters")
	}

	for f := range filterList {
		if !results {
			results = true
		}

		buffer.WriteString(fmt.Sprintf("%s\n", filterList[f].Name))
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

	if err := perms.CanPerform(ctx, author, adminType[sig], deps); err != nil {
		sp.Warn("user doesn't have permission to this command", zap.Error(err))
		return common.SendError("User doesn't have permission to this command")
	}

	sp.Debug("adding filter")
	return AddFilter(ctx, sig, filter, ticker, deps)
}

func AddFilter(ctx context.Context, sig bool, name, ticker string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("filter", name),
		zap.String("ticker", ticker),
		zap.String("role_type", roleType[sig]),
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	filter, err := deps.Storage.GetFilter(ctx, name)
	if err != nil {
		sp.Error("Error getting filter", zap.Error(err))
		return common.SendError("Error getting filter")
	}

	sp.With(zap.Int("filter_id", filter.ID))

	role, err := deps.Storage.GetRole(ctx, "", ticker, &sig)
	if err != nil {
		sp.Error("Error getting role", zap.Error(err))
		return common.SendError("Error getting role")
	}

	sp.With(zap.String("role_id", role.ID))

	roleID, err := strconv.Atoi(role.ID)
	if err != nil {
		sp.Error("Error converting role ID from string to int", zap.String("role.ID", role.ID), zap.Error(err))
		return common.SendError("Error converting role ID from string to int")
	}

	err = deps.Storage.InsertRoleFilter(ctx, roleID, filter.ID)
	if err != nil {
		sp.Error("Error inserting role filter", zap.Error(err))
		return common.SendError("Error inserting role filter")
	}

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

	if err := perms.CanPerform(ctx, author, adminType[sig], deps); err != nil {
		sp.Warn("user doesn't have permission to this command", zap.Error(err))
		return common.SendError("User doesn't have permission to this command")
	}

	sp.Debug("removing filter")
	return RemoveFilter(ctx, sig, filter, ticker, deps)
}

func RemoveFilter(ctx context.Context, sig bool, name, ticker string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("role_type", roleType[sig]),
		zap.String("filter", name),
		zap.String("ticker", ticker),
	)

	err := deps.Storage.DeleteFilter(ctx, name)
	if err != nil {
		if errors.Is(err, goof.NotMember) {
			return common.SendError("User not a member of filter")
		}

		sp.Error("Error deleting filter", zap.Error(err))
		return common.SendError("Error deleting filter")
	}

	sp.Info("removed filter")
	return common.SendSuccess(fmt.Sprintf("Removed filter %s from role %s", name, ticker))
}
