package commands

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/nsqio/go-nsq"
	"go.uber.org/zap"
)

type Command struct {
	logger *zap.SugaredLogger
	db     *sq.StatementBuilderType
	nsq    *nsq.Producer
}

func New(logger *zap.SugaredLogger, db *sq.StatementBuilderType, nsq *nsq.Producer) *Command {
	return &Command{
		logger: logger,
		db:     db,
		nsq:    nsq,
	}
}
