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

	"google.golang.org/protobuf/proto"

	cogmentAPI "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/cogment/cogment/utils"
	"github.com/cogment/cogment/utils/constants"
)

const (
	ActorClassPropertyName          = "__actor_class"
	ImplementationPropertyName      = "__implementation"
	ServiceIDPropertyName           = "__id"
	AuthenticationTokenPropertyName = "__authentication-token"
	RegistrationSourcePropertyName  = "__registration_source"
	VersionPropertyName             = "__version"
)

const servicePath = "service"

type CategoryType int

const (
	InvalidEndpoint CategoryType = iota
	GrpcEndpoint
	DiscoveryEndpoint
	ClientEndpoint
)

type Endpoint struct {
	Category           CategoryType
	APIEndpoint        *cogmentAPI.ServiceEndpoint
	Details            *cogmentAPI.ServiceDetails
	ServiceDiscoveryID uint64
}

func endpointCopy(dst *Endpoint, src *Endpoint) {
	dst.Category = src.Category
	dst.APIEndpoint = proto.Clone(src.APIEndpoint).(*cogmentAPI.ServiceEndpoint)
	dst.ServiceDiscoveryID = src.ServiceDiscoveryID

	dst.Details = proto.Clone(src.Details).(*cogmentAPI.ServiceDetails)
	dst.Details.Properties = utils.CopyStrMap(src.Details.Properties)
}

func Parse(endpointStr string) (*Endpoint, error) {
	endpointURL, err := url.Parse(endpointStr)
	if err != nil {
		return &Endpoint{}, fmt.Errorf("Not a valid URL [%s]: %w", endpointStr, err)
	}
	switch endpointURL.Scheme {
	case constants.GrpcScheme:
		if endpointURL.Path != "" ||
			endpointURL.RawQuery != "" ||
			endpointURL.RawFragment != "" ||
			endpointURL.User != nil ||
			endpointURL.Port() == "" {
			return &Endpoint{}, fmt.Errorf(
				"Invalid grpc endpoint [%s], expected 'grpc://<hostname>:<port>'",
				endpointURL,
			)
		}
		port, err := strconv.ParseUint(endpointURL.Port(), 10, 32)
		if err != nil {
			return &Endpoint{}, fmt.Errorf(
				"Invalid grpc endpoint [%s], expected unsigned integer port",
				endpointURL,
			)
		}
		return &Endpoint{
			Category: GrpcEndpoint,
			APIEndpoint: &cogmentAPI.ServiceEndpoint{
				Host: endpointURL.Hostname(),
				Port: uint32(port),
			},
			Details: &cogmentAPI.ServiceDetails{
				Type:       cogmentAPI.ServiceType_UNKNOWN_SERVICE,
				Properties: map[string]string{},
			},
		}, nil
	case constants.CogmentScheme:
		switch endpointURL.Host {
		case constants.DiscoveryHost:
			if endpointURL.RawFragment != "" ||
				endpointURL.User != nil {
				return &Endpoint{}, fmt.Errorf(
					"Invalid cogment discover endpoint [%s], expected 'cogment://discover/<service_type>?prop=value'",
					endpointURL,
				)
			}

			var serviceType cogmentAPI.ServiceType
			var discoveryServicePath bool
			path := strings.TrimPrefix(endpointURL.Path, "/")
			if len(path) == 0 {
				serviceType = cogmentAPI.ServiceType_UNKNOWN_SERVICE
			} else if path == servicePath {
				serviceType = cogmentAPI.ServiceType_UNKNOWN_SERVICE
				discoveryServicePath = true
			} else {
				serviceType, err = utils.StrToAPIServiceType(path)
				if err != nil {
					return &Endpoint{}, err
				}
			}

			endpoint := Endpoint{
				Category:    DiscoveryEndpoint,
				APIEndpoint: &cogmentAPI.ServiceEndpoint{},
				Details: &cogmentAPI.ServiceDetails{
					Type:       serviceType,
					Properties: map[string]string{},
				},
			}

			for key, valueList := range endpointURL.Query() {
				if len(valueList) > 1 {
					return &Endpoint{}, fmt.Errorf(
						"Invalid cogment endpoint [%s], multiple values defined for property [%s]",
						endpointURL,
						key,
					)
				} else if len(valueList) == 1 {
					value := valueList[0]
					if discoveryServicePath && key == ServiceIDPropertyName {
						if len(endpointURL.Query()) > 1 {
							return &Endpoint{}, fmt.Errorf(
								"A cogment discovery endpoint with '%s' path only accepts '%s' as query [%s]",
								servicePath,
								ServiceIDPropertyName,
								endpointURL,
							)
						}
						intValue, err := strconv.ParseUint(value, 10, 64)
						if err != nil || intValue == 0 {
							return &Endpoint{}, fmt.Errorf(
								"Invalid cogment endpoint [%s], invalid service id [%s]",
								endpointURL,
								value,
							)
						}
						endpoint.ServiceDiscoveryID = intValue
					}
					endpoint.Details.Properties[key] = value
				} else {
					endpoint.Details.Properties[key] = ""
				}
			}

			if discoveryServicePath && endpoint.ServiceDiscoveryID == 0 {
				return &Endpoint{}, fmt.Errorf(
					"Invalid cogment service endpoint [%s], expected '%s' query",
					endpointURL,
					ServiceIDPropertyName,
				)
			}

			return &endpoint, nil
		case constants.ClientHost:
			if endpointURL.Path != "" ||
				endpointURL.RawQuery != "" ||
				endpointURL.RawFragment != "" ||
				endpointURL.User != nil {
				return &Endpoint{}, fmt.Errorf(
					"Invalid cogment client endpoint [%s], expected 'cogment://client'",
					endpointURL,
				)
			}
			return &Endpoint{
				Category: ClientEndpoint,
			}, nil
		default:
			return &Endpoint{}, fmt.Errorf(
				"Invalid cogment endpoint [%s], unexpected host",
				endpointURL,
			)
		}
	default:
		return &Endpoint{},
			fmt.Errorf("Invalid endpoint [%s], unexpected scheme", endpointURL)
	}
}

