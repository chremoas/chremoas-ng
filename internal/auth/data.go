package auth

import (
	"time"
)

type User struct {
	ID     int32  `db:"id" json:"id"`
	ChatID string `dn:"chat_id" json:"chatId"`
}

type Alliance struct {
	ID         int32      `db:"id" json:"json"`
	Name       string     `db:"name" json:"name"`
	Ticker     string     `db:"ticker" json:"ticker"`
	InsertedAt *time.Time `db:"inserted_at" json:"insertedAt"`
	UpdatedAt  *time.Time `db:"updated_at" json:"updatedAt"`
}

type Corporation struct {
	ID         int32      `db:"id" json:"id"`
	Name       string     `db:"name" json:"name"`
	AllianceID int32      `db:"alliance_id" json:"allianceId"`
	InsertedAt *time.Time `db:"inserted_at" json:"insertedAt"`
	UpdatedAt  *time.Time `db:"updated_at" json:"updatedAt"`
	Ticker     string     `db:"ticker" json:"ticker"`
}

type Character struct {
	ID            int32      `db:"id" json:"id"`
	Name          string     `db:"name" json:"name"`
	InsertedAt    *time.Time `db:"inserted_at" json:"insertedAt"`
	UpdatedAt     *time.Time `db:"updated_at" json:"updatedAt"`
	CorporationID int32      `db:"corporation_id" json:"corporationId"`
	Token         string     `db:"token" json:"token"`
}

type AuthenticationCode struct {
	CharacterID int32  `db:"character_id" json:"characterId"`
	Code        string `db:"code" json:"code"`
	Used        bool   `db:"used" json:"used"`
}

type UserCharacterMap struct {
	UserID      int32 `db:"userId" json:"userId"`
	CharacterID int32 `db:"character_id" json:"characterId"`
}

type CreateRequest struct {
	Token       string       `json:"token,omitempty"`
	Character   *Character   `json:"character,omitempty"`
	Corporation *Corporation `json:"corporation,omitempty"`
	Alliance    *Alliance    `json:"alliance,omitempty"`
	AuthScope   []string     `json:"authScope,omitempty"`
}
