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
	"errors"
	"os"

	directoryClient "github.com/cogment/cogment/clients/directory"
	"github.com/cogment/cogment/utils/endpoint"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var directoryWaitForReadyViper = viper.New()

func init() {
	directoryWaitForReadyCmd.Flags().SortFlags = false
	_ = directoryWaitForReadyViper.BindPFlags(directoryWaitForReadyCmd.Flags())
}

var directoryWaitForReadyCmd = &cobra.Command{
	Use:   "WaitForReady",
	Short: "Wait for the directory to be ready and return",
	Args:  cobra.NoArgs,
	RunE: func(_cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), clientViper.GetDuration(clientTimeoutKey))
		defer cancel()

		directoryEndpoint, err := endpoint.Parse(directoryViper.GetString(directoryEndpointKey))
		if err != nil {
			return err
		}

		client, err := directoryClient.CreateClient(
			ctx,
			directoryEndpoint,
			directoryViper.GetString(directoryAuthTokenKey),
		)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				os.Exit(1)
			}
			return err
		}

		err = client.WaitForReady()
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				os.Exit(1)
			}
			return err
		}

		return nil
	},
}
