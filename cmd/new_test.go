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
	"bytes"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"gitlab.com/cogment/cogment/api"
	"net/http"
	"testing"
)

func TestNewCommandInteractiveWithStatusCode(t *testing.T) {

	const name = "my application"
	const id = "my-app-731841"
	const createdAt = 1570558016

	client := resty.New()
	//client.SetDebug(false)

	httpmock.ActivateNonDefault(client.GetClient())
	defer httpmock.DeactivateAndReset()

	var tests = []struct {
		statusCode          int
		expectedApplication *api.Application
		hasErr              bool
	}{
		{201, &api.Application{Id: id, Name: name, CreatedAt: createdAt}, false},
		{400, nil, true},
		{401, nil, true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.statusCode), func(t *testing.T) {
			viper.Reset()
			viper.SetFs(afero.NewMemMapFs())
			initConfig()
			viper.Set("remote", "default")

			httpmock.Reset()
			httpmock.RegisterResponder("POST", `/applications`,
				func(req *http.Request) (*http.Response, error) {
					return httpmock.NewJsonResponse(tt.statusCode, map[string]interface{}{
						"id":         id,
						"name":       name,
						"created_at": createdAt,
					})
				},
			)

			var stdin bytes.Buffer
			//1st input for url, 2nd for name
			stdin.Write([]byte("\n" + name + "\n"))

			application, err := runNewCmd(&cobra.Command{}, client, &stdin)

			assert.Equal(t, 1, httpmock.GetTotalCallCount())
			assert.Equal(t, tt.expectedApplication, application)

			if tt.hasErr {
				assert.NotNil(t, err)
				assert.Empty(t, viper.GetString("default.app"))
			} else {

				if err := viper.ReadInConfig(); err != nil {
					t.Fatal("Unable to read config: ", err)
				}

				assert.Nil(t, err)
				assert.Equal(t, id, viper.GetString("default.app"))
			}

		})
	}
}
