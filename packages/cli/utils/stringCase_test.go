// Copyright 2022 AI Redefined Inc. <dev+cogment@ai-r.com>
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

package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSnakeify(t *testing.T) {
	assert.Equal(t, Snakeify("The     Agent"), "the_agent")
	assert.Equal(t, Snakeify("a-dashing-name"), "a_dashing_name")
}

func TestKebabify(t *testing.T) {
	assert.Equal(t, Kebabify("The     Agent"), "the-agent")
	assert.Equal(t, Kebabify("not_an_ __url"), "not-an-url")
}

func TestPascalify(t *testing.T) {
	assert.Equal(t, Pascalify("The     Agent"), "TheAgent")
	assert.Equal(t, Pascalify("the_agent"), "TheAgent")
}
