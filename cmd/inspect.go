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
	"github.com/cogment/cogment-cli/deployment"
	"github.com/cogment/cogment-cli/helper"
	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"log"
	"net/http"
)

// inspectCmd represents the inspect command
var inspectCmd = &cobra.Command{
	Use:    "inspect",
	Short:  "Inspect an application",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := deployment.PlatformClient(Verbose)
		if err != nil {
			log.Fatal(err)
		}

		app, err := runInspectCmd(client)
		if err != nil {
			log.Fatal(err)

		}

		fmt.Println(helper.PrettyPrint(app))

	},
}

func runInspectCmd(client *resty.Client) (*api.ApplicationDetails, error) {
	appId := helper.CurrentConfig("app")
	if appId == "" {
		log.Fatal("No current application found, maybe try `cogment new`")
	}

	resp, err := client.R().
		SetResult(&api.ApplicationDetails{}).
		Get("/applications/" + appId)

	if err != nil {
		log.Fatalf("%v", err)
	}

	if http.StatusNotFound == resp.StatusCode() {
		return nil, fmt.Errorf("%s", "Application not found")
	}

	if http.StatusOK == resp.StatusCode() {
		app := resp.Result().(*api.ApplicationDetails)
		return app, nil
	}

	return nil, fmt.Errorf("%s", resp.Body())
}

func init() {
	rootCmd.AddCommand(inspectCmd)
}
