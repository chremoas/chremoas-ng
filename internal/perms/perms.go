package perms

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	sl "github.com/bhechinger/spiffylogger"
	"github.com/bwmarrin/discordgo"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/storage"
	"go.uber.org/zap"
)

const serverAdmins = "server_admins"

func List(ctx context.Context, channelID string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	var permList []string

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	permissions, err := deps.Storage.GetPermissions(ctx)
	if err != nil {
		sp.Error("Error getting permissions", zap.Error(err))
		return common.SendErrorf(nil, "Error getting permissions: %s", err)
	}

	for p := range permissions {
		permList = append(permList, fmt.Sprintf("%s: %s", permissions[p].Name, permissions[p].Description))
	}

	if len(permList) == 0 {
		sp.Error("no permissions")
		return common.SendError("No permissions")
	}

	err = common.SendChunkedMessage(ctx, channelID, "Perm List", permList, deps)
	if err != nil {
		sp.Error("Error sending chunked message")
		return common.SendErrorf(nil, "Error sending chunked message: %s", err)
	}

	return nil
}

func Add(ctx context.Context, permission, description, author string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("required_permission", serverAdmins),
		zap.String("permission", permission),
		zap.String("description", description),
		zap.String("author", author),
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if permission == serverAdmins {
		sp.Warn("user doesn't have rights to this permission")
		return common.SendError("User doesn't have rights to this permission", author)
	}

	if err := CanPerform(ctx, author, serverAdmins, deps); err != nil {
		sp.Warn("user doesn't have permission to this command", zap.Error(err))
		return common.SendError("User doesn't have permission to this command", author)
	}

	err := deps.Storage.InsertPermission(ctx, permission, description)
	if err != nil {
		if errors.Is(err, storage.ErrPermissionExists) {
			return common.SendErrorf(&author, "Permission already exists: %s", permission)
		}

		sp.Error("Error Inserting permission", zap.Error(err))
		return common.SendErrorf(&author, "Error inserting permission: %s", permission)
	}

	sp.Info("created permission")
	return common.SendSuccessf(nil, "Created permission `%s`", permission)
}

func Delete(ctx context.Context, permission, author string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("required_permission", serverAdmins),
		zap.String("permission", permission),
		zap.String("author", author),
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if permission == serverAdmins {
		sp.Warn("user doesn't have rights to this permission")
		return common.SendError("User doesn't have rights to this permission", author)
	}

	if err := CanPerform(ctx, author, serverAdmins, deps); err != nil {
		sp.Warn("user doesn't have permission to this command", zap.Error(err))
		return common.SendError("User doesn't have permission to this command", author)
	}

	err := deps.Storage.DeletePermission(ctx, permission)
	if err != nil {
		sp.Error("Error deleting permission")
		return common.SendErrorf(&author, "Error deleting permission: %s", permission)
	}

	sp.Info("deleted permission")
	return common.SendSuccessf(nil, "Deleted permission `%s`", permission)
}

func ListMembers(ctx context.Context, permission string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(zap.String("permission", permission))

	var (
		count    int
		buffer   bytes.Buffer
		messages []*discordgo.MessageSend
	)

	userIDs, err := deps.Storage.ListPermissionMembers(ctx, permission)
	if err != nil {
		sp.Error("Error listing permission membership")
		return common.SendErrorf(nil, "Error listing permission membership for %s", permission)
	}

	for u := range userIDs {
		buffer.WriteString(fmt.Sprintf("%s\n", common.GetUsername(userIDs[u], deps.Session)))
		count += 1
	}

	if count == 0 {
		sp.Warn("permission has no members")
		return common.SendErrorf(nil, "Permission has no members: %s", permission)
	}

	embed := common.NewEmbed()
	embed.SetTitle(fmt.Sprintf("%s members", permission))
	embed.SetDescription(buffer.String())

	return append(messages, &discordgo.MessageSend{Embed: embed.GetMessageEmbed()})
}

