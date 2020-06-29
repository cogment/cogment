package registry

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestRegistryDeleteCommandWithStatus(t *testing.T) {
	client := resty.New()

	httpmock.ActivateNonDefault(client.GetClient())
	defer httpmock.DeactivateAndReset()

	var tests = []struct {
		registryId string
		statusCode int
		hasErr     bool
	}{
		{"1", 200, false},
		{"2", 404, true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.statusCode), func(t *testing.T) {
			httpmock.Reset()
			httpmock.RegisterResponder("DELETE", "/docker-registries/"+tt.registryId,
				func(req *http.Request) (*http.Response, error) {
					return httpmock.NewJsonResponse(tt.statusCode, "")
				},
			)

			err := runDeleteRegistryCmd(client, tt.registryId)

			assert.Equal(t, 1, httpmock.GetTotalCallCount())

			if tt.hasErr {
				assert.NotNil(t, err)
				assert.Equal(t, "Registry ID not found", err.Error())
			} else {
				assert.Nil(t, err)
			}

		})
	}
}
