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

package grpcservers

import (
	"context"
	"fmt"
	"io"

	cogmentAPI "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type DirectoryServer struct {
	cogmentAPI.UnimplementedDirectorySPServer
	db *MemoryDB
}

func (ds *DirectoryServer) Register(inOutStream cogmentAPI.DirectorySP_RegisterServer) error {
	data, ok := metadata.FromIncomingContext(inOutStream.Context())
	if !ok {
		return status.Errorf(codes.DataLoss, "Failed to get metadata")
	}

	token := ""
	if values, exist := data["authentication-token"]; exist && len(values) > 0 {
		token = values[0]
	}

	for {
		request, err := inOutStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		reply := cogmentAPI.RegisterReply{}
		protocol := request.Endpoint.Protocol
		serviceType := request.Details.Type

		if len(request.Endpoint.Host) == 0 {
			reply.Status = cogmentAPI.RegisterReply_FAILED
			reply.ErrorMsg = "Empty host"
			log.Debug("Empty host for registration")
		} else if protocol < cogmentAPI.ServiceEndpoint_GRPC || protocol > cogmentAPI.ServiceEndpoint_COGMENT {
			reply.Status = cogmentAPI.RegisterReply_FAILED
			reply.ErrorMsg = fmt.Sprintf("Invalid protocol [%d]", int32(protocol))
			log.WithFields(logrus.Fields{"protocol": int32(protocol)}).Debug("Invalid protocol for registration")
		} else if (serviceType < cogmentAPI.ServiceType_TRIAL_LIFE_CYCLE_SERVICE ||
			serviceType > cogmentAPI.ServiceType_MODEL_REGISTRY_SERVICE) &&
			serviceType != cogmentAPI.ServiceType_OTHER_SERVICE {
			reply.Status = cogmentAPI.RegisterReply_FAILED
			reply.ErrorMsg = fmt.Sprintf("Invalid service type [%d]", int32(serviceType))
			log.WithFields(logrus.Fields{"type": int32(serviceType)}).Debug("Invalid service type for registration")
		} else {
			id, secret, err := ds.db.Insert(token, request.Endpoint, request.Details)
			if err == nil {
				reply.Status = cogmentAPI.RegisterReply_OK
				reply.ServiceId = uint64(id)
				reply.Secret = secret
				log.WithFields(logrus.Fields{
					"service_id": id,
					"type":       request.Details.Type,
				}).Info("New registered service")
				log.WithFields(logrus.Fields{
					"service_id": id,
					"token":      token,
					"secret":     secret,
					"endpoint":   request.Endpoint.String(),
					"details":    request.Details.String(),
				}).Debug("New service")
			} else {
				reply.Status = cogmentAPI.RegisterReply_FAILED
				reply.ErrorMsg = fmt.Sprintf("Could not add service to database [%s]", err.Error())
				log.WithFields(logrus.Fields{"error": err.Error()}).Debug("Failed to insert in database")
			}
		}

		err = inOutStream.Send(&reply)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ds *DirectoryServer) Deregister(inOutStream cogmentAPI.DirectorySP_DeregisterServer) error {
	data, ok := metadata.FromIncomingContext(inOutStream.Context())
	if !ok {
		return status.Errorf(codes.DataLoss, "Failed to get metadata")
	}

	token := ""
	if values, exist := data["authentication-token"]; exist && len(values) > 0 {
		token = values[0]
	}

	for {
		request, err := inOutStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		id := request.ServiceId
		log := log.WithFields(logrus.Fields{
			"service_id": id,
		})

		reply := cogmentAPI.DeregisterReply{}
		idSecret, idToken, err := ds.db.SelectSecretsByID(ServiceID(id))
		if err != nil {
			reply.Status = cogmentAPI.DeregisterReply_FAILED
			reply.ErrorMsg = fmt.Sprintf("Failed to inquire service id [%s]", err.Error())
			log.WithFields(logrus.Fields{"error": err.Error()}).Debug("Failed to inquire database")
		} else if idToken != token || idSecret != request.Secret {
			reply.Status = cogmentAPI.DeregisterReply_FAILED
			reply.ErrorMsg = fmt.Sprintf("Authentication failure for service id [%d]", id)
			log.WithFields(logrus.Fields{
				"db_token":       idToken,
				"request_token":  token,
				"db_secret":      idSecret,
				"request_secret": request.Secret,
			}).Debug("Authentication failure")
		} else {
			ds.db.Delete(ServiceID(id))
			reply.Status = cogmentAPI.DeregisterReply_OK
			log.Info("Deregistered service")
		}

		err = inOutStream.Send(&reply)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ds *DirectoryServer) Inquire(request *cogmentAPI.InquireRequest, outStream cogmentAPI.DirectorySP_InquireServer,
) error {
	data, ok := metadata.FromIncomingContext(outStream.Context())
	if !ok {
		return status.Errorf(codes.DataLoss, "Failed to get metadata")
	}

	token := ""
	if values, exist := data["authentication-token"]; exist && len(values) > 0 {
		token = values[0]
	}

	var requestDetails *cogmentAPI.ServiceDetails
	var ids *[]ServiceID

	switch inquiry := request.Inquiry.(type) {
	case *cogmentAPI.InquireRequest_ServiceId:
		ids = &[]ServiceID{ServiceID(inquiry.ServiceId)}
	case *cogmentAPI.InquireRequest_Details:
		requestDetails = inquiry.Details
		selectionIds, err := ds.db.SelectByDetails(requestDetails)
		if err != nil {
			return fmt.Errorf("Unable to inquire service details [%w]", err)
		}
		ids = &selectionIds
	case nil:
		return fmt.Errorf("Empty request")
	default:
		return fmt.Errorf("Unknown request type [%T]", inquiry)
	}

	for _, id := range *ids {
		log := log.WithFields(logrus.Fields{
			"service_id": id,
		})

		_, idToken, err := ds.db.SelectSecretsByID(id)
		if err != nil {
			if requestDetails == nil {
				log.Debug("Service inquiry for unknown service ID")
			} else {
				log.Debug("Service was removed from the DB in intermediate step")
			}
		} else if idToken != token {
			logAuth := log.WithFields(logrus.Fields{
				"db_token":      idToken,
				"request_token": token,
			})

			if requestDetails != nil {
				logAuth = logAuth.WithFields(logrus.Fields{
					"request_details": requestDetails.String(),
				})
			}
			logAuth.Debug("Authentication failure")
		} else {
			endpoint, details, err := ds.db.SelectByID(id)
			if err != nil {
				log.Debug("Service was removed from the DB in intermediate step")
			} else {
				data := cogmentAPI.FullServiceData{
					Endpoint:  endpoint,
					ServiceId: uint64(id),
					Details:   details,
				}
				reply := cogmentAPI.InquireReply{Data: &data}
				err = outStream.Send(&reply)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (ds *DirectoryServer) Version(context.Context, *cogmentAPI.VersionRequest) (*cogmentAPI.VersionInfo, error) {
	return &cogmentAPI.VersionInfo{}, nil
}

func RegisterDirectoryServer(grpcServer grpc.ServiceRegistrar) error {
	server := &DirectoryServer{}
	server.db = MakeMemoryDb()
	cogmentAPI.RegisterDirectorySPServer(grpcServer, server)
	return nil
}
