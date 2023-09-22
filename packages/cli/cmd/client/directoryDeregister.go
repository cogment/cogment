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

	cogmentAPI "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/cogment/cogment/utils/constants"
	"github.com/cogment/cogment/utils/endpoint"

	directoryClient "github.com/cogment/cogment/clients/directory"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var directoryDeregisterViper = viper.New()

func init() {
	directoryDeregisterViper.SetDefault(directoryServiceIDKey, 0)
	directoryDeregisterCmd.Flags().Uint64(
		directoryServiceIDKey,
		directoryDeregisterViper.GetUint64(directoryServiceIDKey),
		"The ID of the service to be removed from the Directory",
	)

	directoryDeregisterViper.SetDefault(directoryServiceSecretKey, "")
	directoryDeregisterCmd.Flags().String(
		directoryServiceSecretKey,
		directoryDeregisterViper.GetString(directoryServiceSecretKey),
		"The secret (provided on registration) of the service to be removed from the Directory",
	)

	directoryDeregisterCmd.Flags().SortFlags = false
	_ = directoryDeregisterViper.BindPFlags(directoryDeregisterCmd.Flags())
}

var directoryDeregisterCmd = &cobra.Command{
	Use:   "deregister",
	Short: "Remove a service from the Directory",
	Args:  cobra.NoArgs,
	RunE: func(_cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), clientViper.GetDuration(clientTimeoutKey))
		defer cancel() // This causes grpc "cancel" info output, but Go lint forces us to call it!

		directoryEndpoint, err := endpoint.Parse(directoryViper.GetString(constants.DirectoryEndpointKey))
		if err != nil {
			return err
		}

		client, err := directoryClient.CreateClient(
			ctx,
			directoryEndpoint,
			directoryViper.GetString(constants.DirectoryAuthTokenKey),
		)
		if err != nil {
			return err
		}

		request := cogmentAPI.DeregisterRequest{}
		request.ServiceId = directoryDeregisterViper.GetUint64(directoryServiceIDKey)
		request.Secret = directoryDeregisterViper.GetString(directoryServiceSecretKey)

		return client.Deregister(&request)
	},
}
