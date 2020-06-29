package cmd

import (
	"bytes"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestDeleteCommandWith(t *testing.T) {

	const appId = "my-app-731841"

	client := resty.New()

	viper.SetFs(afero.NewMemMapFs())
	initConfig()
	viper.Set("remote", "default")
	viper.Set("default.app", appId)

	httpmock.ActivateNonDefault(client.GetClient())
	defer httpmock.DeactivateAndReset()

	var tests = []struct {
		statusCode          int
		services            []string
		expectedQueryParams string
		expectedErr         error
	}{
		{204, []string{}, "", nil},
		{204, []string{"svc1"}, "services=svc1", nil},
		{204, []string{"svc1", "svc2"}, "services=svc1,svc2", nil},
		{400, []string{}, "", fmt.Errorf("an error occured. Try verbose mode -v")},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.statusCode), func(t *testing.T) {

			httpmock.Reset()

			httpmock.RegisterResponder("DELETE", `=~^/applications/(.*)\?(.*)`,

				func(req *http.Request) (*http.Response, error) {
					id, err := httpmock.GetSubmatch(req, 1) // 1=first regexp submatch
					assert.NoError(t, err)
					assert.Equal(t, appId, id)

					query, err := httpmock.GetSubmatch(req, 2)
					assert.NoError(t, err)
					assert.Equal(t, tt.expectedQueryParams, query)

					return httpmock.NewBytesResponse(tt.statusCode, []byte{}), nil
				},
			)

			var stdin bytes.Buffer
			stdin.Write([]byte("y\n"))

			err := runDeleteCmd(client, tt.services, &stdin)

			assert.Equal(t, 1, httpmock.GetTotalCallCount())
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}
