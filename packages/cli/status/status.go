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

package status

import (
	"context"
	"fmt"
	"time"

	directoryClient "github.com/cogment/cogment/clients/directory"
	"github.com/cogment/cogment/cmd/services/utils"
	cogmentAPI "github.com/cogment/cogment/grpcapi/cogment/api"
	globalUtils "github.com/cogment/cogment/utils"
	"github.com/cogment/cogment/utils/constants"
	"github.com/cogment/cogment/utils/endpoint"
)

const connectionTimeout = 10 * time.Second

func singleGrpcService(ctx context.Context, srvEndpoint *cogmentAPI.ServiceEndpoint, srvType cogmentAPI.ServiceType,
	statuses []string) bool {

	err := serviceOutput(ctx, srvType, srvEndpoint, statuses)
	if err != nil {
		fmt.Printf(" ::: %v\n", err)
		return false
	}
	fmt.Printf("\n")
	return true
}

func serviceList(ctx context.Context, services *[]*cogmentAPI.FullServiceData,
	verboseLevel int, statuses []string) {

	for _, service := range *services {
		fmt.Printf("[%d", service.ServiceId)
		if verboseLevel >= 1 {
			endpoint, ssl := globalUtils.EndpointToString(service.Endpoint)
			var sslStr string
			if ssl {
				sslStr = " -- ssl"
			}
			fmt.Printf(" -- %s%s", endpoint, sslStr)
		}
		if verboseLevel >= 2 {
			fmt.Printf(" -- %s", globalUtils.APIServiceTypeToStr(service.Details.Type))
		}
		if verboseLevel >= 3 {
			var permStr string
			if service.Permanent {
				permStr = "permanent"
			} else {
				permStr = "transient"
			}
			fmt.Printf(" -- %s", permStr)
		}
		if verboseLevel >= 4 {
			first := true
			for name, value := range service.Details.Properties {
				if first {
					fmt.Printf(" --")
				} else {
					fmt.Printf(",")
				}
				fmt.Printf(" %s = %s", name, value)
				first = false
			}
		}
		fmt.Printf("]")

		singleGrpcService(ctx, service.Endpoint, service.Details.Type, statuses)
	}
}

func requestDirectoryServices(ctx context.Context, dirClient *directoryClient.Client, srvEndpoint *endpoint.Endpoint,
	verboseLevel int, statuses []string) error {

	request := cogmentAPI.InquireRequest{}
	if srvEndpoint.ServiceDiscoveryID != 0 {
		request.Inquiry = &cogmentAPI.InquireRequest_ServiceId{ServiceId: srvEndpoint.ServiceDiscoveryID}
	} else {
		request.Inquiry = &cogmentAPI.InquireRequest_Details{Details: srvEndpoint.Details}
	}

	services, err := dirClient.Inquire(&request)
	if err != nil {
		return fmt.Errorf("Failed to inquire services from directory: %w", err)
	}

	serviceList(ctx, services, verboseLevel, statuses)

	return nil
}

func allDirectoryServices(ctx context.Context, dirClient *directoryClient.Client,
	verboseLevel int, statuses []string) error {

	request := cogmentAPI.InquireRequest{}
	details := cogmentAPI.ServiceDetails{
		Type: cogmentAPI.ServiceType_UNKNOWN_SERVICE, // All service types
	}
	request.Inquiry = &cogmentAPI.InquireRequest_Details{Details: &details}

	services, err := dirClient.Inquire(&request)
	if err != nil {
		return fmt.Errorf("Failed to inquire services from directory: %w", err)
	}

	serviceList(ctx, services, verboseLevel, statuses)

	return nil
}

func Run(dirEndpointStr string, dirToken string, srvEndpointStr string, srvTypeStr string,
	verboseLevel int, statuses []string) error {
	ctx := utils.ContextWithUserTermination()

	serviceEndpoint, err := endpoint.Parse(srvEndpointStr)
	if err != nil {
		return err
	}

	if serviceEndpoint.Category == endpoint.GrpcEndpoint {
		srvType, err := globalUtils.StrToAPIServiceType(srvTypeStr)
		if err != nil {
			return err
		}

		fmt.Printf("%s", srvTypeStr)
		singleGrpcService(ctx, serviceEndpoint.APIEndpoint, srvType, statuses)

	} else if len(dirEndpointStr) != 0 {
		if serviceEndpoint.Category != endpoint.DiscoveryEndpoint {
			return fmt.Errorf(
				"A discovery endpoint (starting with 'cogment://discover') is needed for directory inquiry [%s]",
				srvEndpointStr)
		}

		directoryEndpoint, err := endpoint.Parse(dirEndpointStr)
		if err != nil {
			return err
		}

		fmt.Printf("%s", constants.ServiceTypeDirectory)
		success := singleGrpcService(ctx, directoryEndpoint.APIEndpoint, cogmentAPI.ServiceType_DIRECTORY_SERVICE,
			statuses)
		if !success {
			return nil
		}

		timedCtx, cancel := context.WithTimeout(ctx, connectionTimeout)
		defer cancel()
		dirClient, err := directoryClient.CreateClient(timedCtx, directoryEndpoint, dirToken)
		if err != nil {
			return err
		}

		if len(srvEndpointStr) == 0 {
			err = allDirectoryServices(ctx, dirClient, verboseLevel, statuses)
		} else {
			err = requestDirectoryServices(ctx, dirClient, serviceEndpoint, verboseLevel, statuses)
		}
		if err != nil {
			return err
		}

	} else {
		return fmt.Errorf("A grpc service endpoint or a directory is necessary to inquire statuses")
	}

	return nil
}
