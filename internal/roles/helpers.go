package roles

import (
	"context"
	"encoding/json"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	sl "github.com/bhechinger/spiffylogger"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	"go.uber.org/zap"
)

func getRoleID(name string, deps common.Dependencies) (int, error) {
	var (
		err error
		id  int
	)

	err = deps.DB.Select("id").
		From("roles").
		Where(sq.Eq{"role_nick": name}).
		Scan(&id)

	return id, err
}

func validListItem(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func queueUpdate(ctx context.Context, role payloads.Role, action payloads.Action, deps common.Dependencies) error {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	payload := payloads.RolePayload{
		Action:  action,
		GuildID: deps.GuildID,
		Role:    role,
	}

	b, err := json.Marshal(payload)
	if err != nil {
		sp.Error("error marshalling json for queue", zap.Error(err))
		return err
	}

	sp.Debug("Submitting role queue message",
		zap.String("name", role.Name),
		zap.String("id", role.ID),
		zap.Int64("chat_id", role.ChatID),
	)
	err = deps.RolesProducer.Publish(ctx, b)
	if err != nil {
		sp.Error("error publishing message", zap.Error(err))
		return err
	}

	return nil
}

// GetRoleMembers lists all userIDs that match all the filters for a role.
func GetRoleMembers(ctx context.Context, sig bool, name string, deps common.Dependencies) ([]int, error) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	var (
		err        error
		id         int
		members    []int
		filterList []int
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	rows, err := deps.DB.Select("role_filters.filter").
		From("role_filters").
		InnerJoin("roles ON role_filters.role = roles.id").
		Where(sq.Eq{"sig": sig}).
		Where(sq.Eq{"role_nick": name}).
		QueryContext(ctx)
	if err != nil {
		sp.Error("error getting role filters", zap.Error(err))
		return nil, err
	}
	defer func() {
		if err = rows.Close(); err != nil {
			sp.Error("error closing row", zap.Error(err))
		}
	}()

	// add filters to the membership query
	for rows.Next() {
		err = rows.Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("error scanning role's id (%s): %s", name, err.Error())
		}

		filterList = append(filterList, id)
	}

	rows, err = deps.DB.Select("user_id").
		From("filter_membership").
		Where(sq.Eq{"filter": filterList}).
		GroupBy("user_id").
		Having("count(*) = ?", len(filterList)).
		QueryContext(ctx)
	if err != nil {
		sp.Error("error getting filter memberhship", zap.Error(err))
		return nil, err
	}
	defer func() {
		if err = rows.Close(); err != nil {
			sp.Error("error closing row", zap.Error(err))
		}
	}()

	// add filters to the membership query
	for rows.Next() {
		err = rows.Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("error scanning filter's userID (%s): %s", name, err.Error())
		}

		members = append(members, id)
	}

	return members, nil
}

// GetRoles goes and fetches all the roles of type sig/role. If shortname is set only one role is fetched.
func GetRoles(ctx context.Context, sig bool, shortName *string, deps common.Dependencies) ([]payloads.Role, error) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	var (
		rs        []payloads.Role
		charTotal int
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	q := deps.DB.Select("color", "hoist", "joinable", "managed", "mentionable", "name", "permissions",
		"position", "role_nick", "sig", "sync").
		Where(sq.Eq{"sig": sig}).
		From("roles")

	if shortName != nil {
		q = q.Where(sq.Eq{"role_nick": shortName})
	}

	rows, err := q.QueryContext(ctx)
	if err != nil {
		newErr := fmt.Errorf("error getting %ss: %s", roleType[sig], err)
		sp.Error("error getting role", zap.Error(err), zap.Bool("sig", sig))
		return nil, newErr
	}
	defer func() {
		if err = rows.Close(); err != nil {
			sp.Error("error closing row", zap.Error(err))
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
			sp.Error("error scanning role", zap.Error(err), zap.Bool("sig", sig))
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
