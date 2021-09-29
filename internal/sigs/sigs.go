package sigs

import (
	"fmt"
	"strconv"

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

func New(member, sig, author string, deps common.Dependencies) (*Sig, error) {
	_, err := strconv.Atoi(member)
	if err != nil {
		if !common.IsDiscordUser(member) {
			return nil, fmt.Errorf("second argument must be a discord user")
		}
		member = common.ExtractUserId(member)
	}

	role, err := roles.GetRoles(roles.Sig, &sig, deps)
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
		role:   role[0],
		sig:    sig,
		userID: member,
		author: author,
	}, nil
}

func (s Sig) Add() string {
	if !perms.CanPerform(s.author, "sig_admins", s.dependencies) {
		return common.SendError("User not authorized")
	}
	return filters.AddMember(s.userID, s.sig, s.dependencies)
}

func (s Sig) Remove() string {
	if !perms.CanPerform(s.author, "sig_admins", s.dependencies) {
		return common.SendError("User not authorized")
	}
	return filters.RemoveMember(s.userID, s.sig, s.dependencies)
}

func (s Sig) Join() string {
	if !s.role.Joinable {
		return common.SendError(fmt.Sprintf("'%s' is not a joinable SIG, talk to an admin", s.sig))
	}

	return filters.AddMember(s.userID, s.sig, s.dependencies)
}

func (s Sig) Leave() string {
	if !s.role.Joinable {
		return common.SendError(fmt.Sprintf("'%s' is not a joinable SIG, talk to an admin", s.sig))
	}

	return filters.RemoveMember(s.userID, s.sig, s.dependencies)
}
