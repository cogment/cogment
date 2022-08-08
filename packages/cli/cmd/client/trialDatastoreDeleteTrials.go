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
	"context"
	"fmt"

	trialDatastoreClient "github.com/cogment/cogment/clients/trialDatastore"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// datastoreDeleteTrialsViper represents the configuration of the `cogment client trial_datastore delete_trials` command
var datastoreDeleteTrialsViper = viper.New()

type datastoreDeleteTrialsOutput struct {
	Message  string   `json:"message"`
	TrialIds []string `json:"trial_ids"`
}

func init() {
	// Don't sort alphabetically, keep insertion order
	datastoreDeleteTrialsCmd.Flags().SortFlags = false

	// Bind "cobra" flags defined in the CLI with viper
	_ = datastoreDeleteTrialsViper.BindPFlags(datastoreDeleteTrialsCmd.Flags())
}

// datastoreDeleteTrialsCmd represents the `cogment client trial_datastore delete_trials` command
var datastoreDeleteTrialsCmd = &cobra.Command{
	Use:     "delete_trials",
	Aliases: []string{"delete"},
	Short:   "Delete stored trials",
	Args:    cobra.MinimumNArgs(1),
	RunE: func(_cmd *cobra.Command, args []string) error {
		consoleOutputFormat, err := retrieveConsoleOutputFormat()
		if err != nil {
			return err
		}

		client, err := trialDatastoreClient.CreateClientWithInsecureEndpoint(datastoreViper.GetString(datastoreEndpointKey))
		if err != nil {
			return err
		}

		trialIds := args

		ctx, cancel := context.WithTimeout(context.Background(), clientViper.GetDuration(clientTimeoutKey))
		defer cancel()
		err = client.DeleteTrials(
			ctx,
			trialIds,
		)
		if err != nil {
			if err == context.DeadlineExceeded {
				return fmt.Errorf("timeout [%v] exceeded", clientViper.GetDuration(clientTimeoutKey))
			}
			return err
		}

		output := datastoreDeleteTrialsOutput{
			TrialIds: trialIds,
		}
		output.Message = fmt.Sprintf(
			"%d trials no longer stored",
			len(output.TrialIds),
		)

		switch consoleOutputFormat {
		case text:
			fmt.Println(output.Message)
		case json:
			err := renderJSON(output)
			if err != nil {
				return err
			}
		}
		return nil
	},
}
