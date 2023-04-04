// Copyright 2023 AI Redefined Inc. <dev+cogment@ai-r.com>
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
	"os"

	"github.com/cogment/cogment/cmd/client"
	"github.com/cogment/cogment/cmd/deprecated"
	"github.com/cogment/cogment/cmd/services"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cogment",
	Short: "Cogment",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Services
	rootCmd.AddCommand(services.ServicesCmd)

	// Client
	rootCmd.AddCommand(client.ClientCmd)

	// Version
	rootCmd.AddCommand(versionCmd)

	// Launch
	rootCmd.AddCommand(launchCmd)

	// Deprecated commands
	rootCmd.AddCommand(deprecated.CopyCmd)
	rootCmd.AddCommand(deprecated.GenerateCmd)
	rootCmd.AddCommand(deprecated.InitCmd)
	rootCmd.AddCommand(deprecated.RunCmd)
}
