package payloads

import (
	"github.com/bwmarrin/discordgo"
)

type Action string

const (
	Create Action = "create"
	Delete Action = "Delete"
	Update Action = "update"
)

type Payload struct {
	Action Action          `json:"action"`
	Role   *discordgo.Role `json:"data"`
	Filter *Filter         `json:"filter"`
	Member string          `json:"member"`
}

// Filter is the filter data structure
type Filter struct {
	Name        string `db:"name"`
	Description string `db:"description"`
}

// TODO: Find a better place for these now that they aren't a part of the payload

// Role is the role data structure
type Role struct {
	Color       int32  `db:"color"`
	Hoist       bool   `db:"hoist"`
	Joinable    bool   `db:"joinable"`
	Managed     bool   `db:"managed"`
	Mentionable bool   `db:"mentionable"`
	Name        string `db:"name"`
	Permissions int32  `db:"permissions"`
	Position    int32  `db:"position"`
	ShortName   string `db:"role_nick"`
	Sig         bool   `db:"sig"`
	Sync        bool   `db:"sync"`
	Type        string `db:"chat_type"`
}
