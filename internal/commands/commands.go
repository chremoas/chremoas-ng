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

func getHelp(title, usage, subCommands string) []*common.Embed {
	var embeds []*common.Embed

	embed := common.NewEmbed()
	embed.SetTitle(title)
	embed.AddField("Usage", usage)
	embed.AddField("Subcommands", subCommands)
	return append(embeds, embed)
}
