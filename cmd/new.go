/*
Copyright © 2019 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"bufio"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"gitlab.com/cogment/cogment/api"
	"gitlab.com/cogment/cogment/deployment"
	"gitlab.com/cogment/cogment/helper"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

// newCmd represents the new command
var newCmd = &cobra.Command{
	Use:    "new",
	Short:  "Create an application on the platform",
	Hidden: true,

	Run: func(cmd *cobra.Command, args []string) {
		client, err := deployment.PlatformClient(Verbose)
		if err != nil {
			log.Fatal(err)
		}

		app, err := runNewCmd(cmd, client, os.Stdin)
		if err != nil {
			log.Fatal(err)

		}

		out, err := deployment.ResponseFormat(app)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(string(out))
	},
}

func createApplicationFromReader(flags *pflag.FlagSet, stdin io.Reader) *api.Application {
	var name string
	var err error

	if flags.Changed("name") {
		name, err = flags.GetString("name")
	} else {
		fmt.Print("Name: ")
		reader := bufio.NewReader(stdin)
		name, err = reader.ReadString('\n')
	}

	if err != nil {
		log.Fatalf("%v", err)
	}

	name = strings.TrimSuffix(name, "\n")
	app := api.Application{Name: name}

	return &app
}

func runNewCmd(cmd *cobra.Command, client *resty.Client, stdin io.Reader) (*api.Application, error) {
	project := createApplicationFromReader(cmd.Flags(), stdin)

	resp, err := client.R().
		SetBody(project).
		SetResult(&api.Application{}).
		Post("/applications")

	if err != nil {
		log.Fatalf("%v", err)
	}

	if http.StatusCreated == resp.StatusCode() {
		app := resp.Result().(*api.Application)

		remote := viper.GetString("remote")
		viper.Set(fmt.Sprintf("%s.app", remote), app.Id)

		err = viper.WriteConfigAs(helper.CfgFile)
		if err != nil {
			fmt.Println(err)
		}
		return app, nil
	}

	return nil, fmt.Errorf("%s", resp.Body())

}

func init() {
	rootCmd.AddCommand(newCmd)

	newCmd.Flags().StringP("name", "n", "", "Name of your application")
}
