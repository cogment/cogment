package cmd

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"gitlab.com/cogment/cogment/api"
	"net/http"
	"testing"
)

func TestInspectCommandWithStatusCode(t *testing.T) {

	const appId = "my-app-731841"

	client := resty.New()
	//client.SetDebug(false)

	viper.SetFs(afero.NewMemMapFs())
	initConfig()
	viper.Set("app", appId)

	httpmock.ActivateNonDefault(client.GetClient())
	defer httpmock.DeactivateAndReset()

	myErr := make(map[string]interface{})
	myErr["detail"] = "error"

	appDetails := &api.ApplicationDetails{}

	var tests = []struct {
		statusCode  int
		response    interface{}
		expectedApp *api.ApplicationDetails
		expectedErr error
	}{
		{200, appDetails, appDetails, nil},
		{400, myErr, nil, fmt.Errorf("{\"detail\":\"error\"}")},
		{401, myErr, nil, fmt.Errorf("{\"detail\":\"error\"}")},
		{404, "invalid", nil, fmt.Errorf("Application not found")},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.statusCode), func(t *testing.T) {

			httpmock.Reset()
			httpmock.RegisterResponder("GET", fmt.Sprintf("/applications/%s", appId),
				func(req *http.Request) (*http.Response, error) {
					return httpmock.NewJsonResponse(tt.statusCode, tt.response)
				},
			)

			app, err := runInspectCmd(client)

			assert.Equal(t, 1, httpmock.GetTotalCallCount())
			assert.Equal(t, tt.expectedApp, app)
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}
