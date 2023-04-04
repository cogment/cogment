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

	cogmentAPI "github.com/cogment/cogment/grpcapi/cogment/api"
	directoryService "github.com/cogment/cogment/services/directory"
	"github.com/cogment/cogment/version"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	// Options
	directoryEndpointKey     = "directory_endpoint"
	directoryEndpointEnvKey  = "COGMENT_DIRECTORY_ENDPOINT"
	directoryAuthTokenKey    = "directory_authentication_token"
	directoryAuthTokenEnvKey = "COGMENT_DIRECTORY_AUTHENTICATION_TOKEN"

	directoryRegisterHostKey      = "host"
	directoryRegisterPortKey      = "port"
	directoryRegisterProtocolKey  = "protocol"
	directoryRegisterSslKey       = "ssl_required"
	directoryServiceTypeKey       = "type"
	directoryServicePropertiesKey = "properties"
	directoryServiceIDKey         = "service_id"
	directoryServiceSecretKey     = "secret"
	directoryServicePermanent     = "permanent"

	directoryServiceTypeActor         = "actor"
	directoryServiceTypeEnvironment   = "environment"
	directoryServiceTypePrehook       = "prehook"
	directoryServiceTypeDatalog       = "datalog"
	directoryServiceTypeLifecycle     = "lifecycle"
	directoryServiceTypeClientActor   = "actservice"
	directoryServiceTypeDatastore     = "datastore"
	directoryServiceTypeModelRegistry = "modelregistry"
	directoryServiceTypeOther         = "other"

	directoryProtocolTypeGrpc    = "grpc"
	directoryProtocolTypeCogment = "cogment"
)

var additionalRegistrationProperties = map[string]string{
	"__registration_source": "Cogment-Command_Line",
	"__version":             version.Version,
}
var directoryViper = viper.New()

func init() {
	directoryViper.SetDefault(
		directoryEndpointKey,
		fmt.Sprintf("grpc://localhost:%d", directoryService.DefaultOptions.Port),
	)
	_ = directoryViper.BindEnv(directoryEndpointKey, directoryEndpointEnvKey)
	directoryCmd.PersistentFlags().String(
		directoryEndpointKey,
		directoryViper.GetString(directoryEndpointKey),
		"The directory gRPC endpoint URL",
	)
	directoryCmd.PersistentFlags().String(
		"endpoint",
		directoryViper.GetString(directoryEndpointKey),
		"",
	)
	_ = directoryCmd.PersistentFlags().MarkDeprecated(
		"endpoint",
		fmt.Sprintf("please use --%s instead", directoryEndpointKey),
	)

	directoryViper.SetDefault(directoryAuthTokenKey, "")
	_ = directoryViper.BindEnv(directoryAuthTokenKey, directoryAuthTokenEnvKey)
	directoryCmd.PersistentFlags().String(
		directoryAuthTokenKey,
		directoryViper.GetString(directoryAuthTokenKey),
		"The authentication token for the services in the Directory",
	)
	directoryCmd.PersistentFlags().String(
		"auth_token",
		directoryViper.GetString(directoryAuthTokenKey),
		"",
	)
	_ = directoryCmd.PersistentFlags().MarkDeprecated(
		"auth_token",
		fmt.Sprintf("please use --%s instead", directoryAuthTokenKey),
	)

	directoryCmd.PersistentFlags().SortFlags = false
	_ = directoryViper.BindPFlags(directoryCmd.PersistentFlags())

	directoryCmd.AddCommand(directoryRegisterCmd)
	directoryCmd.AddCommand(directoryDeregisterCmd)
	directoryCmd.AddCommand(directoryInquireCmd)
}

