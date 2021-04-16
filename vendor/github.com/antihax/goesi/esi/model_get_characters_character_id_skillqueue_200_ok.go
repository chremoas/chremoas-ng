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

import (
	"time"
)

/* A list of GetCharactersCharacterIdSkillqueue200Ok. */
//easyjson:json
type GetCharactersCharacterIdSkillqueue200OkList []GetCharactersCharacterIdSkillqueue200Ok

/* 200 ok object */
//easyjson:json
type GetCharactersCharacterIdSkillqueue200Ok struct {
	FinishDate      time.Time `json:"finish_date,omitempty"`       /* Date on which training of the skill will complete. Omitted if the skill queue is paused. */
	FinishedLevel   int32     `json:"finished_level,omitempty"`    /* finished_level integer */
	LevelEndSp      int32     `json:"level_end_sp,omitempty"`      /* level_end_sp integer */
	LevelStartSp    int32     `json:"level_start_sp,omitempty"`    /* Amount of SP that was in the skill when it started training it's current level. Used to calculate % of current level complete. */
	QueuePosition   int32     `json:"queue_position,omitempty"`    /* queue_position integer */
	SkillId         int32     `json:"skill_id,omitempty"`          /* skill_id integer */
	StartDate       time.Time `json:"start_date,omitempty"`        /* start_date string */
	TrainingStartSp int32     `json:"training_start_sp,omitempty"` /* training_start_sp integer */
}