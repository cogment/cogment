// Copyright 2023 AI Redefined Inc. <dev+cogment@ai-r.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package actor

import (
	"github.com/cogment/cogment/services/proxy/trialspec"
)

type SentReward struct {
	TickID     int64                  `json:"tick_id"`
	Receiver   string                 `json:"receiver"`
	Value      float32                `json:"value"`
	Confidence float32                `json:"confidence"`
	UserData   map[string]interface{} `json:"user_data,omitempty"`
}

type SentEvent struct {
	TickID  uint64                      `json:"tick_id"`
	Action  *trialspec.DynamicPbMessage `json:"action,omitempty"`
	Rewards []SentReward                `json:"rewards,omitempty"`
}

type RecvRewardSource struct {
	Sender     string                      `json:"sender"`
	Value      float32                     `json:"value"`
	Confidence float32                     `json:"confidence"`
	UserData   *trialspec.DynamicPbMessage `json:"user_data,omitempty"`
}

type RecvReward struct {
	TickID  uint64             `json:"tick_id"`
	Value   float32            `json:"value"`
	Sources []RecvRewardSource `json:"sources"`
}

type RecvEvent struct {
	TickID      uint64                      `json:"tick_id"`
	Observation *trialspec.DynamicPbMessage `json:"observation,omitempty"`
	Rewards     []RecvReward                `json:"rewards,omitempty"`
}
