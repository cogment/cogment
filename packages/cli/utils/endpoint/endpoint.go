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

package endpoint

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	grpcapi "github.com/cogment/cogment/grpcapi/cogment/api"
)

const ActorClassPropertyName = "__actor_class"
const ImplementationPropertyName = "__implementation"
const ServiceIDPropertyName = "__id"
const AuthenticationTokenPropertyName = "__authentication-token"
const RegistrationSourcePropertyName = "__registration_source"
const VersionPropertyName = "__registration_source"

const GrpcScheme = "grpc"
const CogmentScheme = "cogment"
const DiscoveryHost = "discover"
const ClientHost = "client"

type Type int

const (
	InvalidEndpoint Type = iota
	GrpcEndpoint
	DiscoveryEndpoint
	ClientEndpoint
)

var serviceTypesPath = map[grpcapi.ServiceType]string{
	grpcapi.ServiceType_ACTOR_SERVICE:                   "actor",
	grpcapi.ServiceType_ENVIRONMENT_SERVICE:             "environment",
	grpcapi.ServiceType_DATALOG_SERVICE:                 "datalog",
	grpcapi.ServiceType_TRIAL_LIFE_CYCLE_SERVICE:        "lifecycle",
	grpcapi.ServiceType_CLIENT_ACTOR_CONNECTION_SERVICE: "actservice",
	grpcapi.ServiceType_MODEL_REGISTRY_SERVICE:          "modelregistry",
	grpcapi.ServiceType_OTHER_SERVICE:                   "service",
}

func parseServiceType(path string) (grpcapi.ServiceType, error) {
	path = strings.TrimPrefix(path, "/")
	switch path {
	case serviceTypesPath[grpcapi.ServiceType_ACTOR_SERVICE]:
		return grpcapi.ServiceType_ACTOR_SERVICE, nil
	case serviceTypesPath[grpcapi.ServiceType_ACTOR_SERVICE]:
		return grpcapi.ServiceType_ENVIRONMENT_SERVICE, nil
	case serviceTypesPath[grpcapi.ServiceType_ACTOR_SERVICE]:
		return grpcapi.ServiceType_DATALOG_SERVICE, nil
	case serviceTypesPath[grpcapi.ServiceType_ACTOR_SERVICE]:
		return grpcapi.ServiceType_TRIAL_LIFE_CYCLE_SERVICE, nil
	case serviceTypesPath[grpcapi.ServiceType_ACTOR_SERVICE]:
		return grpcapi.ServiceType_CLIENT_ACTOR_CONNECTION_SERVICE, nil
	case serviceTypesPath[grpcapi.ServiceType_ACTOR_SERVICE]:
		return grpcapi.ServiceType_MODEL_REGISTRY_SERVICE, nil
	case "":
		// Considering that an empty serviceStr is the same as "service" to handle "cogment://discover"
		fallthrough
	case serviceTypesPath[grpcapi.ServiceType_ACTOR_SERVICE]:
		return grpcapi.ServiceType_UNKNOWN_SERVICE, nil
	default:
		return grpcapi.ServiceType_UNKNOWN_SERVICE, fmt.Errorf(
			"Invalid cogment service type [%s]",
			path,
		)
	}
}

type Endpoint struct {
	endpointType     Type
	resolvedHostname string
	resolvedPort     uint32
	serviceType      grpcapi.ServiceType
	properties       map[string]string
}

