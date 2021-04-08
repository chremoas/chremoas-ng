package commands

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/disgord/x/mux"

	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/roles"
)

const helpStr = `
Usage: !role <subcommand> <arguments>

Subcommands:
    *list: List all Roles*
    create: Add Role
    destroy: Delete role
    *info: Get Role Info
    *keys: Get valid role keys
    *types: Get valid role types
    set: Set role key
    *list_members: List Role members
    *list_roles: List user Roles
`

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func (c Command) Role(s *discordgo.Session, m *discordgo.Message, ctx *mux.Context) {
	var (
		all bool
		rs  string
	)

	c.logger.Infof("Recieved: %s", m.Content)
	cmdStr := strings.Split(m.Content, " ")

	switch cmdStr[1] {
	case "create":
		if len(cmdStr) < 4 {
			rs = "Usage: !role create <role_name> <role_description>"
		} else {
			rs = roles.Add(cmdStr[2], strings.Join(cmdStr[3:], " "), "discord", c.logger, c.db, c.nsq)
		}

	case "destroy":
		if len(cmdStr) < 3 {
			rs = "Usage: !role destroy <role_name>"
		} else {
			rs = roles.Destroy(cmdStr[2], c.logger, c.db, c.nsq)
		}

	case "list":
		if len(cmdStr) > 2 && cmdStr[2] == "all" {
			all = true
		}
		rs = roles.List(c.logger, c.db, false, all)

	case "info":
		if len(cmdStr) < 3 {
			rs = "Usage: !role info <role_name>"
		} else {
			rs = roles.Info(cmdStr[2], false, c.logger, c.db)
		}

	case "keys":
		rs = roles.Keys()

	case "types":
		rs = roles.Types()

	case "list_members":
		if len(cmdStr) < 3 {
			rs = "Usage: !role list_members <role_name>"
		} else {
			rs = roles.Members(cmdStr[2], c.logger, c.db)
		}

	case "list_roles":
		if len(cmdStr) < 3 {
			rs = roles.ListUserRoles(m.Author.ID, c.logger, c.db)
		} else {
			rs = roles.ListUserRoles(common.ExtractUserId(cmdStr[2]), c.logger, c.db)
		}

	case "help":
	default:
		rs = fmt.Sprintf("```%s```", helpStr)
	}

	_, err := s.ChannelMessageSend(m.ChannelID, rs)
	if err != nil {
		c.logger.Errorf("Error sending command: %s", err)
	}
}
