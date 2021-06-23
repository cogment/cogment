// Copyright 2021 AI Redefined Inc. <dev+cogment@ai-r.com>
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

package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeVersionString(t *testing.T) {

	vStr, err := SanitizeVersion("v1.0.0")
	assert.NoError(t, err)
	assert.Equal(t, vStr, "1.0.0")

	vStr, err = SanitizeVersion("1.0")
	assert.NoError(t, err)
	assert.Equal(t, vStr, "1.0")

	vStr, err = SanitizeVersion("v3")
	assert.NoError(t, err)
	assert.Equal(t, vStr, "3")

	vStr, err = SanitizeVersion("foo")
	assert.Error(t, err)
	assert.Empty(t, vStr)
}