func Parse(endpointStr string) (Endpoint, error) {
	endpointURL, err := url.Parse(endpointStr)
	if err != nil {
		return Endpoint{}, fmt.Errorf("Not a valid URL [%s]: %w", endpointStr, err)
	}
	switch endpointURL.Scheme {
	case GrpcScheme:
		if endpointURL.Path != "" ||
			endpointURL.RawQuery != "" ||
			endpointURL.RawFragment != "" ||
			endpointURL.User != nil ||
			endpointURL.Port() == "" {
			return Endpoint{}, fmt.Errorf(
				"Invalid grpc endpoint [%s], expected 'grpc://<hostname>:<port>'",
				endpointURL,
			)
		}
		port, err := strconv.ParseUint(endpointURL.Port(), 10, 32)
		if err != nil {
			return Endpoint{}, fmt.Errorf(
				"Invalid grpc endpoint [%s], expected unsigned integer port",
				endpointURL,
			)
		}
		return Endpoint{
			endpointType:     GrpcEndpoint,
			resolvedHostname: endpointURL.Hostname(),
			resolvedPort:     uint32(port),
			serviceType:      grpcapi.ServiceType_UNKNOWN_SERVICE,
			properties:       map[string]string{},
		}, nil
	case CogmentScheme:
		switch endpointURL.Host {
		case DiscoveryHost:
			if endpointURL.RawFragment != "" ||
				endpointURL.User != nil {
				return Endpoint{}, fmt.Errorf(
					"Invalid cogment discover endpoint [%s], expected 'cogment://discover/<service_type>?prop=value'",
					endpointURL,
				)
			}
			serviceType, err := parseServiceType(endpointURL.Path)
			if err != nil {
				return Endpoint{}, err
			}
			endpoint := Endpoint{
				endpointType:     DiscoveryEndpoint,
				resolvedHostname: "",
				resolvedPort:     0,
				serviceType:      serviceType,
				properties:       map[string]string{},
			}
			for key, values := range endpointURL.Query() {
				if len(values) > 1 {
					return Endpoint{}, fmt.Errorf(
						"Invalid cogment endpoint [%s], multiple values defined for property [%s]",
						endpointURL,
						key,
					)
				} else if len(values) == 1 {
					value := values[0]
					if key == ServiceIDPropertyName {
						if _, err := strconv.ParseUint(value, 10, 64); err != nil {
							return Endpoint{}, fmt.Errorf(
								"Invalid cogment endpoint [%s], invalid service id [%s]",
								endpointURL,
								value,
							)
						}
					}
					endpoint.properties[key] = value
				} else {
					endpoint.properties[key] = ""
				}
			}
			return endpoint, nil
		case ClientHost:
			if endpointURL.Path != "" ||
				endpointURL.RawQuery != "" ||
				endpointURL.RawFragment != "" ||
				endpointURL.User != nil {
				return Endpoint{}, fmt.Errorf(
					"Invalid cogment client endpoint [%s], expected 'cogment://client'",
					endpointURL,
				)
			}
			return Endpoint{
				endpointType:     ClientEndpoint,
				resolvedHostname: "",
				resolvedPort:     0,
				properties:       map[string]string{},
			}, nil
		default:
			return Endpoint{}, fmt.Errorf(
				"Invalid cogment endpoint [%s], unexpected host",
				endpointURL,
			)
		}
	default:
		return Endpoint{},
			fmt.Errorf("Invalid endpoint [%s], unexpected scheme", endpointURL)
	}
}

func MustParse(endpointStr string) Endpoint {
	endpoint, err := Parse(endpointStr)
	if err != nil {
		panic(err)
	}
	return endpoint
}

func (endpoint *Endpoint) IsValid() bool {
	return endpoint.endpointType != InvalidEndpoint
}

func (endpoint *Endpoint) Type() Type {
	return endpoint.endpointType
}

func (endpoint *Endpoint) URL() *url.URL {
	switch endpoint.endpointType {
	case GrpcEndpoint:
		return &url.URL{
			Scheme: GrpcScheme,
			Host:   fmt.Sprintf("%s:%d", endpoint.resolvedHostname, endpoint.resolvedPort),
		}
	case DiscoveryEndpoint:
		query := url.Values{}
		for propertyKey, propertyValue := range endpoint.properties {
			query.Set(propertyKey, propertyValue)
		}
		return &url.URL{
			Scheme:   CogmentScheme,
			Host:     DiscoveryHost,
			Path:     serviceTypesPath[endpoint.serviceType],
			RawQuery: query.Encode(),
		}
	case ClientEndpoint:
		return &url.URL{
			Scheme: CogmentScheme,
			Host:   ClientHost,
		}
	default:
		return nil
	}
}

func (endpoint *Endpoint) IsResolved() bool {
	switch endpoint.endpointType {
	case GrpcEndpoint:
		return true
	case DiscoveryEndpoint:
		return endpoint.resolvedHostname != ""
	default:
		return false
	}
}

