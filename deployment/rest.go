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

package deployment

import (
	"encoding/json"
	"errors"
	"github.com/cogment/cogment-cli/helper"
	"github.com/go-resty/resty/v2"
)

func PlatformClient(verbose bool) (*resty.Client, error) {
	baseURL := helper.CurrentConfig("url")
	token := helper.CurrentConfig("token")

	if baseURL == "" {
		return nil, errors.New("API URL is not defined, maybe try `cogment configure remote`")
	}

	client := resty.New()
	client.SetHostURL(baseURL)
	client.SetHeader("Authorization", "Token "+token)
	//client.SetAuthToken(token)
	client.SetDebug(verbose)

	// Registering global Error object structure for JSON/XML request
	//client.SetError(Error{}) // or resty.SetError(Error{})

	return client, nil
}

func ResponseFormat(i interface{}) ([]byte, error) {
	return json.Marshal(i)
}
