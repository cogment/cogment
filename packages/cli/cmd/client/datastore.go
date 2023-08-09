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

package client

import (
	"fmt"

	trialDatastoreService "github.com/cogment/cogment/services/datastore"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// datastoreViper represents the configuration of the `cogment client trial_datastore` command
var datastoreViper = viper.New()

const datastoreEndpointKey = "endpoint"

func init() {
	datastoreViper.SetDefault(
		datastoreEndpointKey,
		fmt.Sprintf("grpc://localhost:%d", trialDatastoreService.DefaultOptions.Port),
	)
	_ = datastoreViper.BindEnv(datastoreEndpointKey, "COGMENT_TRIAL_DATASTORE_ENDPOINT")
	datastoreCmd.PersistentFlags().String(
		datastoreEndpointKey,
		datastoreViper.GetString(datastoreEndpointKey),
		"The datastore endpoint URL",
	)

	// Don't sort alphabetically, keep insertion order
	datastoreCmd.PersistentFlags().SortFlags = false

	// Bind "cobra" flags defined in the CLI with viper
	_ = datastoreViper.BindPFlags(datastoreCmd.PersistentFlags())

	// Add the subcommands
	datastoreCmd.AddCommand(datastoreListTrialsCmd)
	datastoreCmd.AddCommand(datastoreDeleteTrialsCmd)
	datastoreCmd.AddCommand(datastoreExportCmd)
	datastoreCmd.AddCommand(datastoreImportCmd)
}

// datastoreCmd represents the `cogment client trial_datastore` command
var datastoreCmd = &cobra.Command{
	Use:     "trial_datastore",
	Aliases: []string{"datastore"},
	Short:   "Run trial datastore client",
	Args:    cobra.NoArgs,
}
