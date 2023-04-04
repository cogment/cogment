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
	"github.com/cogment/cogment/utils"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// datastoreExportViper represents the configuration of the `cogment client trial_datastore export` command
var datastoreExportViper = viper.New()

const datastoreExportFileKey = "file"

type datastoreExportOutput struct {
	Bytes    int      `json:"bytes"`
	Message  string   `json:"message"`
	TrialIds []string `json:"trial_ids"`
	FilePath string   `json:"filepath"`
}

func init() {
	datastoreExportViper.SetDefault(
		datastoreExportFileKey,
		"",
	)

	datastoreExportCmd.Flags().String(
		datastoreExportFileKey,
		datastoreExportViper.GetString(datastoreExportFileKey),
		"Trial samples output file path, if not defined, will write to stdout",
	)

	// Don't sort alphabetically, keep insertion order
	datastoreExportCmd.Flags().SortFlags = false

	// Bind "cobra" flags defined in the CLI with viper
	_ = datastoreExportViper.BindPFlags(datastoreExportCmd.Flags())
}

// datastoreExportCmd represents the `cogment client trial_datastore export` command
var datastoreExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export all the samples of the desired trials",
	Args:  cobra.MinimumNArgs(1),
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
		filePath := datastoreExportViper.GetString(datastoreExportFileKey)
		filePath, err = filepath.Abs(filePath)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), clientViper.GetDuration(clientTimeoutKey))
		defer cancel()

		if filePath == "" {
			_, err = client.ExportTrials(
				ctx,
				trialIds,
				os.Stdout,
			)
			if err != nil {
				return err
			}
			return nil
		}

		f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, 0755)
		if err != nil {
			return err
		}
		defer f.Close()
		bytesWritten, err := client.ExportTrials(
			ctx,
			trialIds,
			f,
		)
		if err != nil {
			return err
		}
		message := fmt.Sprintf(
			"%d trials exported to %q (%s written)",
			len(trialIds),
			filePath,
			utils.FormatBytes(bytesWritten),
		)
		switch consoleOutputFormat {
		case text:
			fmt.Println(message)
		case json:
			err := renderJSON(&datastoreExportOutput{
				Bytes:    bytesWritten,
				TrialIds: trialIds,
				FilePath: filePath,
				Message:  message,
			})
			if err != nil {
				return err
			}
		}
		return nil
	},
}
