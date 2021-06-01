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

	"github.com/magiconair/properties/assert"
)

func TestSnakeify(t *testing.T) {
	var tests = []struct {
		in       string
		expected string
	}{
		{"The     Agent", "the_agent"},
		{"a-dashing-name", "a_dashing_name"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			out := Snakeify(tt.in)

			assert.Equal(t, out, tt.expected)
		})
	}
}

func TestKebabify(t *testing.T) {
	var tests = []struct {
		in       string
		expected string
	}{
		{"The     Agent", "the-agent"},
		{"not_an_ __url", "not-an-url"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			out := Kebabify(tt.in)

			assert.Equal(t, out, tt.expected)
		})
	}
}

func TestPascalify(t *testing.T) {
	var tests = []struct {
		in       string
		expected string
	}{
		{"the     agent", "TheAgent"},
		{"the_agent", "TheAgent"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			out := Pascalify(tt.in)

			assert.Equal(t, out, tt.expected)
		})
	}
}
