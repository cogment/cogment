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

	cogmentAPI "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/cogment/cogment/utils"
	"github.com/cogment/cogment/utils/constants"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func serviceOutput(ctx context.Context, srvType cogmentAPI.ServiceType, endpoint *cogmentAPI.ServiceEndpoint,
	statuses []string) error {

	if srvType == cogmentAPI.ServiceType_OTHER_SERVICE {
		return fmt.Errorf("Cannot inquire statuses from [%s] service", constants.ServiceTypeOther)
	}

	var serviceTypeList []cogmentAPI.ServiceType
	if srvType != cogmentAPI.ServiceType_UNKNOWN_SERVICE {
		serviceTypeList = append(serviceTypeList, srvType)
	} else {
		for _, serviceStr := range constants.ServiceTypeAllList() {
			serviceType, _ := utils.StrToAPIServiceType(serviceStr)
			if serviceType == cogmentAPI.ServiceType_OTHER_SERVICE {
				continue
			}
			serviceTypeList = append(serviceTypeList, serviceType)
		}
	}

	timedCtx, cancel := context.WithTimeout(ctx, connectionTimeout)
	address := fmt.Sprintf("%s:%d", endpoint.Host, endpoint.Port)
	connection, err := grpc.DialContext(timedCtx, address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	cancel()
	if err != nil {
		return fmt.Errorf("Failed to connect to [%s]: %w", address, err)
	}
	defer connection.Close()

	request := cogmentAPI.StatusRequest{Names: statuses}
	reply := cogmentAPI.StatusReply{}
	for _, serviceType := range serviceTypeList {
		method, typeErr := utils.ServiceTypeToMethodStr(serviceType)
		if typeErr != nil {
			return typeErr
		}
		method += "/Status"

		err = connection.Invoke(ctx, method, &request, &reply, grpc.WaitForReady(false))
		if err == nil {
			if srvType == cogmentAPI.ServiceType_UNKNOWN_SERVICE {
				fmt.Printf("%s", utils.APIServiceTypeToStr(serviceType))
			}
			for status, value := range reply.Statuses {
				fmt.Printf(" [%s:%s]", status, value)
			}

			break
		}
	}

	if err != nil {
		return fmt.Errorf("Status call to [%s] failed: %w", address, err)
	}

	return nil
}
