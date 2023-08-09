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

package services

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// servicesViper represents the configuration of the services command
var servicesViper = viper.New()

var servicesLogLevelKey = "log_level"
var servicesLogFileKey = "log_file"
var servicesLogFormatKey = "log_format"

// ServicesCmd represents the services command
var ServicesCmd = &cobra.Command{
	Use:   "services",
	Short: "Run cogment services",
	Args:  cobra.NoArgs,
}

func init() {
	servicesViper.SetDefault(servicesLogLevelKey, logrus.InfoLevel.String())
	_ = servicesViper.BindEnv(servicesLogLevelKey, "COGMENT_LOG_LEVEL")
	ServicesCmd.PersistentFlags().String(
		servicesLogLevelKey,
		servicesViper.GetString(servicesLogLevelKey),
		fmt.Sprintf("Minimum logging level as one of %v", expectedLogLevels),
	)

	_ = servicesViper.BindEnv(servicesLogFileKey, "COGMENT_LOG_FILE")
	ServicesCmd.PersistentFlags().String(
		servicesLogFileKey,
		servicesViper.GetString(servicesLogFileKey),
		"Log file output",
	)

	_ = servicesViper.BindEnv(servicesLogFormatKey, "COGMENT_LOG_FORMAT")
	ServicesCmd.PersistentFlags().String(
		servicesLogFormatKey,
		servicesViper.GetString(servicesLogFormatKey),
		fmt.Sprintf(
			"Log format as one of %v, default is %q, when a log file is specified it is %q",
			expectedLogFormats, text, json,
		),
	)

	// Don't sort alphabetically, keep insertion order
	ServicesCmd.PersistentFlags().SortFlags = false

	// Bind "cobra" flags defined in the CLI with viper
	_ = servicesViper.BindPFlags(ServicesCmd.PersistentFlags())

	// Add the service subcommands
	ServicesCmd.AddCommand(registryCmd)
	ServicesCmd.AddCommand(orchestratorCmd)
	ServicesCmd.AddCommand(datastoreCmd)
	ServicesCmd.AddCommand(directoryCmd)
	ServicesCmd.AddCommand(proxyCmd)
}
