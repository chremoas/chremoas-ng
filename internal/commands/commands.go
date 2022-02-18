package commands

import (
	"github.com/bwmarrin/discordgo"
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

func getHelp(title, usage, subCommands string) []*discordgo.MessageSend {
	var embeds []*discordgo.MessageSend

	embed := common.NewEmbed()
	embed.SetTitle(title)
	embed.AddField("Usage", usage)
	if subCommands != "" {
		embed.AddField("Subcommands", subCommands)
	}

	return append(embeds, &discordgo.MessageSend{Embed: embed.GetMessageEmbed()})
}
