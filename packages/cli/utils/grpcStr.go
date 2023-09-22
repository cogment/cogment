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

	cogmentAPI "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/cogment/cogment/utils/constants"
)

func EndpointToString(endpoint *cogmentAPI.ServiceEndpoint) (string, bool) {
	var endpointStr string
	ssl := false

	switch endpoint.Protocol {
	case cogmentAPI.ServiceEndpoint_UNKNOWN:
		endpointStr = "<unknown>://"
	case cogmentAPI.ServiceEndpoint_GRPC:
		endpointStr = constants.GrpcScheme + "://"
	case cogmentAPI.ServiceEndpoint_GRPC_SSL:
		endpointStr = constants.GrpcScheme + "://"
		ssl = true
	case cogmentAPI.ServiceEndpoint_COGMENT:
		endpointStr = constants.CogmentScheme + "://"
	default:
		endpointStr = "<invalid>://"
	}

	endpointStr = endpointStr + endpoint.Host

	if endpoint.Port != 0 {
		endpointStr = endpointStr + ":" + fmt.Sprint(endpoint.Port)
	}

	return endpointStr, ssl
}

func APIServiceTypeToStr(apiType cogmentAPI.ServiceType) string {
	var typeStr string
	switch apiType {
	case cogmentAPI.ServiceType_UNKNOWN_SERVICE:
		typeStr = ""
	case cogmentAPI.ServiceType_ACTOR_SERVICE:
		typeStr = constants.ServiceTypeActor
	case cogmentAPI.ServiceType_ENVIRONMENT_SERVICE:
		typeStr = constants.ServiceTypeEnvironment
	case cogmentAPI.ServiceType_PRE_HOOK_SERVICE:
		typeStr = constants.ServiceTypePrehook
	case cogmentAPI.ServiceType_DATALOG_SERVICE:
		typeStr = constants.ServiceTypeDatalog
	case cogmentAPI.ServiceType_TRIAL_LIFE_CYCLE_SERVICE:
		typeStr = constants.ServiceTypeLifecycle
	case cogmentAPI.ServiceType_CLIENT_ACTOR_CONNECTION_SERVICE:
		typeStr = constants.ServiceTypeClientActor
	case cogmentAPI.ServiceType_DATASTORE_SERVICE:
		typeStr = constants.ServiceTypeDatastore
	case cogmentAPI.ServiceType_MODEL_REGISTRY_SERVICE:
		typeStr = constants.ServiceTypeModelRegistry
	case cogmentAPI.ServiceType_DIRECTORY_SERVICE:
		typeStr = constants.ServiceTypeDirectory
	case cogmentAPI.ServiceType_OTHER_SERVICE:
		typeStr = constants.ServiceTypeOther
	default:
		typeStr = "<invalid>"
	}

	return typeStr
}

func StrToAPIServiceType(typeStr string) (cogmentAPI.ServiceType, error) {
	var serviceType cogmentAPI.ServiceType
	switch typeStr {
	case "":
		serviceType = cogmentAPI.ServiceType_UNKNOWN_SERVICE
	case constants.ServiceTypeActor:
		serviceType = cogmentAPI.ServiceType_ACTOR_SERVICE
	case constants.ServiceTypeEnvironment:
		serviceType = cogmentAPI.ServiceType_ENVIRONMENT_SERVICE
	case constants.ServiceTypePrehook:
		serviceType = cogmentAPI.ServiceType_PRE_HOOK_SERVICE
	case constants.ServiceTypeDatalog:
		serviceType = cogmentAPI.ServiceType_DATALOG_SERVICE
	case constants.ServiceTypeLifecycle:
		serviceType = cogmentAPI.ServiceType_TRIAL_LIFE_CYCLE_SERVICE
	case constants.ServiceTypeClientActor:
		serviceType = cogmentAPI.ServiceType_CLIENT_ACTOR_CONNECTION_SERVICE
	case constants.ServiceTypeDatastore:
		serviceType = cogmentAPI.ServiceType_DATASTORE_SERVICE
	case constants.ServiceTypeModelRegistry:
		serviceType = cogmentAPI.ServiceType_MODEL_REGISTRY_SERVICE
	case constants.ServiceTypeDirectory:
		serviceType = cogmentAPI.ServiceType_DIRECTORY_SERVICE
	case constants.ServiceTypeOther:
		serviceType = cogmentAPI.ServiceType_OTHER_SERVICE
	default:
		return serviceType, fmt.Errorf("Invalid service name [%s]", typeStr)
	}

	return serviceType, nil
}

func StrToAPIProtocol(protocolStr string, ssl bool) (cogmentAPI.ServiceEndpoint_Protocol, error) {
	var protocol cogmentAPI.ServiceEndpoint_Protocol
	switch protocolStr {
	case constants.GrpcScheme:
		if ssl {
			protocol = cogmentAPI.ServiceEndpoint_GRPC_SSL
		} else {
			protocol = cogmentAPI.ServiceEndpoint_GRPC
		}
	case constants.CogmentScheme:
		protocol = cogmentAPI.ServiceEndpoint_COGMENT
	default:
		return protocol, fmt.Errorf("Invalid protocol [%s]", protocolStr)
	}

	return protocol, nil
}

func ServiceTypeToMethodStr(serviceType cogmentAPI.ServiceType) (string, error) {
	var method string
	switch serviceType {
	case cogmentAPI.ServiceType_UNKNOWN_SERVICE:
		return method, fmt.Errorf("Unknown service type [%v]", serviceType)
	case cogmentAPI.ServiceType_ACTOR_SERVICE:
		method = "/cogmentAPI.ServiceActorSP/Status"
	case cogmentAPI.ServiceType_ENVIRONMENT_SERVICE:
		method = "/cogmentAPI.EnvironmentSP/Status"
	case cogmentAPI.ServiceType_PRE_HOOK_SERVICE:
		method = "/cogmentAPI.TrialHooksSP/Status"
	case cogmentAPI.ServiceType_DATALOG_SERVICE:
		method = "/cogmentAPI.DatalogSP/Status"
	case cogmentAPI.ServiceType_TRIAL_LIFE_CYCLE_SERVICE:
		method = "/cogmentAPI.TrialLifecycleSP/Status"
	case cogmentAPI.ServiceType_CLIENT_ACTOR_CONNECTION_SERVICE:
		method = "/cogmentAPI.ClientActorSP/Status"
	case cogmentAPI.ServiceType_DATASTORE_SERVICE:
		method = "/cogmentAPI.TrialDatastoreSP/Status"
	case cogmentAPI.ServiceType_MODEL_REGISTRY_SERVICE:
		method = "/cogmentAPI.ModelRegistrySP/Status"
	case cogmentAPI.ServiceType_DIRECTORY_SERVICE:
		method = "/cogmentAPI.DirectorySP/Status"
	case cogmentAPI.ServiceType_OTHER_SERVICE:
		return method, fmt.Errorf("Not a Cogment service type [%v]", serviceType)
	default:
		return method, fmt.Errorf("Invalid service type [%v]", serviceType)
	}

	return method, nil
}
