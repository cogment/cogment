// Copyright 2022 AI Redefined Inc. <dev+cogment@ai-r.com>
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

package client

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// clientViper represents the configuration of the `cogment client` command
var clientViper = viper.New()

const (
	clientConsoleOutputFormatKey = "console_output"
	clientTimeoutKey             = "timeout"
	defaultClientTimeout         = 30 * time.Second
)

// ClientCmd represents the `cogment client` command
var ClientCmd = &cobra.Command{
	Use:   "client",
	Short: "Run cogment client",
	Args:  cobra.NoArgs,
}

func init() {
	clientViper.SetDefault(clientConsoleOutputFormatKey, string(text))
	_ = clientViper.BindEnv(clientConsoleOutputFormatKey, "COGMENT_CLIENT_CONSOLE_OUTPUT")
	ClientCmd.PersistentFlags().String(
		clientConsoleOutputFormatKey,
		clientViper.GetString(clientConsoleOutputFormatKey),
		fmt.Sprintf(
			"Set console output format as one of %v",
			expectedOutputFormats,
		),
	)

	clientViper.SetDefault(clientTimeoutKey, defaultClientTimeout)
	_ = clientViper.BindEnv(clientTimeoutKey, "COGMENT_CLIENT_TIMEOUT")
	ClientCmd.PersistentFlags().Duration(
		clientTimeoutKey,
		clientViper.GetDuration(clientTimeoutKey),
		"Timeout for the operation",
	)

	// Don't sort alphabetically, keep insertion order
	ClientCmd.PersistentFlags().SortFlags = false

	// Bind "cobra" flags defined in the CLI with viper
	_ = clientViper.BindPFlags(ClientCmd.PersistentFlags())

	// Add the client subcommands
	ClientCmd.AddCommand(directoryCmd)
	ClientCmd.AddCommand(datastoreCmd)
}
