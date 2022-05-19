package common

import (
	"context"
	"fmt"
	"strconv"

	"github.com/bhechinger/go-sets"
	sl "github.com/bhechinger/spiffylogger"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	"go.uber.org/zap"
)

const (
	Role = false
	Sig  = true
)

var roleType = map[bool]string{Role: "role", Sig: "sig"}

func GetUserRoles(ctx context.Context, sig bool, userID string, deps Dependencies) ([]payloads.Role, error) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("role_type", roleType[sig]),
		zap.String("user_id", userID),
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	_, err := strconv.Atoi(userID)
	if err != nil {
		if !IsDiscordUser(userID) {
			sp.Warn("second argument must be a discord user")
			return nil, fmt.Errorf("second argument must be a discord user")
		}
		userID = ExtractUserId(userID)
	}

	return deps.Storage.GetMemberRoles(ctx, userID, sig)
}

func GetMembership(ctx context.Context, userID string, deps Dependencies) (*sets.StringSet, error) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(zap.String("user_id", userID))

	sigs, err := GetUserRoles(ctx, Sig, userID, deps)
	if err != nil {
		sp.Error("Error getting user roles", zap.Error(err))
		return nil, err
	}

	roles, err := GetUserRoles(ctx, Role, userID, deps)
	if err != nil {
		sp.Error("Error getting user roles", zap.Error(err))
		return nil, err
	}

	output := RoleToSet(sigs)
	output.Merge(RoleToSet(roles))

	return output, nil
}

func RoleToSet(roles []payloads.Role) *sets.StringSet {
	newSet := sets.NewStringSet()

	for _, role := range roles {
		newSet.Add(fmt.Sprintf("%d", role.ChatID))
	}

	return newSet
}
