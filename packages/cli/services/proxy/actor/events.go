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

//nolint:lll
type SentReward struct {
	// int64 / uint64 values are serialized/deserialized as string to be aligned with the behavior of protobuf
	TickID     int64                  `json:"tick_id,string" description:"The tick associated with the reward"`
	Receiver   string                 `json:"receiver" description:"Name of the receiving actor"`
	Value      float32                `json:"value" description:"The numerical value of the provided reward"`
	Confidence float32                `json:"confidence" description:"The weight of this reward in computing the final (aggregated) reward" validate:"gte=0,lte=1"`
	UserData   map[string]interface{} `json:"user_data,omitempty" description:"Additional user data"`
}

type SentEvent struct {
	// int64 / uint64 values are serialized/deserialized as string to be aligned with the behavior of protobuf
	TickID  uint64                      `json:"tick_id,string"`
	Action  *trialspec.DynamicPbMessage `json:"action,omitempty"`
	Rewards []SentReward                `json:"rewards,omitempty"`
}

//nolint:lll
type RecvRewardSource struct {
	Sender     string                      `json:"sender" description:"Name of the sending actor or environment"`
	Value      float32                     `json:"value" description:"The numerical value of the reward"`
	Confidence float32                     `json:"confidence" description:"The weight of this reward in computing the final (aggregated) reward" validate:"gte=0,lte=1"`
	UserData   *trialspec.DynamicPbMessage `json:"user_data,omitempty" description:"Additional user data"`
}

type RecvReward struct {
	// int64 / uint64 values are serialized/deserialized as string to be aligned with the behavior of protobuf
	TickID  uint64             `json:"tick_id,string" description:"The tick associated with the reward"`
	Value   float32            `json:"value" description:"The numerical value of the final reward"`
	Sources []RecvRewardSource `json:"sources" description:"Sources for the reward"`
}

//nolint:lll
type RecvEvent struct {
	// int64 / uint64 values are serialized/deserialized as string to be aligned with the behavior of protobuf
	TickID      uint64                      `json:"tick_id,string" description:"Identifier of the tick (time step)"`
	Observation *trialspec.DynamicPbMessage `json:"observation,omitempty" description:"Observation received by the actor for this tick"`
	Rewards     []RecvReward                `json:"rewards,omitempty" description:"Rewards received by the actor"`
}
