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
	"github.com/dustin/go-humanize"
	"github.com/go-resty/resty/v2"
	"github.com/ryanuber/columnize"
	"github.com/spf13/cobra"
	"gitlab.com/cogment/cogment/api"
	"gitlab.com/cogment/cogment/deployment"
	"gitlab.com/cogment/cogment/helper"
	"log"
	"net/http"
	"strings"
	"time"
)

// psCmd represents the ps command
var psCmd = &cobra.Command{
	Use:    "ps",
	Short:  "List your applications",
	Hidden: true,

	Run: func(cmd *cobra.Command, args []string) {
		client, err := deployment.PlatformClient(Verbose)
		if err != nil {
			log.Fatal(err)
		}

		apps, err := runPsCmd(client)
		if err != nil {
			log.Fatal(err)
		}

		var output []string
		row := []string{"ID", "NAME", "CREATED AT", "LAST DEPLOYMENT"}
		output = append(output, strings.Join(row, "|"))
		appId := helper.CurrentConfig("app")

		for _, app := range apps {
			if appId == app.Id {
				app.Id = "*" + app.Id
			}

			lastDeployment := "N/A"
			if app.LastDeploymentAt > 0 {
				lastDeployment = humanize.Time(time.Unix(int64(app.LastDeploymentAt), 0))
			}

			row := []string{
				app.Id,
				app.Name,
				humanize.Time(time.Unix(int64(app.CreatedAt), 0)),
				lastDeployment,
			}

			output = append(output, strings.Join(row, "|"))
		}
		result := columnize.SimpleFormat(output)
		fmt.Println(result)
	},
}

func runPsCmd(client *resty.Client) ([]*api.Application, error) {
	var apps []*api.Application

	resp, err := client.R().
		SetResult(&apps).
		Get("/applications")

	if err != nil {
		log.Fatalf("%v", err)
	}

	if http.StatusOK == resp.StatusCode() {
		return apps, nil
	}

	return nil, fmt.Errorf("%s", resp.Body())

}

func init() {
	rootCmd.AddCommand(psCmd)

}
