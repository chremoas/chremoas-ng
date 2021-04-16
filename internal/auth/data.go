package auth

import (
	"time"
)

type AuthenticationScope struct {
	ID   int64  `db:"id" json:"id"`
	Name string `dn:"name" json:"name"`
}

type User struct {
	ID     int64  `db:"id" json:"id"`
	ChatID string `dn:"chat_id" json:"chatId"`
}

type Alliance struct {
	ID         int64      `db:"id" json:"json"`
	Name       string     `db:"name" json:"name"`
	Ticker     string     `db:"ticker" json:"ticker"`
	InsertedAt *time.Time `db:"inserted_at" json:"insertedAt"`
	UpdatedAt  *time.Time `db:"updated_at" json:"updatedAt"`
}

type Corporation struct {
	ID         int64      `db:"id" json:"id"`
	Name       string     `db:"name" json:"name"`
	AllianceID int64      `db:"alliance_id" json:"allianceId"`
	InsertedAt *time.Time `db:"inserted_at" json:"insertedAt"`
	UpdatedAt  *time.Time `db:"updated_at" json:"updatedAt"`
	Ticker     string     `db:"ticker" json:"ticker"`
}

type Character struct {
	ID            int64      `db:"id" json:"id"`
	Name          string     `db:"name" json:"name"`
	InsertedAt    *time.Time `db:"inserted_at" json:"insertedAt"`
	UpdatedAt     *time.Time `db:"updated_at" json:"updatedAt"`
	CorporationId int64      `db:"corporation_id" json:"corporationId"`
	Token         string     `db:"token" json:"token"`
}

type AuthenticationCode struct {
	CharacterID int64  `db:"character_id" json:"characterId"`
	Code        string `db:"code" json:"code"`
	Used        bool   `db:"used" json:"used"`
}

type UserCharacterMap struct {
	UserID      int64 `db:"userId" json:"userId"`
	CharacterID int64 `db:"character_id" json:"characterId"`
}

type AuthenticationScopeCharacterMap struct {
	CharacterID int64 `db:"character_id" json:"characterId"`
	ScopeID     int64 `db:"scope_id" json:"scopeId"`
}

type CreateRequest struct {
	Token       string       `json:"token,omitempty"`
	Character   *Character   `json:"character,omitempty"`
	Corporation *Corporation `json:"corporation,omitempty"`
	Alliance    *Alliance    `json:"alliance,omitempty"`
	AuthScope   []string     `json:"authScope,omitempty"`
}
