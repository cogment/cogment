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

	cogmentAPI "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/cogment/cogment/utils"
	"github.com/cogment/cogment/utils/endpoint"

	directoryClient "github.com/cogment/cogment/clients/directory"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var directoryInquireViper = viper.New()

func init() {
	directoryInquireViper.SetDefault(directoryServiceIDKey, 0)
	directoryInquireCmd.Flags().Uint64(
		directoryServiceIDKey,
		directoryInquireViper.GetUint64(directoryServiceIDKey),
		"Numerical ID of the service to inquire from the Directory",
	)

	directoryInquireViper.SetDefault(directoryServiceTypeKey, "")
	directoryInquireCmd.Flags().String(
		directoryServiceTypeKey,
		directoryInquireViper.GetString(directoryServiceTypeKey),
		"Type of service to inquire from the Directory (actor, environment, "+
			"prehook, datalog, lifecycle, actservice, datastore, modelregistry or other)",
	)

	directoryInquireViper.SetDefault(directoryServicePropertiesKey, "")
	directoryInquireCmd.Flags().String(
		directoryServicePropertiesKey,
		directoryInquireViper.GetString(directoryServicePropertiesKey),
		"Properties of the service to inquire from the Directory (in the form 'name=value,name=value')",
	)

	directoryInquireCmd.Flags().SortFlags = false
	_ = directoryInquireViper.BindPFlags(directoryInquireCmd.Flags())
}

var directoryInquireCmd = &cobra.Command{
	Use:   "inquire",
	Short: "Find services in the Directory",
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
			return err
		}

		serviceID := directoryInquireViper.GetUint64(directoryServiceIDKey)
		typeStr := directoryInquireViper.GetString(directoryServiceTypeKey)
		propertiesStr := directoryInquireViper.GetString(directoryServicePropertiesKey)

		request := cogmentAPI.InquireRequest{}
		if serviceID != 0 {
			if len(typeStr) != 0 || len(propertiesStr) != 0 {
				return fmt.Errorf("Can only inquire with a service ID or type/properties, not both")
			}

			request.Inquiry = &cogmentAPI.InquireRequest_ServiceId{ServiceId: serviceID}

		} else {
			details := cogmentAPI.ServiceDetails{}

			serviceType, err := strToAPIServiceType(typeStr)
			if err != nil {
				return err
			}
			details.Type = serviceType

			properties, err := utils.ParseProperties(propertiesStr)
			if err != nil {
				return err
			}
			if len(properties) > 0 {
				details.Properties = make(map[string]string)
				for name, value := range properties {
					details.Properties[name] = value
				}
			}

			request.Inquiry = &cogmentAPI.InquireRequest_Details{Details: &details}
		}

		services, err := client.Inquire(&request)
		if err != nil {
			return fmt.Errorf("Failed to inquire service: %w", err)
		}

		fmt.Printf("[%d] Services found\n", len(*services))
		for _, service := range *services {
			fmt.Printf("Service ID [%d]\n", service.ServiceId)
			endpoint, ssl := endpointToString(service.Endpoint)
			fmt.Printf("\tEndpoint [%s] SSL required [%t]\n", endpoint, ssl)
			fmt.Printf("\tType [%s]\n", apiServiceTypeToStr(service.Details.Type))
			fmt.Printf("\tPermanent [%t]\n", service.Permanent)
			for name, value := range service.Details.Properties {
				fmt.Printf("\t[%s] = [%s]\n", name, value)
			}
		}

		return nil
	},
}
