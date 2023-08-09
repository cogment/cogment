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

package controller

import (
	"encoding/json"
	"path"
	"testing"

	"github.com/cogment/cogment/services/proxy/trialspec"
	"github.com/stretchr/testify/assert"
)

const testDataDir string = "../../../testdata"

func TestTrialParamsJSON(t *testing.T) {
	tsm, err := trialspec.NewFromFile(path.Join(testDataDir, "cogment.yaml"))
	assert.NoError(t, err)

	params := NewTrialParams(tsm)

	input := `{` +
		`"config":{"foo":"testing unmarshalling"},` +
		`"actors": [` +
		`{"actor_class": "plane","default_action": {"path": [{"x": 1, "y": 1},{"x": 2, "y": 2}]}},` +
		`{"actor_class": "ai_drone","config": {"member": "a member"}}` +
		`]}`
	err = json.Unmarshal([]byte(input), &params)
	assert.NoError(t, err)
	assert.Len(t, params.Actors, 2)
}
