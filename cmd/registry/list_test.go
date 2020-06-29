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

func TestRegistryListCommandWithStatusCode(t *testing.T) {
	client := resty.New()

	httpmock.ActivateNonDefault(client.GetClient())
	defer httpmock.DeactivateAndReset()

	httpResponse := []map[string]interface{}{
		{
			"id":           1,
			"registry_url": "http://url1.com",
			"created_at":   1572129864,
			"updated_at":   1572395933,
		}, {
			"id":           2,
			"registry_url": "http://url2.com",
			"created_at":   1572395934,
			"updated_at":   1572395939,
		},
	}

	errorResponse := []map[string]interface{}{
		{
			"error": "details",
		},
	}

	var tests = []struct {
		statusCode               int
		expectedDockerRegistries []*api.DockerRegistry
		hasErr                   bool
		httpResponse             []map[string]interface{}
	}{
		{200, []*api.DockerRegistry{
			{
				Id:          1,
				RegistryUrl: "http://url1.com",
				CreatedAt:   1572129864,
				UpdatedAt:   1572395933,
			}, {
				Id:          2,
				RegistryUrl: "http://url2.com",
				CreatedAt:   1572395934,
				UpdatedAt:   1572395939,
			},
		}, false, httpResponse},
		{400, nil, true, errorResponse},
		{401, nil, true, errorResponse},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.statusCode), func(t *testing.T) {
			httpmock.Reset()
			httpmock.RegisterResponder("GET", `/docker-registries`,
				func(req *http.Request) (*http.Response, error) {
					return httpmock.NewJsonResponse(tt.statusCode, tt.httpResponse)
				},
			)

			dockerRegistries, err := runListRegistryCmd(client)

			assert.Equal(t, 1, httpmock.GetTotalCallCount())
			assert.Equal(t, tt.expectedDockerRegistries, dockerRegistries)

			if tt.hasErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.expectedDockerRegistries, dockerRegistries)
			}

		})
	}
}
