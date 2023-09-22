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

	"github.com/cogment/cogment/utils/constants"
	"github.com/cogment/cogment/version"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	// Options
	directoryRegisterHostKey      = "host"
	directoryRegisterPortKey      = "port"
	directoryRegisterProtocolKey  = "protocol"
	directoryRegisterSslKey       = "ssl_required"
	directoryServiceTypeKey       = "type"
	directoryServicePropertiesKey = "properties"
	directoryServiceIDKey         = "service_id"
	directoryServiceSecretKey     = "secret"
	directoryServicePermanent     = "permanent"
)

var additionalRegistrationProperties = map[string]string{
	"__registration_source": "Cogment-Command_Line",
	"__version":             version.Version,
}
var directoryViper = viper.New()

func init() {
	directoryViper.SetDefault(
		constants.DirectoryEndpointKey,
		fmt.Sprintf("grpc://localhost:%d", constants.DefaultDirectoryPort),
	)
	_ = directoryViper.BindEnv(constants.DirectoryEndpointKey, constants.DirectoryEndpointEnv)
	directoryCmd.PersistentFlags().String(
		constants.DirectoryEndpointKey,
		directoryViper.GetString(constants.DirectoryEndpointKey),
		constants.DirectoryEndpointDesc,
	)
	directoryCmd.PersistentFlags().String(
		"endpoint",
		directoryViper.GetString(constants.DirectoryEndpointKey),
		"",
	)
	_ = directoryCmd.PersistentFlags().MarkDeprecated(
		"endpoint",
		fmt.Sprintf("please use --%s instead", constants.DirectoryEndpointKey),
	)

	directoryViper.SetDefault(constants.DirectoryAuthTokenKey, "")
	_ = directoryViper.BindEnv(constants.DirectoryAuthTokenKey, constants.DirectoryAuthTokenEnv)
	directoryCmd.PersistentFlags().String(
		constants.DirectoryAuthTokenKey,
		directoryViper.GetString(constants.DirectoryAuthTokenKey),
		constants.DirectoryAuthTokenDesc,
	)
	directoryCmd.PersistentFlags().String(
		"auth_token",
		directoryViper.GetString(constants.DirectoryAuthTokenKey),
		"",
	)
	_ = directoryCmd.PersistentFlags().MarkDeprecated(
		"auth_token",
		fmt.Sprintf("please use --%s instead", constants.DirectoryAuthTokenKey),
	)

	directoryCmd.PersistentFlags().SortFlags = false
	_ = directoryViper.BindPFlags(directoryCmd.PersistentFlags())

	directoryCmd.AddCommand(directoryRegisterCmd)
	directoryCmd.AddCommand(directoryDeregisterCmd)
	directoryCmd.AddCommand(directoryInquireCmd)
	directoryCmd.AddCommand(directoryWaitForReadyCmd)
}

var directoryCmd = &cobra.Command{
	Use:   "directory",
	Short: "Run directory client",
	Args:  cobra.NoArgs,
}
