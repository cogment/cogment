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
	"bufio"
	"fmt"
	"github.com/cogment/cogment-cli/deployment"
	"github.com/cogment/cogment-cli/helper"
	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:    "delete [service1 service2 ...]",
	Short:  "Delete an application ",
	Hidden: true,

	Run: func(cmd *cobra.Command, args []string) {
		client, err := deployment.PlatformClient(Verbose)
		if err != nil {
			log.Fatal(err)
		}

		err = runDeleteCmd(client, args, os.Stdin)
		if err != nil {
			log.Fatal(err)
		}
	},
}

func getAcceptDelete(stdin io.Reader, appId string, services []string) string {
	reader := bufio.NewReader(stdin)

	svcString := appId + " and all its services"
	if len(services) > 0 {
		svcString = fmt.Sprintf("%s from %s", strings.Join(services, ","), appId)
	}

	fmt.Printf("Do you want to delete %s (y/N): ", svcString)
	accept, _ := reader.ReadString('\n')
	accept = strings.TrimSpace(accept)
	return accept
}

func runDeleteCmd(client *resty.Client, services []string, stdin io.Reader) error {
	appId := helper.CurrentConfig("app")
	if appId == "" {
		log.Fatal("No current application found, maybe try `cogment new`")
	}

	query := ""
	if len(services) > 0 {
		query = fmt.Sprintf("services=%s", strings.Join(services, ","))
	}

	accept := getAcceptDelete(stdin, appId, services)
	if accept != "y" {
		fmt.Println("Deletion aborted")
		os.Exit(0)
	}

	url := fmt.Sprintf("/applications/%s?%s", appId, query)
	resp, err := client.R().Delete(url)
	if err != nil {
		return err
	}

	if http.StatusNoContent != resp.StatusCode() {
		return fmt.Errorf("an error occured. Try verbose mode -v")
	}

	return nil
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
