package commands

import (
	"github.com/chremoas/chremoas-ng/internal/common"
)

type Command struct {
	dependencies common.Dependencies
}

func New(deps common.Dependencies) *Command {
	return &Command{
		dependencies: deps,
	}
}
