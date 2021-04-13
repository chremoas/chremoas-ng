package sigs

import (
	"fmt"
	"strconv"

	sq "github.com/Masterminds/squirrel"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/filters"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	"github.com/chremoas/chremoas-ng/internal/perms"
	"github.com/chremoas/chremoas-ng/internal/roles"
	"github.com/nsqio/go-nsq"
	"go.uber.org/zap"
)

type Sig struct {
	logger *zap.SugaredLogger
	db     *sq.StatementBuilderType
	nsq    *nsq.Producer
	role   payloads.Role
	sig    string
	userID string
	author string
}

func New(member, sig, author string, logger *zap.SugaredLogger, db *sq.StatementBuilderType, nsq *nsq.Producer) (*Sig, error) {
	_, err := strconv.Atoi(member)
	if err != nil {
		if !common.IsDiscordUser(member) {
			return nil, fmt.Errorf("second argument must be a discord user")
		}
		member = common.ExtractUserId(member)
	}

	role, err := roles.GetRoles(roles.Sig, &sig, logger, db)
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
		logger: logger,
		db:     db,
		nsq:    nsq,
		role:   role[0],
		sig:    sig,
		userID: member,
		author: author,
	}, nil
}

func (s Sig) Add() string {
	if !perms.CanPerform(s.author, "sig_admins", s.logger, s.db) {
		return common.SendError("User not authorized")
	}
	return filters.AddMember(roles.Sig, s.userID, s.sig, s.author, s.logger, s.db, s.nsq)
}

func (s Sig) Remove() string {
	if !perms.CanPerform(s.author, "sig_admins", s.logger, s.db) {
		return common.SendError("User not authorized")
	}
	return filters.RemoveMember(roles.Sig, s.userID, s.sig, s.author, s.logger, s.db, s.nsq)
}

func (s Sig) Join() string {
	if !s.role.Joinable {
		return common.SendError(fmt.Sprintf("'%s' is not a joinable SIG, talk to an admin", s.sig))
	}

	return s.Add()
}

func (s Sig) Leave() string {
	if !s.role.Joinable {
		return common.SendError(fmt.Sprintf("'%s' is not a joinable SIG, talk to an admin", s.sig))
	}

	return s.Remove()
}
