package common

import (
	"github.com/bwmarrin/discordgo"
	"github.com/chremoas/chremoas-ng/internal/queue"
	"github.com/chremoas/chremoas-ng/internal/storage"
)

type Dependencies struct {
	Storage         *storage.Storage
	MembersProducer *queue.Producer
	RolesProducer   *queue.Producer
	Session         *discordgo.Session
	GuildID         string
}