func MustParse(endpointStr string) *Endpoint {
	endpoint, err := Parse(endpointStr)
	if err != nil {
		panic(err)
	}
	return endpoint
}

func (endpoint *Endpoint) Copy() *Endpoint {
	newEndpoint := Endpoint{}
	endpointCopy(&newEndpoint, endpoint)
	return &newEndpoint
}

func (endpoint *Endpoint) IsValid() bool {
	return endpoint.Category != InvalidEndpoint
}

func (endpoint *Endpoint) Address() string {
	return fmt.Sprintf("%s:%d", endpoint.Host(), endpoint.Port())
}

func (endpoint *Endpoint) Host() string {
	return endpoint.APIEndpoint.Host
}

func (endpoint *Endpoint) Port() uint32 {
	return endpoint.APIEndpoint.Port
}

func (endpoint *Endpoint) URL() *url.URL {
	switch endpoint.Category {
	case GrpcEndpoint:
		return &url.URL{
			Scheme: constants.GrpcScheme,
			Host:   endpoint.Address(),
		}
	case DiscoveryEndpoint:
		query := url.Values{}
		for propertyKey, propertyValue := range endpoint.Details.Properties {
			query.Set(propertyKey, propertyValue)
		}
		var path string
		if endpoint.ServiceDiscoveryID != 0 {
			path = servicePath
		} else {
			path = utils.APIServiceTypeToStr(endpoint.Details.Type)
		}
		return &url.URL{
			Scheme:   constants.CogmentScheme,
			Host:     constants.DiscoveryHost,
			Path:     path,
			RawQuery: query.Encode(),
		}
	case ClientEndpoint:
		return &url.URL{
			Scheme: constants.CogmentScheme,
			Host:   constants.ClientHost,
		}
	default:
		return &url.URL{}
	}
}

func (endpoint *Endpoint) IsResolved() bool {
	switch endpoint.Category {
	case GrpcEndpoint:
		return true
	case DiscoveryEndpoint:
		return endpoint.Host() != ""
	default:
		return false
	}
}

func (endpoint *Endpoint) ResolvedURL() (*url.URL, error) {
	switch endpoint.Category {
	case GrpcEndpoint:
		return endpoint.URL(), nil
	case DiscoveryEndpoint:
		if endpoint.Host() != "" {
			return &url.URL{
				Scheme: constants.GrpcScheme,
				Host:   endpoint.Address(),
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
	serviceType cogmentAPI.ServiceType,
) (*Endpoint, error) {
	switch endpoint.Category {
	case GrpcEndpoint:
		return &Endpoint{}, errors.New("gRPC endpoint, already resolved")
	case DiscoveryEndpoint:
		if endpoint.Host() != "" {
			return &Endpoint{}, errors.New("discovery endpoint was already resolved")
		}
		if endpoint.Details.Type != cogmentAPI.ServiceType_UNKNOWN_SERVICE && endpoint.Details.Type != serviceType {
			return &Endpoint{}, fmt.Errorf(
				"requested service type [%s] doesn't match resolved one [%s]",
				endpoint.Details.Type,
				serviceType,
			)
		}

		if requestedServiceID := endpoint.ServiceDiscoveryID; requestedServiceID != 0 {
			if requestedServiceID != serviceID {
				return &Endpoint{}, fmt.Errorf(
					"requested service id [%d] doesn't match resolved one [%d]",
					requestedServiceID,
					serviceID,
				)
			}
		}
		return &Endpoint{
			Category: DiscoveryEndpoint,
			APIEndpoint: &cogmentAPI.ServiceEndpoint{
				Host: hostname,
				Port: port,
			},
			Details: &cogmentAPI.ServiceDetails{
				Type:       serviceType,
				Properties: endpoint.Details.Properties,
			},
		}, nil
	case ClientEndpoint:
		return &Endpoint{}, errors.New("Client endpoints can't be resolved")
	default:
		return &Endpoint{}, errors.New("Invalid endpoint")
	}
}

func (endpoint *Endpoint) SetServiceType(serviceType cogmentAPI.ServiceType) {
	endpoint.Details.Type = serviceType
}

func (endpoint *Endpoint) AuthenticationToken() (string, bool) {
	switch endpoint.Category {
	case DiscoveryEndpoint:
		token, found := endpoint.Details.Properties[AuthenticationTokenPropertyName]
		return token, found
	default:
		return "", false
	}
}

func (endpoint *Endpoint) String() string {
	url := endpoint.URL()
	switch endpoint.Category {
	case DiscoveryEndpoint:
		str := url.String()
		if endpoint.Host() != "" {
			str += fmt.Sprintf(" [resolved to grpc://%s]", endpoint.Address())
		}
		return str
	default:
		return url.String()
	}
}

func (endpoint *Endpoint) MarshalString() string {
	switch endpoint.Category {
	case InvalidEndpoint:
		return ""
	default:
		return endpoint.URL().String()
	}
}

func (endpoint *Endpoint) MarshalText() ([]byte, error) {
	return []byte(endpoint.MarshalString()), nil
}

func (endpoint *Endpoint) UnmarshalText(b []byte) error {
	parsedVal, err := Parse(string(b))
	endpointCopy(endpoint, parsedVal)
	return err
}
