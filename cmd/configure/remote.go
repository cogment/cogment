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

package configure

import (
	"bufio"
	"fmt"
	"github.com/spf13/viper"
	"gitlab.com/cogment/cogment/helper"
	"io"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

type Origin struct {
	Token string
	Url   string
}

const defaultApiUrl = "https://platform-gateway.cogsaas.com"

func NewRemoteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remote [name]",
		Short: "Add a remote platform",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := "default"
			if len(args) > 0 {
				name = args[0]
			}

			if err := runConfigureRemoteCmd(name, os.Stdin); err != nil {
				log.Fatalln(err)
			}

			fmt.Printf("%s remote has been added to %s\n", name, helper.CfgFile)
		},
	}

}

func createOriginFromReader(stdin io.Reader) *Origin {
	reader := bufio.NewReader(stdin)
	o := Origin{}

	fmt.Printf("URL (%s): ", defaultApiUrl)
	url, _ := reader.ReadString('\n')
	url = strings.TrimSuffix(url, "\n")
	if len(url) < 1 {
		url = defaultApiUrl
	}
	o.Url = url

	fmt.Print("API token: ")
	token, _ := reader.ReadString('\n')
	o.Token = strings.TrimSuffix(token, "\n")

	return &o
}

func runConfigureRemoteCmd(name string, stdin io.Reader) error {
	o := createOriginFromReader(stdin)

	viper.Set(fmt.Sprintf("%s.token", name), o.Token)
	viper.Set(fmt.Sprintf("%s.url", name), o.Url)

	remote := viper.GetString("remote")
	if len(remote) < 1 {
		viper.Set("remote", name)
	}

	if err := viper.WriteConfigAs(helper.CfgFile); err != nil {
		fmt.Printf("Unable to write config : %s", err)
		return err
	}

	return nil
}
