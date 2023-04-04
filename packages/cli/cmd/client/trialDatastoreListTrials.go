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
	"context"
	"fmt"
	"os"

	trialDatastoreClient "github.com/cogment/cogment/clients/trialDatastore"
	"github.com/cogment/cogment/utils"
	"github.com/olekukonko/tablewriter"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// datastoreListTrialsViper represents the configuration of the `cogment client trial_datastore list_trials` command
var datastoreListTrialsViper = viper.New()

const (
	datastoreListTrialsCountKey      = "count"
	datastoreListTrialsFromKey       = "from"
	datastoreListTrialsPropertiesKey = "properties"
)

func init() {
	datastoreListTrialsViper.SetDefault(
		datastoreListTrialsCountKey,
		10,
	)

	datastoreListTrialsCmd.Flags().Uint(
		datastoreListTrialsCountKey,
		datastoreListTrialsViper.GetUint(datastoreListTrialsCountKey),
		"Maximum number of trials to retrieve",
	)

	datastoreListTrialsViper.SetDefault(datastoreListTrialsPropertiesKey, "")
	datastoreListTrialsCmd.Flags().String(
		datastoreListTrialsPropertiesKey,
		datastoreListTrialsViper.GetString(datastoreListTrialsPropertiesKey),
		"Desired trial properties (in the form 'name=value,name=value')",
	)

	datastoreListTrialsViper.SetDefault(
		datastoreListTrialsFromKey,
		"",
	)

	datastoreListTrialsCmd.Flags().String(
		datastoreListTrialsFromKey,
		datastoreListTrialsViper.GetString(datastoreListTrialsFromKey),
		"Handle defining the first trial to retrieve (use `next trial handle` retrieved from a previous call)",
	)

	// Don't sort alphabetically, keep insertion order
	datastoreListTrialsCmd.Flags().SortFlags = false

	// Bind "cobra" flags defined in the CLI with viper
	_ = datastoreListTrialsViper.BindPFlags(datastoreListTrialsCmd.Flags())
}

// datastoreListTrialsCmd represents the `cogment client trial_datastore list_trials` command
var datastoreListTrialsCmd = &cobra.Command{
	Use:     "list_trials",
	Aliases: []string{"list"},
	Short:   "List the stored trials",
	Args:    cobra.NoArgs,
	RunE: func(_cmd *cobra.Command, _args []string) error {
		consoleOutputFormat, err := retrieveConsoleOutputFormat()
		if err != nil {
			return err
		}

		trialsCount := datastoreListTrialsViper.GetUint(datastoreListTrialsCountKey)
		if trialsCount == 0 {
			return fmt.Errorf(
				"invalid argument \"--%s\" specified, expected a strictly positive number",
				datastoreListTrialsCountKey,
			)
		}

		properties, err := utils.ParseProperties(datastoreListTrialsViper.GetString(datastoreListTrialsPropertiesKey))
		if err != nil {
			return fmt.Errorf("invalid argument \"--%s\" specified, %w", datastoreListTrialsPropertiesKey, err)
		}

		client, err := trialDatastoreClient.CreateClientWithInsecureEndpoint(datastoreViper.GetString(datastoreEndpointKey))
		if err != nil {
			return err
		}

		fromHandle := datastoreListTrialsViper.GetString(datastoreListTrialsFromKey)

		ctx, cancel := context.WithTimeout(context.Background(), clientViper.GetDuration(clientTimeoutKey))
		defer cancel()
		rep, err := client.ListTrials(
			ctx,
			trialsCount,
			fromHandle,
			properties,
		)
		if err != nil {
			if err == context.DeadlineExceeded {
				return fmt.Errorf("timeout (%v) exceeded", clientViper.GetDuration(clientTimeoutKey))
			}
			return err
		}

		switch consoleOutputFormat {
		case text:
			table := tablewriter.NewWriter(os.Stdout)
			table.SetBorder(false)
			table.SetFooterAlignment(tablewriter.ALIGN_RIGHT)
			table.SetHeader([]string{
				"trial id",
				"user id",
				"state",
				"samples",
				"actors",
				"properties",
			})
			for _, trialInfo := range rep.TrialInfos {
				table.Append([]string{
					trialInfo.TrialId,
					trialInfo.UserId,
					trialInfo.LastState.String(),
					fmt.Sprintf("%d", trialInfo.SamplesCount),
					fmt.Sprintf("%d", len(trialInfo.Params.Actors)),
					utils.FormatProperties(trialInfo.Params.Properties),
				})
			}
			var caption string
			if fromHandle == "" {
				caption = fmt.Sprintf(
					"%d trials retrieved",
					len(rep.TrialInfos),
				)
			} else {
				caption = fmt.Sprintf(
					"%d trials retrieved from <%s>",
					len(rep.TrialInfos),
					fromHandle,
				)
			}
			caption += fmt.Sprintf(
				", next trial handle is <%s>",
				rep.NextTrialHandle,
			)
			table.SetCaption(true, caption)

			table.Render()
		case json:
			err := renderJSONFromProto(rep)
			if err != nil {
				return err
			}
		}
		return nil
	},
}