func (endpoint *Endpoint) ResolvedURL() (*url.URL, error) {
	switch endpoint.endpointType {
	case GrpcEndpoint:
		return endpoint.URL(), nil
	case DiscoveryEndpoint:
		if endpoint.resolvedHostname != "" {
			return &url.URL{
				Scheme: GrpcScheme,
				Host:   fmt.Sprintf("%s:%d", endpoint.resolvedHostname, endpoint.resolvedPort),
			}, nil
		}
		return nil, errors.New("Unresolved discovery endpoint")
	case ClientEndpoint:
		return nil, errors.New("Client endpoints can't be resolved")
	default:
		return nil, errors.New("Invalid endpoint")
	}
}

func (endpoint *Endpoint) Resolve(
	serviceID uint64,
	hostname string,
	port uint32,
	serviceType grpcapi.ServiceType,
) (Endpoint, error) {
	switch endpoint.endpointType {
	case GrpcEndpoint:
		return Endpoint{}, errors.New("gRPC endpoint, already resolved")
	case DiscoveryEndpoint:
		if endpoint.resolvedHostname != "" {
			return Endpoint{}, errors.New("discovery endpoint was already resolved")
		}
		if endpoint.serviceType != grpcapi.ServiceType_UNKNOWN_SERVICE && endpoint.serviceType != serviceType {
			return Endpoint{}, fmt.Errorf(
				"requested service type [%s] doesn't match resolved one [%s]",
				endpoint.serviceType,
				serviceType,
			)
		}

		if requestedServiceID, ok := endpoint.ServiceID(); ok {
			if requestedServiceID != serviceID {
				return Endpoint{}, fmt.Errorf(
					"requested service id [%d] doesn't match resolved one [%d]",
					requestedServiceID,
					serviceID,
				)
			}
		}
		return Endpoint{
			endpointType:     DiscoveryEndpoint,
			resolvedHostname: hostname,
			resolvedPort:     port,
			serviceType:      serviceType,
			properties:       endpoint.properties,
		}, nil
	case ClientEndpoint:
		return Endpoint{}, errors.New("Client endpoints can't be resolved")
	default:
		return Endpoint{}, errors.New("Invalid endpoint")
	}
}

func (endpoint *Endpoint) SetServiceType(serviceType grpcapi.ServiceType) {
	endpoint.serviceType = serviceType
}

func (endpoint *Endpoint) ServiceType() grpcapi.ServiceType {
	return endpoint.serviceType
}

func (endpoint *Endpoint) ServiceID() (uint64, bool) {
	switch endpoint.endpointType {
	case DiscoveryEndpoint:
		serviceIDStr, ok := endpoint.properties[ServiceIDPropertyName]
		if !ok {
			return 0, false
		}

		// Validity already checked in `Parse`
		serviceID, _ := strconv.ParseUint(serviceIDStr, 10, 64)
		return serviceID, true
	case GrpcEndpoint:
		fallthrough
	case ClientEndpoint:
		fallthrough
	default:
		return 0, false
	}
}

func (endpoint *Endpoint) AuthenticationToken() (string, bool) {
	switch endpoint.endpointType {
	case DiscoveryEndpoint:
		token, found := endpoint.properties[AuthenticationTokenPropertyName]
		return token, found
	default:
		return "", false
	}
}

func (endpoint *Endpoint) Properties() map[string]string {
	return endpoint.properties
}

func (endpoint Endpoint) String() string {
	url := endpoint.URL()
	switch endpoint.endpointType {
	case DiscoveryEndpoint:
		str := url.String()
		if endpoint.resolvedHostname != "" {
			str += fmt.Sprintf(
				" [resolved to grpc://%s:%d]",
				endpoint.resolvedHostname,
				endpoint.resolvedPort,
			)
		}
		return str
	default:
		return url.String()
	}
}

func (endpoint Endpoint) MarshalString() string {
	switch endpoint.endpointType {
	case InvalidEndpoint:
		return ""
	default:
		return endpoint.URL().String()
	}
}

func (endpoint Endpoint) MarshalText() ([]byte, error) {
	return []byte(endpoint.MarshalString()), nil
}

func (endpoint *Endpoint) UnmarshalText(b []byte) error {
	var err error
	*endpoint, err = Parse(string(b))
	return err
}
