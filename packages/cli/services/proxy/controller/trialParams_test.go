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
	"fmt"
	"path"
	"testing"

	"github.com/cogment/cogment/services/proxy/trialspec"
	"github.com/stretchr/testify/assert"
	"gopkg.in/go-playground/validator.v8"
)

const testDataDir string = "../../../testdata"

func TestTrialParamsFromJSON(t *testing.T) {
	tsm, err := trialspec.NewFromFile(path.Join(testDataDir, "cogment.yaml"))
	assert.NoError(t, err)

	params := &TrialParams{}

	input := `{` +
		`"config":{"foo":"testing unmarshalling"},` +
		`"environment":{"implementation":"my_environment"},` +
		`"actors": [` +
		`{"actor_class": "plane","default_action": {"path": [{"x": 1, "y": 1},{"x": 2, "y": 2}]}},` +
		`{"actor_class": "ai_drone","config": {"member": "a member"}}` +
		`]}`
	err = json.Unmarshal([]byte(input), &params)
	assert.NoError(t, err)
	assert.Len(t, params.Actors, 2)

	pbParams, err := params.FullyUnmarshal(tsm)
	assert.NoError(t, err)
	assert.Len(t, pbParams.Actors, 2)
}

func TestInvalidTrialParamsFromJSON(t *testing.T) {
	tsm, err := trialspec.NewFromFile(path.Join(testDataDir, "cogment.yaml"))
	assert.NoError(t, err)

	params := &TrialParams{}

	input := `{` +
		`"config":{"foo":"testing unmarshalling"},` +
		`"environment":{"implementation":"my_environment"},` +
		`"actors": []` +
		`}`
	err = json.Unmarshal([]byte(input), &params)
	assert.NoError(t, err)
	assert.Len(t, params.Actors, 0)

	_, err = params.FullyUnmarshal(tsm)

	var validationErr validator.ValidationErrors
	assert.ErrorAs(t, err, &validationErr)
	fmt.Println(validationErr)
	assert.Len(t, validationErr, 1)
	assert.Contains(t, validationErr, "TrialParams.Actors")
}
