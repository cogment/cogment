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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSentRewardJSONSerialization(t *testing.T) {
	reward := SentReward{TickID: 12, Receiver: "John", Value: 78, Confidence: 1}

	output, err := json.Marshal(reward)
	assert.NoError(t, err)

	assert.Equal(t, `{"tick_id":"12","receiver":"John","value":78,"confidence":1}`, string(output))

	input1 := `{"tick_id":"12"}`
	deserializedReward1 := SentReward{}

	err = json.Unmarshal([]byte(input1), &deserializedReward1)
	assert.NoError(t, err)

	assert.Equal(t, int64(12), deserializedReward1.TickID)

	input2 := `{"tick_id":12}`
	deserializedReward2 := SentReward{}

	err = json.Unmarshal([]byte(input2), &deserializedReward2)
	assert.Error(t, err)
}
