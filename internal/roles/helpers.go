package roles

import (
	"context"
	"encoding/json"

	sl "github.com/bhechinger/spiffylogger"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	"go.uber.org/zap"
)

// I don't think this is actually used
// func getRoleID(name string, deps common.Dependencies) (int, error) {
// 	var (
// 		err error
// 		id  int
// 	)
//
// 	err = deps.DB.Select("id").
// 		From("roles").
// 		Where(sq.Eq{"role_nick": name}).
// 		Scan(&id)
//
// 	return id, err
// }

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
		Action:        action,
		GuildID:       deps.GuildID,
		Role:          role,
		CorrelationID: sp.GetCorrelationID(),
	}

	sp.With(
		zap.Any("role", role),
		zap.Any("action", action),
		zap.Any("payload", payload),
	)

	b, err := json.Marshal(payload)
	if err != nil {
		sp.Error("error marshalling json for queue", zap.Error(err))
		return err
	}

	sp.Debug("Submitting role queue message")
	err = deps.RolesProducer.Publish(ctx, b)
	if err != nil {
		sp.Error("error publishing message", zap.Error(err))
		return err
	}

	return nil
}

// GetRoleMembers lists all userIDs that match all the filters for a role.
func GetRoleMembers(ctx context.Context, sig bool, name string, deps common.Dependencies) ([]int64, error) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("name", name),
		zap.String("role_type", roleType[sig]),
	)

	var (
		err        error
		filterList []int64
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	roleFilters, err := deps.Storage.GetRoleFilters(ctx, sig, name)
	if err != nil {
		sp.Error("Error getting role filters")
		return nil, err
	}

	for f := range roleFilters {
		filterList = append(filterList, roleFilters[f].Filter)
	}

	members, err := deps.Storage.GetRoleMembers(ctx, filterList)
	if err != nil {
		sp.Error("Error getting role members", zap.Error(err))
		return nil, err
	}

	return members, nil
}
