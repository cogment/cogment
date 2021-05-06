// Copyright 2021 Artificial Intelligence Redefined <dev+cogment@ai-r.com>
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

package deployment

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestCreateManifestFromCompose(t *testing.T) {

	var tests = []struct {
		services    []string
		expectedSvc int
		expectedErr error
	}{
		{[]string{}, 5, nil},
		{[]string{""}, 0, fmt.Errorf("")},
		{[]string{"dont-exist"}, 0, fmt.Errorf("")},
		{[]string{"the-driver"}, 1, nil},
		{[]string{"dont-exist", "the-driver"}, 0, fmt.Errorf("")},
		{[]string{"the-driver", "env", "orchestrator"}, 3, nil},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.services, ","), func(t *testing.T) {
			manifest, err := CreateManifestFromCompose("../testdata/docker-compose.yaml", tt.services)

			if tt.expectedErr == nil {
				assert.Len(t, manifest.Services, tt.expectedSvc)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestLoadCompose(t *testing.T) {
	out, err := CreateManifestFromCompose("../testdata/docker-compose.yaml", []string{})

	assert.NoError(t, err)
	assert.Len(t, out.Services, len(out.Services))
	assert.Contains(t, out.Services, "the-driver")
	assert.Contains(t, out.Services, "the-shooter")
	assert.Contains(t, out.Services, "env")
	assert.Contains(t, out.Services, "orchestrator")

	driver := out.Services["the-driver"]
	assert.Equal(t, driver.Image, "registry.gitlab.com/change_me/the_driver")
	assert.True(t, driver.getPushImage())
	assert.Equal(t, "1", driver.Environment["COGMENT_GRPC_REFLECTION"])
	assert.Equal(t, "DRIVER", driver.Environment["AGENT"])

	assert.Len(t, driver.Ports, 1)
	assert.EqualValues(t, 9000, driver.Ports[0].Port)
	assert.False(t, driver.Ports[0].Public)

	shooter := out.Services["the-shooter"]
	assert.EqualValues(t, shooter.Image, "registry.gitlab.com/change_me/the_shooter")
	assert.True(t, shooter.getPushImage())
	assert.Equal(t, "1", shooter.Environment["COGMENT_GRPC_REFLECTION"])
	assert.Equal(t, "On", shooter.Environment["DEBUG"])
	assert.Equal(t, "  shooter  ", shooter.Environment["AGENT"])

	assert.Len(t, shooter.Ports, 1)
	assert.EqualValues(t, 9000, shooter.Ports[0].Port)
	assert.False(t, shooter.Ports[0].Public)

	env := out.Services["env"]
	assert.Equal(t, env.Image, "registry.gitlab.com/change_me/env")
	assert.True(t, env.getPushImage())
	assert.Empty(t, env.Environment)
	assert.Len(t, env.Ports, 2)

	assert.EqualValues(t, 9000, env.Ports[0].Port)
	assert.False(t, env.Ports[0].Public)

	assert.EqualValues(t, 9001, env.Ports[1].Port)
	assert.True(t, env.Ports[1].Public)

	orchestrator := out.Services["orchestrator"]
	assert.Equal(t, orchestrator.Image, "registry.gitlab.com/change_me/orchestrator:2.1")
	assert.True(t, orchestrator.getPushImage())

	assert.Empty(t, orchestrator.Environment)

	assert.EqualValues(t, 9000, orchestrator.Ports[0].Port)
	assert.False(t, orchestrator.Ports[0].Public)

	redis := out.Services["redis"]
	assert.Equal(t, redis.Image, "redis:5-alpine")
	assert.False(t, redis.getPushImage())

}
