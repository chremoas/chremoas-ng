package commands

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/bwmarrin/discordgo"
	"github.com/nsqio/go-nsq"
	"go.uber.org/zap"
)

type Command struct {
	logger  *zap.SugaredLogger
	db      *sq.StatementBuilderType
	nsq     *nsq.Producer
	discord *discordgo.Session
}

func New(logger *zap.SugaredLogger, db *sq.StatementBuilderType, nsq *nsq.Producer, discord *discordgo.Session) *Command {
	return &Command{
		logger: logger,
		db:     db,
		nsq:    nsq,
		discord:     discord,
	}
}
