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

package httpserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToken(t *testing.T) {
	tokenString, err := MakeAndSerializeToken("a_trial", "an_actor", "my_secret")
	assert.NoError(t, err)

	tokenClaims, err := ParseAndVerifyToken(tokenString, "my_secret")
	assert.NoError(t, err)

	assert.Equal(t, "a_trial", tokenClaims.TrialID)
	assert.Equal(t, "an_actor", tokenClaims.ActorName)
}

func TestParseBadToken(t *testing.T) {
	_, err := ParseAndVerifyToken("blabla", "my_secret")
	assert.Error(t, err)
}

func TestParseBadSecret(t *testing.T) {
	tokenString, err := MakeAndSerializeToken("a_trial", "an_actor", "my_secret")
	assert.NoError(t, err)

	_, err = ParseAndVerifyToken(tokenString, "my_secret_is_wrong")
	assert.Error(t, err)
}
