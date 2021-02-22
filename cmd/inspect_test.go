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

package cmd

import (
	"fmt"
	"github.com/cogment/cogment-cli/api"
	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
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
