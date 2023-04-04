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
	"path/filepath"

	trialDatastoreClient "github.com/cogment/cogment/clients/trialDatastore"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// datastoreImportViper represents the configuration of the `cogment client trial_datastore import` command
var datastoreImportViper = viper.New()

const (
	datastoreImportFileKey   = "file"
	datastoreImportPrefixKey = "prefix"
	datastoreImportUserIDKey = "user_id"
)

type importedTrialInfos struct {
	TrialID      string `json:"trial_id"`
	SamplesCount int    `json:"samples_count"`
}

type datastoreImportOutput struct {
	Message      string               `json:"message"`
	SamplesCount int                  `json:"samples_count"`
	Trials       []importedTrialInfos `json:"trials"`
	FilePath     string               `json:"filepath"`
}

func init() {
	datastoreImportViper.SetDefault(
		datastoreImportFileKey,
		"",
	)

	datastoreImportCmd.Flags().String(
		datastoreImportFileKey,
		datastoreImportViper.GetString(datastoreImportFileKey),
		"Trial samples input file path, if not defined, will read from stdin",
	)

	datastoreImportViper.SetDefault(
		datastoreImportPrefixKey,
		"",
	)

	datastoreImportCmd.Flags().String(
		datastoreImportPrefixKey,
		datastoreImportViper.GetString(datastoreImportPrefixKey),
		"Prefix applied to the IDs of imported trials",
	)

	datastoreImportViper.SetDefault(
		datastoreImportUserIDKey,
		"cogment CLI",
	)

	datastoreImportCmd.Flags().String(
		datastoreImportUserIDKey,
		datastoreImportViper.GetString(datastoreImportUserIDKey),
		"User ID",
	)

	// Don't sort alphabetically, keep insertion order
	datastoreImportCmd.Flags().SortFlags = false

	// Bind "cobra" flags defined in the CLI with viper
	_ = datastoreImportViper.BindPFlags(datastoreImportCmd.Flags())
}

// datastoreImportCmd represents the `cogment client trial_datastore export` command
var datastoreImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Import trials and their samples",
	Args:  cobra.NoArgs,
	RunE: func(_cmd *cobra.Command, args []string) error {
		consoleOutputFormat, err := retrieveConsoleOutputFormat()
		if err != nil {
			return err
		}

		client, err := trialDatastoreClient.CreateClientWithInsecureEndpoint(datastoreViper.GetString(datastoreEndpointKey))
		if err != nil {
			return err
		}

		filePath := datastoreImportViper.GetString(datastoreImportFileKey)
		filePath, err = filepath.Abs(filePath)
		if err != nil {
			return err
		}

		prefix := datastoreImportViper.GetString(datastoreImportPrefixKey)
		userID := datastoreImportViper.GetString(datastoreImportUserIDKey)

		ctx, cancel := context.WithTimeout(context.Background(), clientViper.GetDuration(clientTimeoutKey))
		defer cancel()

		var sampleCounts map[string]int
		if filePath == "" {
			filePath = "stdin"
			sampleCounts, err = client.ImportTrials(
				ctx,
				userID,
				prefix,
				os.Stdin,
			)
			if err != nil {
				return err
			}
		} else {
			f, err := os.Open(filePath)
			if err != nil {
				return err
			}
			defer f.Close()
			sampleCounts, err = client.ImportTrials(
				ctx,
				userID,
				prefix,
				f,
			)
			if err != nil {
				return err
			}
		}

		output := datastoreImportOutput{
			SamplesCount: 0,
			Trials:       []importedTrialInfos{},
			FilePath:     filePath,
		}
		for trialID, sampleCount := range sampleCounts {
			output.SamplesCount += sampleCount
			output.Trials = append(output.Trials, importedTrialInfos{
				TrialID:      trialID,
				SamplesCount: sampleCount,
			})
		}
		output.Message = fmt.Sprintf(
			"%d trials imported (%d samples) from %q",
			len(output.Trials),
			output.SamplesCount,
			output.FilePath,
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
