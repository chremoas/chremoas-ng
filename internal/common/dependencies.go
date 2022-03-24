package common

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/bwmarrin/discordgo"
	"github.com/chremoas/chremoas-ng/internal/queue"
)

type Dependencies struct {
	DB              *sq.StatementBuilderType
	MembersProducer *queue.Producer
	RolesProducer   *queue.Producer
	Session         *discordgo.Session
	GuildID         string
}
