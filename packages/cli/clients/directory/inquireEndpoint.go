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

package directory

import (
	"context"
	"fmt"

	grpcapi "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/cogment/cogment/utils/endpoint"
)

func InquireEndpoint(
	ctx context.Context,
	inquiredEndpoint *endpoint.Endpoint,
	directoryEndpoint *endpoint.Endpoint,
	directoryAuthToken string,
) ([]*endpoint.Endpoint, error) {
	log := log.WithField("endpoint", inquiredEndpoint)

	if inquiredEndpoint.IsResolved() {
		log.Debug("No need to inquire directory, endpoint is already resolved")
		return []*endpoint.Endpoint{inquiredEndpoint}, nil
	}

	if !directoryEndpoint.IsValid() {
		return []*endpoint.Endpoint{},
			fmt.Errorf("Can't inquire endpoint [%s], no valid directory endpoint provided", inquiredEndpoint)
	}

	log = log.WithField("directory_endpoint", directoryEndpoint)
	if inquiredEndpoint.Category != endpoint.DiscoveryEndpoint {
		return []*endpoint.Endpoint{},
			fmt.Errorf("Can't inquire non-discovery endpoint [%s]", inquiredEndpoint)
	}

	inquiryDirectoryAuthToken, ok := inquiredEndpoint.AuthenticationToken()
	if ok {
		log.Debug("Using the authentication token specified in the endpoint properties")
		directoryAuthToken = inquiryDirectoryAuthToken
	}

	directoryClient, err := CreateClient(ctx, directoryEndpoint, directoryAuthToken)
	if err != nil {
		return []*endpoint.Endpoint{}, err
	}

	var request grpcapi.InquireRequest
	if inquiredEndpoint.ServiceDiscoveryID != 0 {
		request = grpcapi.InquireRequest{
			Inquiry: &grpcapi.InquireRequest_ServiceId{
				ServiceId: inquiredEndpoint.ServiceDiscoveryID,
			},
		}
	} else {
		request = grpcapi.InquireRequest{
			Inquiry: &grpcapi.InquireRequest_Details{
				Details: inquiredEndpoint.Details,
			},
		}
	}
	services, err := directoryClient.Inquire(&request)
	if err != nil {
		return []*endpoint.Endpoint{}, err
	}
	if len(*services) == 0 {
		return []*endpoint.Endpoint{},
			fmt.Errorf("No service matching endpoint [%s] found in directory [%s]", inquiredEndpoint, directoryEndpoint)
	}
	resolvedEndpoints := []*endpoint.Endpoint{}
	for _, service := range *services {
		resolvedEndpoint, err := inquiredEndpoint.Resolve(
			service.ServiceId,
			service.Endpoint.Host,
			service.Endpoint.Port,
			service.Details.Type,
		)
		if err != nil {
			return []*endpoint.Endpoint{}, err
		}
		resolvedEndpoints = append(resolvedEndpoints, resolvedEndpoint)
	}

	return resolvedEndpoints, nil
}
