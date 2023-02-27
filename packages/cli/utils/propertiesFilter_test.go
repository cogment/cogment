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

func TestSelectsExact(t *testing.T) {
	filter := NewPropertiesFilter(map[string]string{"key": "value", "key2": "", "key3": "value3"})
	assert.True(t, filter.Selects(map[string]string{"key": "value", "key2": "", "key3": "value3"}))
	assert.False(t, filter.Selects(map[string]string{"key": "value", "key2": "blah", "key3": "value3"}))
	assert.False(t, filter.Selects(map[string]string{"key": "value", "key2": "", "key3": ""}))
}

func TestSelectsSubset(t *testing.T) {
	filter := NewPropertiesFilter(map[string]string{"key": "", "key2": "value2"})
	assert.True(t, filter.Selects(map[string]string{"key": "", "key2": "value2", "key3": "value3"}))
	assert.False(t, filter.Selects(map[string]string{"key": "", "key2": "blah", "key3": "value3"}))
	assert.False(t, filter.Selects(map[string]string{"key": "value", "key2": "value2", "key3": ""}))
}

func TestSelectsEmpty(t *testing.T) {
	filter := NewPropertiesFilter(map[string]string{})
	assert.True(t, filter.Selects(map[string]string{}))
	assert.True(t, filter.Selects(map[string]string{"key": "", "key2": "blah", "key3": "value3"}))
}
