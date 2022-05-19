package payloads

type Action string

const (
	Add    Action = "add"
	Upsert Action = "upsert"
	Delete Action = "delete"
)

type RolePayload struct {
	Action        Action `json:"action,omitempty"`
	GuildID       string `json:"guildId"`
	Role          Role   `json:"role,omitempty"`
	CorrelationID string `json:"correlation_id"`
}

type MemberPayload struct {
	Action        Action `json:"action"`
	GuildID       string `json:"guildId"`
	MemberID      string `json:"memberId"`
	RoleID        string `json:"roleId"`
	CorrelationID string `json:"correlation_id"`
}

// Filter is the filter data structure
type Filter struct {
	ID          int    `json:"id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// RoleFilter is the filter data structure
type RoleFilter struct {
	ID     int   `json:"id,omitempty"`
	Role   int64 `json:"role"`
	Filter int64 `json:"filter"`
}

// Permission is the filter data structure
type Permission struct {
	ID          int    `json:"id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// TODO: Find a better place for these now that they aren't a part of the payload

// Role is the role data structure
type Role struct {
	// discordgo.Role
	ID          string `json:"id,omitempty"`
	Name        string `json:"name"`
	Managed     bool   `json:"managed,omitempty"`
	Mentionable bool   `json:"mentionable,omitempty"`
	Hoist       bool   `json:"hoist,omitempty"`
	Color       int    `json:"color,omitempty"`
	Position    int    `json:"position,omitempty"`
	Permissions int64  `json:"permissions,omitempty"`

	// Chremoas bits
	ChatID    int64  `json:"chat_id,omitempty"`
	Joinable  bool   `json:"joinable,omitempty"`
	ShortName string `json:"role_nick"`
	Sig       bool   `json:"sig,omitempty"`
	Sync      bool   `json:"sync,omitempty"`
	Type      string `json:"chat_type"`
}
