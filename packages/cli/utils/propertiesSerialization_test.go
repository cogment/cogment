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

package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParsePropertiesSingle1(t *testing.T) {
	t.Parallel()
	properties, err := ParseProperties("key=value")
	assert.NoError(t, err)
	assert.Len(t, properties, 1)
	assert.Contains(t, properties, "key")
	assert.Equal(t, properties["key"], "value")
}

func TestParsePropertiesSingle2(t *testing.T) {
	t.Parallel()
	properties, err := ParseProperties("k%20ey = v+al")
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{
		"k%20ey": "v+al",
	}, properties)
}

func TestParsePropertiesSingle3(t *testing.T) {
	t.Parallel()
	properties, err := ParseProperties("key=val,key=noval")
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{
		"key": "noval",
	}, properties)
}

func TestParsePropertiesSeveral1(t *testing.T) {
	t.Parallel()
	properties, err := ParseProperties("key=value,key2=val&val,  key3")
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{
		"key":  "value",
		"key2": "val&val",
		"key3": "",
	}, properties)
}

func TestParsePropertiesSeveral2(t *testing.T) {
	t.Parallel()
	properties, err := ParseProperties(" ke y = v al   , key2  ")
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{
		"ke y": "v al",
		"key2": "",
	}, properties)
}

func TestFormatPropertiesSingle(t *testing.T) {
	t.Parallel()
	assert.Equal(t, FormatProperties(map[string]string{
		"key": "value",
	}), "key=value")
}

func TestFormatPropertiesSeveral(t *testing.T) {
	t.Parallel()
	assert.Contains(t,
		[]string{
			"key=value,key2=value2",
			"key2=value2,key=value",
		}, FormatProperties(map[string]string{
			"key":  "value",
			"key2": "value2",
		}))

	assert.Contains(t,
		[]string{
			"key,key2=value2",
			"key2=value2,key",
		}, FormatProperties(map[string]string{
			"key":  "",
			"key2": "value2",
		}))
}
