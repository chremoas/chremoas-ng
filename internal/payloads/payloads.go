package payloads

type Action string

const (
	CreateRole Action = "createRole"
	DeleteRole Action = "DeleteRole"
	UpdateRole Action = "updateRole"
)

type Payload struct {
	Action Action `json:"action"`
	Data   interface{}
}

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
