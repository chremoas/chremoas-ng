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
	Role   *discordgo.Role `json:"role,omitempty"`
	Filter *Filter         `json:"filter,omitempty"`
	Member string          `json:"member,omitempty"`
}

// Filter is the filter data structure
type Filter struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// TODO: Find a better place for these now that they aren't a part of the payload

// Role is the role data structure
type Role struct {
	ID          int64  `json:"id,omitempty"`
	Color       int32  `json:"color,omitempty"`
	Hoist       bool   `json:"hoist,omitempty"`
	Joinable    bool   `json:"joinable,omitempty"`
	Managed     bool   `json:"managed,omitempty"`
	Mentionable bool   `json:"mentionable,omitempty"`
	Name        string `json:"name"`
	Permissions int32  `json:"permissions,omitempty"`
	Position    int32  `json:"position,omitempty"`
	ShortName   string `json:"role_nick"`
	Sig         bool   `json:"sig,omitempty"`
	Sync        bool   `json:"sync,omitempty"`
	Type        string `json:"chat_type"`
}
