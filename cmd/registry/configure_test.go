// Copyright 2020 Artificial Intelligence Redefined <dev+cogment@ai-r.com>
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

package registry

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"gitlab.com/cogment/cogment/api"
	"net/http"
	"testing"
)

func TestRegistryConfigureWithStatusCode(t *testing.T) {
	client := resty.New()

	httpmock.ActivateNonDefault(client.GetClient())
	defer httpmock.DeactivateAndReset()

	httpResponse := map[string]interface{}{
		"id":           1,
		"registry_url": "http://url1.com",
		"created_at":   1572129864,
		"updated_at":   1572395933,
	}

	errorResponse := map[string]interface{}{
		"error": "message",
	}

	var tests = []struct {
		statusCode             int
		expectedDockerRegistry *api.DockerRegistry
		hasErr                 bool
		httpResponse           map[string]interface{}
	}{
		{200, &api.DockerRegistry{
			Id:          1,
			RegistryUrl: "http://url1.com",
			CreatedAt:   1572129864,
			UpdatedAt:   1572395933,
		}, false, httpResponse},
		{201, &api.DockerRegistry{
			Id:          1,
			RegistryUrl: "http://url1.com",
			CreatedAt:   1572129864,
			UpdatedAt:   1572395933,
		}, false, httpResponse},
		{400, nil, true, errorResponse},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.statusCode), func(t *testing.T) {
			httpmock.Reset()
			httpmock.RegisterResponder("POST", `/docker-registries`,
				func(req *http.Request) (*http.Response, error) {
					return httpmock.NewJsonResponse(tt.statusCode, tt.httpResponse)
				},
			)

			dockerRegistry, err := runConfigureRegistryCmd(client, "http://url.com", "username", "password")

			assert.Equal(t, 1, httpmock.GetTotalCallCount())

			if tt.hasErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.expectedDockerRegistry, dockerRegistry)
			}

		})
	}
}
