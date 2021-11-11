/*
 * EVE Swagger Interface
 *
 * An OpenAPI for EVE Online
 *
 * OpenAPI spec version: 1.8.2
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

/* A list of GetUniverseRaces200Ok. */
//easyjson:json
type GetUniverseRaces200OkList []GetUniverseRaces200Ok

/* 200 ok object */
//easyjson:json
type GetUniverseRaces200Ok struct {
	AllianceId  int32  `json:"alliance_id,omitempty"` /* The alliance generally associated with this race */
	Description string `json:"description,omitempty"` /* description string */
	Name        string `json:"name,omitempty"`        /* name string */
	RaceId      int32  `json:"race_id,omitempty"`     /* race_id integer */
}
