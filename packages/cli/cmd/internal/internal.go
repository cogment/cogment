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

package internal

import (
	"github.com/cogment/cogment/services/proxy"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// InternalCmd represents the `cogment internal` command
var InternalCmd = &cobra.Command{
	Use:    "internal",
	Short:  "Run cogment internal commands",
	Args:   cobra.NoArgs,
	Hidden: true,
}

// generateProxyAPISpecViper represents the configuration of the `cogment generate_proxy_api_spec` command
var generateProxyAPISpecViper = viper.New()

const generateProxyAPISpecOutputKey = "output"

// generateProxyAPISpecCmd represents the `cogment generate_proxy_api_spec` command
var generateProxyAPISpecCmd = &cobra.Command{
	Use:   "generate_proxy_api_spec",
	Short: "Generate the web proxy http openapi spec",
	Args:  cobra.NoArgs,
	RunE: func(_cmd *cobra.Command, _args []string) error {
		output := generateProxyAPISpecViper.GetString(generateProxyAPISpecOutputKey)

		return proxy.GenerateOpenAPISpec(output)
	},
}

func init() {
	generateProxyAPISpecViper.SetDefault(generateProxyAPISpecOutputKey, "./web-proxy-openapi.json")

	generateProxyAPISpecCmd.PersistentFlags().String(
		generateProxyAPISpecOutputKey,
		generateProxyAPISpecViper.GetString(generateProxyAPISpecOutputKey),
		"Path to the json output file",
	)

	// Don't sort alphabetically, keep insertion order
	generateProxyAPISpecCmd.PersistentFlags().SortFlags = false

	// Bind "cobra" flags defined in the CLI with viper
	_ = generateProxyAPISpecViper.BindPFlags(generateProxyAPISpecCmd.PersistentFlags())

	// Add the client subcommands
	InternalCmd.AddCommand(generateProxyAPISpecCmd)
}
