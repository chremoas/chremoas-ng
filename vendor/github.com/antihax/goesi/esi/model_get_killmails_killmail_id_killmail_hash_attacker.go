/*
 * EVE Swagger Interface
 *
 * An OpenAPI for EVE Online
 *
 * OpenAPI spec version: 1.7.15
 *
 * Generated by: https://github.com/swagger-api/swagger-codegen.git
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package esi

/* A list of GetKillmailsKillmailIdKillmailHashAttacker. */
//easyjson:json
type GetKillmailsKillmailIdKillmailHashAttackerList []GetKillmailsKillmailIdKillmailHashAttacker

/* attacker object */
//easyjson:json
type GetKillmailsKillmailIdKillmailHashAttacker struct {
	AllianceId     int32   `json:"alliance_id,omitempty"`     /* alliance_id integer */
	CharacterId    int32   `json:"character_id,omitempty"`    /* character_id integer */
	CorporationId  int32   `json:"corporation_id,omitempty"`  /* corporation_id integer */
	DamageDone     int32   `json:"damage_done,omitempty"`     /* damage_done integer */
	FactionId      int32   `json:"faction_id,omitempty"`      /* faction_id integer */
	FinalBlow      bool    `json:"final_blow,omitempty"`      /* Was the attacker the one to achieve the final blow  */
	SecurityStatus float32 `json:"security_status,omitempty"` /* Security status for the attacker  */
	ShipTypeId     int32   `json:"ship_type_id,omitempty"`    /* What ship was the attacker flying  */
	WeaponTypeId   int32   `json:"weapon_type_id,omitempty"`  /* What weapon was used by the attacker for the kill  */
}
