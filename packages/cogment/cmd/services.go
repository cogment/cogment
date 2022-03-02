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

package cmd

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// servicesViper represents the configuration of the orchestrator command
var servicesViper = viper.New()

var servicesLogLevelKey = "log_level"
var servicesLogFileKey = "log_file"

var expectedLogLevels []string

// servicesCmd represents the services command
var servicesCmd = &cobra.Command{
	Use:   "services",
	Short: "Run cogment services",
	Args:  cobra.NoArgs,
}

func init() {
	rootCmd.AddCommand(servicesCmd)

	expectedLogLevels = make([]string, 0)
	for _, level := range logrus.AllLevels {
		expectedLogLevels = append(expectedLogLevels, level.String())
	}

	servicesViper.SetDefault(servicesLogLevelKey, logrus.InfoLevel.String())
	_ = servicesViper.BindEnv(servicesLogLevelKey, "COGMENT_LOG_LEVEL")
	servicesCmd.PersistentFlags().String(
		servicesLogLevelKey,
		servicesViper.GetString(servicesLogLevelKey),
		fmt.Sprintf("Set minimum logging level as one of %v", expectedLogLevels),
	)

	_ = servicesViper.BindEnv(servicesLogFileKey, "COGMENT_LOG_FILE")
	servicesCmd.PersistentFlags().String(
		servicesLogFileKey,
		servicesViper.GetString(servicesLogFileKey),
		"Set base file for daily log output (env variable)",
	)

	// Don't sort alphabetically, keep insertion order
	servicesCmd.PersistentFlags().SortFlags = false

	// Bind "cobra" flags defined in the CLI with viper
	_ = servicesViper.BindPFlags(servicesCmd.PersistentFlags())
}
