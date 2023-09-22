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

package utils

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cogment/cogment/clients/directory"
	"github.com/cogment/cogment/utils"
	"github.com/cogment/cogment/utils/constants"
	"github.com/cogment/cogment/utils/endpoint"
)

func PopulateDirectoryRegistrationOptionsFlags(
	serviceName string,
	cmd *cobra.Command,
	viper *viper.Viper,
	defaultValues directory.RegistrationOptions) {

	viper.SetDefault(constants.DirectoryEndpointKey, defaultValues.DirectoryEndpoint)
	_ = viper.BindEnv(constants.DirectoryEndpointKey, constants.DirectoryEndpointEnv)
	cmd.Flags().String(
		constants.DirectoryEndpointKey,
		viper.GetString(constants.DirectoryEndpointKey),
		constants.DirectoryEndpointDesc,
	)

	viper.SetDefault(constants.DirectoryAuthTokenKey, defaultValues.DirectoryAuthToken)
	_ = viper.BindEnv(constants.DirectoryAuthTokenKey, constants.DirectoryAuthTokenEnv)
	cmd.Flags().String(
		constants.DirectoryAuthTokenKey,
		viper.GetString(constants.DirectoryAuthTokenKey),
		constants.DirectoryAuthTokenDesc,
	)

	viper.SetDefault(constants.DirectoryRegistrationHostKey, defaultValues.DirectoryRegistrationHost)
	_ = viper.BindEnv(
		constants.DirectoryRegistrationHostKey,
		fmt.Sprintf("COGMENT_%s_DIRECTORY_REGISTRATION_HOST", serviceName),
	)
	cmd.Flags().String(
		constants.DirectoryRegistrationHostKey,
		viper.GetString(constants.DirectoryRegistrationHostKey),
		fmt.Sprintf("Host to register as the %s in the Directory (self discover host by default)", serviceName),
	)

	// To load in a map directly from a viper environment variable, we would need a JSON string, so instead
	// we bypass viper to read this value from the environment directly, then parse it manually to a map.
	// Viper recognizes different "map" string formats on the command line and the environment variable??!!!
	directoryUsageRegistrationPropertiesDefaultValue := defaultValues.DirectoryRegistrationProperties
	directoryUsageRegistrationPropertiesEnvValueStr, ok := os.LookupEnv(
		fmt.Sprintf("COGMENT_%s_DIRECTORY_REGISTRATION_PROPERTIES", serviceName),
	)
	if ok {
		directoryUsageRegistrationPropertiesEnvValue, err := utils.ParseProperties(
			directoryUsageRegistrationPropertiesEnvValueStr,
		)
		if err != nil {
			log.WithField(
				"value",
				directoryUsageRegistrationPropertiesEnvValueStr,
			).Warn("Invalid value format for directory registration properties in environment, ignoring it")
		} else {
			directoryUsageRegistrationPropertiesDefaultValue = directoryUsageRegistrationPropertiesEnvValue
		}
	}
	viper.SetDefault(
		constants.DirectoryRegistrationPropertiesKey,
		directoryUsageRegistrationPropertiesDefaultValue,
	)
	cmd.Flags().StringToString(
		constants.DirectoryRegistrationPropertiesKey,
		viper.GetStringMapString(constants.DirectoryRegistrationPropertiesKey),
		fmt.Sprintf("Properties to register to the Directory for the %s", serviceName),
	)
}

func GetDirectoryRegistrationOptions(viper *viper.Viper) (directory.RegistrationOptions, error) {
	directoryEndpointStr := viper.GetString(constants.DirectoryEndpointKey)

	directoryEndpoint := &endpoint.Endpoint{}
	if len(directoryEndpointStr) > 0 {
		var err error
		directoryEndpoint, err = endpoint.Parse(directoryEndpointStr)
		if err != nil {
			return directory.RegistrationOptions{},
				fmt.Errorf("invalid directory endpoint [%s]: %w", directoryEndpointStr, err)
		}
	}

	return directory.RegistrationOptions{
		DirectoryEndpoint:               directoryEndpoint,
		DirectoryAuthToken:              viper.GetString(constants.DirectoryAuthTokenKey),
		DirectoryRegistrationHost:       viper.GetString(constants.DirectoryRegistrationHostKey),
		DirectoryRegistrationProperties: viper.GetStringMapString(constants.DirectoryRegistrationPropertiesKey),
	}, nil
}
