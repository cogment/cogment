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
	"github.com/cogment/cogment-cli/api"
	"github.com/cogment/cogment-cli/deployment"
	"github.com/dustin/go-humanize"
	"github.com/go-resty/resty/v2"
	"github.com/ryanuber/columnize"
	"github.com/spf13/cobra"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func runListRegistryCmd(client *resty.Client) ([]*api.DockerRegistry, error) {
	var registries []*api.DockerRegistry

	resp, err := client.R().
		SetResult(&registries).
		Get("/docker-registries")

	if err != nil {
		log.Fatalf("%v", err)
	}

	if http.StatusOK == resp.StatusCode() {
		return registries, nil
	}

	return nil, fmt.Errorf("%s", resp.Body())
}

func NewRegistryListCommand(verbose bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List configured Docker registries",
		Run: func(cmd *cobra.Command, args []string) {
			client, err := deployment.PlatformClient(verbose)
			if err != nil {
				log.Fatal(err)
			}

			registries, err := runListRegistryCmd(client)
			if err != nil {
				log.Fatal(err)
			}

			var output []string
			row := []string{"ID", "REGISTRY URL", "CREATED AT", "UPDATED AT"}
			output = append(output, strings.Join(row, "|"))

			for _, registry := range registries {

				var row = []string{
					strconv.Itoa(registry.Id),
					registry.RegistryUrl,
					humanize.Time(time.Unix(int64(registry.CreatedAt), 0)),
					humanize.Time(time.Unix(int64(registry.UpdatedAt), 0)),
				}
				output = append(output, strings.Join(row, "|"))
			}
			result := columnize.SimpleFormat(output)
			fmt.Println(result)
		},
	}

	return cmd
}
