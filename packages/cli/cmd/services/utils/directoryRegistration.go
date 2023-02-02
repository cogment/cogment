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

package utils

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	servicesUtils "github.com/cogment/cogment/services/utils"
	"github.com/cogment/cogment/utils"
)

var directoryEndpointKey = "directory_endpoint"
var directoryAuthTokenKey = "directory_authentication_token"
var directoryRegistrationHostKey = "directory_registration_host"
var directoryRegistrationPropertiesKey = "directory_registration_properties"

func PopulateDirectoryRegistrationOptionsFlags(
	serviceName string,
	cmd *cobra.Command,
	viper *viper.Viper,
	defaultValues servicesUtils.DirectoryRegistrationOptions) {

	viper.SetDefault(directoryEndpointKey, defaultValues.DirectoryEndpoint)
	_ = viper.BindEnv(directoryEndpointKey, "COGMENT_DIRECTORY_ENDPOINT")
	cmd.Flags().String(
		directoryEndpointKey,
		viper.GetString(directoryEndpointKey),
		"Directory service gRPC endpoint",
	)

	viper.SetDefault(directoryAuthTokenKey, defaultValues.DirectoryAuthToken)
	_ = viper.BindEnv(directoryAuthTokenKey, "COGMENT_DIRECTORY_AUTHENTICATION_TOKEN")
	cmd.Flags().String(
		directoryAuthTokenKey,
		viper.GetString(directoryAuthTokenKey),
		"Authentication token for directory services",
	)

	viper.SetDefault(directoryRegistrationHostKey, defaultValues.DirectoryRegistrationHost)
	_ = viper.BindEnv(
		directoryRegistrationHostKey,
		fmt.Sprintf("COGMENT_%s_DIRECTORY_REGISTRATION_HOST", serviceName),
	)
	cmd.Flags().String(
		directoryRegistrationHostKey,
		viper.GetString(directoryRegistrationHostKey),
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
		directoryUsageRegistrationPropertiesEnvValue, err := utils.ParseStringToString(
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
		directoryRegistrationPropertiesKey,
		directoryUsageRegistrationPropertiesDefaultValue,
	)
	cmd.Flags().StringToString(
		directoryRegistrationPropertiesKey,
		viper.GetStringMapString(directoryRegistrationPropertiesKey),
		fmt.Sprintf("Properties to register to the Directory for the %s", serviceName),
	)
}

func GetDirectoryRegistrationOptions(viper *viper.Viper) servicesUtils.DirectoryRegistrationOptions {
	return servicesUtils.DirectoryRegistrationOptions{
		DirectoryEndpoint:               viper.GetString(directoryEndpointKey),
		DirectoryAuthToken:              viper.GetString(directoryAuthTokenKey),
		DirectoryRegistrationHost:       viper.GetString(directoryRegistrationHostKey),
		DirectoryRegistrationProperties: viper.GetStringMapString(directoryRegistrationPropertiesKey),
	}
}