func strToAPIServiceType(typeStr string) (cogmentAPI.ServiceType, error) {
	var serviceType cogmentAPI.ServiceType
	switch typeStr {
	case "":
		serviceType = cogmentAPI.ServiceType_UNKNOWN_SERVICE
	case directoryServiceTypeActor:
		serviceType = cogmentAPI.ServiceType_ACTOR_SERVICE
	case directoryServiceTypeEnvironment:
		serviceType = cogmentAPI.ServiceType_ENVIRONMENT_SERVICE
	case directoryServiceTypePrehook:
		serviceType = cogmentAPI.ServiceType_PRE_HOOK_SERVICE
	case directoryServiceTypeDatalog:
		serviceType = cogmentAPI.ServiceType_DATALOG_SERVICE
	case directoryServiceTypeLifecycle:
		serviceType = cogmentAPI.ServiceType_TRIAL_LIFE_CYCLE_SERVICE
	case directoryServiceTypeClientActor:
		serviceType = cogmentAPI.ServiceType_CLIENT_ACTOR_CONNECTION_SERVICE
	case directoryServiceTypeDatastore:
		serviceType = cogmentAPI.ServiceType_DATASTORE_SERVICE
	case directoryServiceTypeModelRegistry:
		serviceType = cogmentAPI.ServiceType_MODEL_REGISTRY_SERVICE
	case directoryServiceTypeOther:
		serviceType = cogmentAPI.ServiceType_OTHER_SERVICE
	default:
		return 0, fmt.Errorf("Invalid service type [%s]", typeStr)
	}

	return serviceType, nil
}

func apiServiceTypeToStr(apiType cogmentAPI.ServiceType) string {
	switch apiType {
	case cogmentAPI.ServiceType_UNKNOWN_SERVICE:
		return ""
	case cogmentAPI.ServiceType_ACTOR_SERVICE:
		return directoryServiceTypeActor
	case cogmentAPI.ServiceType_ENVIRONMENT_SERVICE:
		return directoryServiceTypeEnvironment
	case cogmentAPI.ServiceType_PRE_HOOK_SERVICE:
		return directoryServiceTypePrehook
	case cogmentAPI.ServiceType_DATALOG_SERVICE:
		return directoryServiceTypeDatalog
	case cogmentAPI.ServiceType_TRIAL_LIFE_CYCLE_SERVICE:
		return directoryServiceTypeLifecycle
	case cogmentAPI.ServiceType_CLIENT_ACTOR_CONNECTION_SERVICE:
		return directoryServiceTypeClientActor
	case cogmentAPI.ServiceType_DATASTORE_SERVICE:
		return directoryServiceTypeDatastore
	case cogmentAPI.ServiceType_MODEL_REGISTRY_SERVICE:
		return directoryServiceTypeModelRegistry
	case cogmentAPI.ServiceType_OTHER_SERVICE:
		return directoryServiceTypeOther
	default:
		return "<unknown>"
	}
}

func strToAPIProtocol(protocolStr string, ssl bool) (cogmentAPI.ServiceEndpoint_Protocol, error) {
	var result cogmentAPI.ServiceEndpoint_Protocol
	switch protocolStr {
	case directoryProtocolTypeGrpc:
		if ssl {
			result = cogmentAPI.ServiceEndpoint_GRPC_SSL
		} else {
			result = cogmentAPI.ServiceEndpoint_GRPC
		}
	case directoryProtocolTypeCogment:
		result = cogmentAPI.ServiceEndpoint_COGMENT
	default:
		return 0, fmt.Errorf("Invalid protocol [%s]", protocolStr)
	}

	return result, nil
}

func endpointToString(endpoint *cogmentAPI.ServiceEndpoint) (string, bool) {
	var result string
	ssl := false

	switch endpoint.Protocol {
	case cogmentAPI.ServiceEndpoint_GRPC:
		result = directoryProtocolTypeGrpc + "://"
	case cogmentAPI.ServiceEndpoint_GRPC_SSL:
		result = directoryProtocolTypeGrpc + "://"
		ssl = true
	case cogmentAPI.ServiceEndpoint_COGMENT:
		result = directoryProtocolTypeCogment + "://"
	default:
		result = result + "<unknown>://"
	}

	result = result + endpoint.Host

	if endpoint.Port != 0 {
		result = result + ":" + fmt.Sprint(endpoint.Port)
	}

	return result, ssl
}

var directoryCmd = &cobra.Command{
	Use:   "directory",
	Short: "Run directory client",
	Args:  cobra.NoArgs,
}
