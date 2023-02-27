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

	cogmentAPI "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/cogment/cogment/utils"

	directoryClient "github.com/cogment/cogment/clients/directory"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var directoryRegisterViper = viper.New()

func init() {
	directoryRegisterViper.SetDefault(directoryRegisterHostKey, "")
	directoryRegisterCmd.Flags().String(
		directoryRegisterHostKey,
		directoryRegisterViper.GetString(directoryRegisterHostKey),
		"Host of the service to be registered in the Directory",
	)

	directoryRegisterViper.SetDefault(directoryRegisterPortKey, 0)
	directoryRegisterCmd.Flags().Uint32(
		directoryRegisterPortKey,
		directoryRegisterViper.GetUint32(directoryRegisterPortKey),
		"TCP port of the service to be registered in the Directory",
	)

	directoryRegisterViper.SetDefault(directoryRegisterProtocolKey, "grpc")
	directoryRegisterCmd.Flags().String(
		directoryRegisterProtocolKey,
		directoryRegisterViper.GetString(directoryRegisterProtocolKey),
		"The communication protocol of the service to be registered in the Directory ('grpc' or 'cogment')",
	)

	directoryRegisterViper.SetDefault(directoryRegisterSslKey, false)
	directoryRegisterCmd.Flags().Bool(
		directoryRegisterSslKey,
		directoryRegisterViper.GetBool(directoryRegisterSslKey),
		"Whether the service to register in the Directory requires SSL communication)",
	)

	directoryRegisterViper.SetDefault(directoryServiceTypeKey, "")
	directoryRegisterCmd.Flags().String(
		directoryServiceTypeKey,
		directoryRegisterViper.GetString(directoryServiceTypeKey),
		"Type of service to be registered in the Directory (e.g. 'actor', 'environment')",
	)

	directoryRegisterViper.SetDefault(directoryServicePropertiesKey, "")
	directoryRegisterCmd.Flags().String(
		directoryServicePropertiesKey,
		directoryRegisterViper.GetString(directoryServicePropertiesKey),
		"The properties of the service to be registered in the Directory (in the form 'name=value,name=value')",
	)

	directoryRegisterViper.SetDefault(directoryServicePermanent, false)
	directoryRegisterCmd.Flags().Bool(
		directoryServicePermanent,
		directoryRegisterViper.GetBool(directoryServicePermanent),
		"Whether the service is permanent in the directory and will not be subjected to health checks",
	)

	directoryRegisterCmd.Flags().SortFlags = false
	_ = directoryRegisterViper.BindPFlags(directoryRegisterCmd.Flags())
}

var directoryRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a service in the Directory",
	Args:  cobra.NoArgs,
	RunE: func(_cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), clientViper.GetDuration(clientTimeoutKey))
		defer cancel() // This causes grpc "cancel" info output, but Go lint forces us to call it!

		client, err := directoryClient.CreateClient(
			ctx,
			directoryViper.GetString(directoryEndpointKey),
			directoryViper.GetString(directoryAuthTokenKey),
		)
		if err != nil {
			return err
		}

		request := cogmentAPI.RegisterRequest{
			Endpoint: &cogmentAPI.ServiceEndpoint{},
			Details:  &cogmentAPI.ServiceDetails{},
		}
		request.Endpoint.Host = directoryRegisterViper.GetString(directoryRegisterHostKey)
		request.Endpoint.Port = directoryRegisterViper.GetUint32(directoryRegisterPortKey)

		ssl := directoryRegisterViper.GetBool(directoryRegisterSslKey)
		protocolStr := directoryRegisterViper.GetString(directoryRegisterProtocolKey)
		protocol, err := strToAPIProtocol(protocolStr, ssl)
		if err != nil {
			return err
		}
		request.Endpoint.Protocol = protocol

		typeStr := directoryRegisterViper.GetString(directoryServiceTypeKey)
		serviceType, err := strToAPIServiceType(typeStr)
		if err != nil {
			return err
		}
		request.Details.Type = serviceType

		request.Permanent = directoryRegisterViper.GetBool(directoryServicePermanent)

		propertiesStr := directoryRegisterViper.GetString(directoryServicePropertiesKey)
		properties, err := utils.ParseProperties(propertiesStr)
		if err != nil {
			return err
		}
		request.Details.Properties = make(map[string]string)
		for name, value := range properties {
			request.Details.Properties[name] = value
		}
		for name, value := range additionalRegistrationProperties {
			request.Details.Properties[name] = value
		}

		id, secret, err := client.Register(&request)
		if err != nil {
			return fmt.Errorf("Failed to register service: %w", err)
		}

		fmt.Printf("Service ID [%d] \tSecret [%s]\n", id, secret)

		return nil
	},
}