func AddMember(ctx context.Context, user, permission, author string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("required_permission", serverAdmins),
		zap.String("permission", permission),
		zap.String("user", user),
		zap.String("author", author),
	)

	var permissionID int

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if permission == serverAdmins {
		sp.Warn("user doesn't have rights to this permission")
		return common.SendError("User doesn't have rights to this permission", author)
	}

	if err := CanPerform(ctx, author, serverAdmins, deps); err != nil {
		sp.Warn("user doesn't have permission to this command", zap.Error(err))
		return common.SendError("User doesn't have permission to this command", author)
	}

	if !common.IsDiscordUser(user) {
		sp.Warn("second argument must be a discord user")
		return common.SendError("second argument must be a discord user", author)
	}

	userID := common.ExtractUserId(user)

	perm, err := deps.Storage.GetPermission(ctx, permission)
	if err != nil {
		if errors.Is(err, storage.ErrNoPermission) {
			return common.SendErrorf(&author, "No such permission: %s", permission)
		}

		sp.Error("Error getting permission", zap.Error(err))
		return common.SendErrorf(&author, "Error getting permission: %s", permission)
	}

	sp.With(zap.Int("permission_id", perm.ID))

	err = deps.Storage.InsertPermissionMembership(ctx, permissionID, userID)
	if err != nil {
		if errors.Is(err, storage.ErrPermissionMember) {
			return common.SendErrorf(&author, "Already a member of permission: %s", permission)
		}

		return common.SendErrorf(&author, "Error inserting permission membership: %s", permission)
	}

	sp.Info("added user to permission")
	return common.SendSuccessf(
		nil,
		"Added %s to `%s`",
		common.GetUsername(userID, deps.Session),
		permission,
	)
}

func RemoveMember(ctx context.Context, user, permission, author string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("required_permission", serverAdmins),
		zap.String("permission", permission),
		zap.String("user", user),
		zap.String("author", author),
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if permission == serverAdmins {
		sp.Warn("user doesn't have rights to this permission")
		return common.SendError("User doesn't have rights to this permission", author)
	}

	if err := CanPerform(ctx, author, serverAdmins, deps); err != nil {
		sp.Warn("user doesn't have permission to this command", zap.Error(err))
		return common.SendError("User doesn't have permission to this command", author)
	}

	if !common.IsDiscordUser(user) {
		sp.Warn("second argument must be a discord user")
		return common.SendError("second argument must be a discord user", author)
	}

	userID := common.ExtractUserId(user)

	sp.With(zap.String("userID", userID))

	perm, err := deps.Storage.GetPermission(ctx, permission)
	if err != nil {
		sp.Error("Error getting permission", zap.Error(err))
		return common.SendErrorf(&author, "Error getting permission: %s", permission)
	}

	sp.With(zap.Int("permission_id", perm.ID))

	err = deps.Storage.DeletePermissionMembership(ctx, perm.ID, userID)
	if err != nil {
		sp.Error("Error removing permission membership", zap.Error(err))
		return common.SendErrorf(&author, "Error removing permission memberhsip: %s", err)
	}

	sp.Info("removed user from permission")
	return common.SendSuccessf(
		nil,
		"Removed <@%s> from `%s`",
		common.GetUsername(userID, deps.Session),
		permission,
	)
}

func UserPerms(ctx context.Context, user string, deps common.Dependencies) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(zap.String("user", user))

	var (
		buffer   bytes.Buffer
		messages []*discordgo.MessageSend
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if !common.IsDiscordUser(user) {
		sp.Warn("second argument must be a discord user")
		return common.SendError("second argument must be a discord user")
	}

	userID := common.ExtractUserId(user)

	permissions, err := deps.Storage.GetUserPermissions(ctx, userID)
	if err != nil {
		sp.Error("Error getting user permissions", zap.Error(err))
		return common.SendError("Error getting user permissions", userID)
	}

	for p := range permissions {
		buffer.WriteString(fmt.Sprintf("\t%s\n", permissions[p].Name))
	}

	embed := common.NewEmbed()
	embed.SetTitle(fmt.Sprintf("%s's Permissions", common.GetUsername(userID, deps.Session)))
	embed.SetDescription(buffer.String())

	return append(messages, &discordgo.MessageSend{Embed: embed.GetMessageEmbed()})
}

func CanPerform(ctx context.Context, authorID, permission string, deps common.Dependencies) error {
	_, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("author_id", authorID),
		zap.String("permission", permission),
	)

	// This is super jank, and I don't like it, need to come up with a better way.
	if authorID == "auth-web" {
		return nil
	}

	perm, err := deps.Storage.GetPermission(ctx, permission)
	if err != nil {
		sp.Error("Error getting permission", zap.Error(err))
		return err
	}

	sp.With(zap.Int("permission_id", perm.ID))

	count, err := deps.Storage.GetPermissionCount(ctx, authorID, perm.ID)
	if err != nil {
		sp.Error("Error getting permission count", zap.Error(err))
		return err
	}

	sp.With(zap.Int("count", count))

	if count == 0 {
		return err
	}

	return nil
}
