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
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/cogment/cogment-cli/api"
	"github.com/cogment/cogment-cli/deployment"
	"github.com/docker/docker/pkg/term"
	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"io"
	"log"
	"net/http"
	"os"
)

var Username string

func runConfigureRegistryCmd(client *resty.Client, registryUrl string, username string, password string) (*api.DockerRegistry, error) {
	registryConfiguration := api.RegistryConfiguration{RegistryUrl: registryUrl, Username: username, Password: password}

	resp, err := client.R().
		SetBody(registryConfiguration).
		SetResult(&api.DockerRegistry{}).
		Post("/docker-registries")

	if err != nil {
		log.Fatalf("%v", err)
	}

	if http.StatusCreated == resp.StatusCode() || http.StatusOK == resp.StatusCode() {
		dockerRegistry := resp.Result().(*api.DockerRegistry)
		return dockerRegistry, nil
	}

	fmt.Println(fmt.Errorf("%s", resp.Body()))

	return nil, fmt.Errorf("%s", resp.Body())
}

func NewRegistryConfigureCommand(verbose bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "configure [REGISTRY URL]",
		Short: "Configure credentials for a Docker Registry",
		Run: func(cmd *cobra.Command, args []string) {
			registryUrl := "https://index.docker.io/v1/"

			if len(args) > 0 {
				registryUrl = args[0]
			}

			in := os.Stdin.Fd()

			oldState, err := term.SaveState(in)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Fprintf(os.Stdout, "Password: ")
			term.DisableEcho(in, oldState)

			password := readInput(cmd.InOrStdin(), cmd.OutOrStdout())
			fmt.Fprint(os.Stdout, "\n")

			term.RestoreTerminal(in, oldState)

			client, err := deployment.PlatformClient(verbose)
			if err != nil {
				log.Fatal(err)
			}

			dockerRegistry, err := runConfigureRegistryCmd(client, registryUrl, Username, password)

			if err != nil {
				log.Fatal(err)
			}

			responseBody, err := json.Marshal(dockerRegistry)

			if err != nil {
				log.Fatal(err)
			}

			fmt.Println(string(responseBody))
		},
	}

	cmd.Flags().StringVarP(&Username, "username", "u", "", "Username")
	cmd.MarkFlagRequired("username")

	return cmd
}

func readInput(in io.Reader, out io.Writer) string {
	reader := bufio.NewReader(in)
	line, _, err := reader.ReadLine()
	if err != nil {
		log.Fatal(err)
	}
	return string(line)
}
