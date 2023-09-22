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

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cogment/cogment/status"
	"github.com/cogment/cogment/utils/constants"
)

const (
	serviceEndpointKey = "endpoint"
	serviceTypeKey     = "type"
)

var statusViper = viper.New()

var statusCmd = &cobra.Command{
	Use:          "status",
	Short:        "Request status from Cogment services",
	Args:         cobra.MinimumNArgs(0),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		verboseLevel := statusViper.GetInt(constants.VerboseKey)
		dirEndpoint := statusViper.GetString(constants.DirectoryEndpointKey)
		dirToken := statusViper.GetString(constants.DirectoryAuthTokenKey)
		serviceEndpoint := statusViper.GetString(serviceEndpointKey)
		serviceType := statusViper.GetString(serviceTypeKey)

		return status.Run(dirEndpoint, dirToken, serviceEndpoint, serviceType, verboseLevel, args)
	},
}

func init() {
	statusCmd.Flags().CountP(
		constants.VerboseKey,
		constants.VerboseShortKey,
		constants.VerboseDesc)

	statusViper.SetDefault(
		constants.DirectoryEndpointKey,
		fmt.Sprintf("grpc://localhost:%d", constants.DefaultDirectoryPort),
	)
	_ = statusViper.BindEnv(constants.DirectoryEndpointKey, constants.DirectoryEndpointEnv)
	statusCmd.Flags().String(
		constants.DirectoryEndpointKey,
		statusViper.GetString(constants.DirectoryEndpointKey),
		constants.DirectoryEndpointDesc,
	)

	statusViper.SetDefault(constants.DirectoryAuthTokenKey, "")
	_ = statusViper.BindEnv(constants.DirectoryAuthTokenKey, constants.DirectoryAuthTokenEnv)
	statusCmd.Flags().String(
		constants.DirectoryAuthTokenKey,
		statusViper.GetString(constants.DirectoryAuthTokenKey),
		constants.DirectoryAuthTokenDesc,
	)

	statusViper.SetDefault(serviceEndpointKey, "cogment://discover")
	statusCmd.Flags().String(
		serviceEndpointKey,
		statusViper.GetString(serviceEndpointKey),
		"Endpoint of the service from which to request statuses",
	)

	statusViper.SetDefault(serviceTypeKey, "")
	statusCmd.Flags().String(
		serviceTypeKey,
		statusViper.GetString(serviceTypeKey),
		fmt.Sprintf("The type of service to directly request status (%s)", constants.ServiceTypeAllDesc),
	)

	// Don't sort alphabetically, keep insertion order
	statusCmd.Flags().SortFlags = false

	// Bind "cobra" flags defined in the CLI with viper
	_ = statusViper.BindPFlags(statusCmd.Flags())
}
