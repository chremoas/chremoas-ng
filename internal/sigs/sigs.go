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

	_, err := strconv.Atoi(member)
	if err != nil {
		if !common.IsDiscordUser(member) {
			return nil, fmt.Errorf("second argument must be a discord user")
		}
		member = common.ExtractUserId(member)
	}

	role, err := roles.GetRoles(ctx, roles.Sig, &sig, deps)
	if err != nil {
		return nil, err
	}
	if len(role) == 0 {
		return nil, fmt.Errorf("no such sig: `%s`", sig)
	}
	if !role[0].Sig {
		return nil, fmt.Errorf("not a sig: `%s`", sig)
	}

	return &Sig{
		dependencies: deps,
		role:         role[0],
		sig:          sig,
		userID:       member,
		author:       author,
	}, nil
}

func (s Sig) Add(ctx context.Context) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	if !perms.CanPerform(ctx, s.author, "sig_admins", s.dependencies) {
		return common.SendError("User not authorized")
	}
	return filters.AddMember(ctx, s.userID, s.sig, s.dependencies)
}

func (s Sig) Remove(ctx context.Context) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	if !perms.CanPerform(ctx, s.author, "sig_admins", s.dependencies) {
		return common.SendError("User not authorized")
	}
	return filters.RemoveMember(ctx, s.userID, s.sig, s.dependencies)
}

func (s Sig) Join(ctx context.Context) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	if !s.role.Joinable {
		return common.SendError(fmt.Sprintf("'%s' is not a joinable SIG, talk to an admin", s.sig))
	}

	return filters.AddMember(ctx, s.userID, s.sig, s.dependencies)
}

func (s Sig) Leave(ctx context.Context) []*discordgo.MessageSend {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	if !s.role.Joinable {
		return common.SendError(fmt.Sprintf("'%s' is not a joinable SIG, talk to an admin", s.sig))
	}

	return filters.RemoveMember(ctx, s.userID, s.sig, s.dependencies)
}
