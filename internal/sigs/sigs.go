package sigs

import (
	"context"
	"fmt"
	"strconv"

	sl "github.com/bhechinger/spiffylogger"
	"github.com/bwmarrin/discordgo"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/filters"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	"github.com/chremoas/chremoas-ng/internal/perms"
	"github.com/chremoas/chremoas-ng/internal/roles"
	"go.uber.org/zap"
)

type Sig struct {
	dependencies common.Dependencies
	role         payloads.Role
	sig          string
	userID       string
	author       string
}

func New(ctx context.Context, member, sig, author string, deps common.Dependencies) (*Sig, error) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("member", member),
		zap.String("sig", sig),
	)

	_, err := strconv.Atoi(member)
	if err != nil {
		if !common.IsDiscordUser(member) {
			sp.Error("second argument must be a discord user")
			return nil, fmt.Errorf("second argument must be a discord user")
		}
		member = common.ExtractUserId(member)
	}

	roleList, err := deps.Storage.GetRolesByType(ctx, roles.Sig)
	if err != nil {
		sp.Error("error getting roles")
		return nil, err
	}
	if len(roleList) == 0 {
		sp.Error("no such sig")
		return nil, fmt.Errorf("no such sig: `%s`", sig)
	}
	if !roleList[0].Sig {
		sp.Error("not a sig")
		return nil, fmt.Errorf("not a sig: `%s`", sig)
	}

	return &Sig{
		dependencies: deps,
		role:         roleList[0],
		sig:          sig,
		userID:       member,
		author:       author,
	}, nil
}

func (s Sig) Add(ctx context.Context) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	if err := perms.CanPerform(ctx, s.author, "sig_admins", s.dependencies); err != nil {
		sp.Error("User not authorized", zap.Error(err))
		return common.SendError(&s.author, "User not authorized")
	}
	return filters.AddMember(ctx, s.userID, s.sig, s.dependencies)
}

func (s Sig) Remove(ctx context.Context) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	if err := perms.CanPerform(ctx, s.author, "sig_admins", s.dependencies); err != nil {
		sp.Error("User not authorized", zap.Error(err))
		return common.SendError(&s.author, "User not authorized")
	}
	return filters.RemoveMember(ctx, s.userID, s.sig, s.dependencies)
}

func (s Sig) Join(ctx context.Context) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	if !s.role.Joinable {
		return common.SendErrorf(&s.author, "'%s' is not a joinable SIG, talk to an admin", s.sig)
	}

	return filters.AddMember(ctx, s.userID, s.sig, s.dependencies)
}

func (s Sig) Leave(ctx context.Context) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	if !s.role.Joinable {
		return common.SendErrorf(&s.author, "'%s' is not a joinable SIG, talk to an admin", s.sig)
	}

	return filters.RemoveMember(ctx, s.userID, s.sig, s.dependencies)
}
