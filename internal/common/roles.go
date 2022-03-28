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

	var roles []payloads.Role

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	_, err := strconv.Atoi(userID)
	if err != nil {
		if !IsDiscordUser(userID) {
			return nil, fmt.Errorf("second argument must be a discord user")
		}
		userID = ExtractUserId(userID)
	}

	query := deps.DB.Select("role_nick", "name", "chat_id").
		From("").
		Suffix("getMemberRoles(?, ?)", userID, strconv.FormatBool(sig))

	LogSQL(sp, query)

	rows, err := query.QueryContext(ctx)
	if err != nil {
		sp.Error("error getting role membership", zap.Error(err), zap.String("user id", userID))
		return nil, fmt.Errorf("error getting user %ss (%s): %s", roleType[sig], userID, err)
	}

	defer func() {
		if err = rows.Close(); err != nil {
			sp.Error("error closing row", zap.Error(err))
		}
	}()

	for rows.Next() {
		var role payloads.Role

		err = rows.Scan(
			&role.ShortName,
			&role.Name,
			&role.ChatID,
		)
		if err != nil {
			newErr := fmt.Errorf("error scanning %s row: %s", roleType[sig], err)
			sp.Error("error scanning row", zap.Error(newErr))
			return nil, newErr
		}

		roles = append(roles, role)
	}

	return roles, nil
}

func GetMembership(ctx context.Context, userID string, deps Dependencies) (*sets.StringSet, error) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sigs, err := GetUserRoles(ctx, Sig, userID, deps)
	if err != nil {
		return nil, err
	}

	roles, err := GetUserRoles(ctx, Role, userID, deps)
	if err != nil {
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
