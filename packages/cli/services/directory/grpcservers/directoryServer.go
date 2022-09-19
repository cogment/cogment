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
	"sort"

	cogmentAPI "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/cogment/cogment/utils"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const secretLength = 5

type DirectoryServer struct {
	cogmentAPI.UnimplementedDirectorySPServer
	db *MemoryDB
}

// We want to sort in reverse: from most recent (larger number) to less recent (smaller number).
// So the "Less" function is actually a "More".
type sortableRecords []*DbRecord

func (sr sortableRecords) Len() int      { return len(sr) }
func (sr sortableRecords) Swap(i, j int) { sr[i], sr[j] = sr[j], sr[i] }
func (sr sortableRecords) Less(i, j int) bool {
	return sr[i].registerTimestamp > sr[j].registerTimestamp
}

func (ds *DirectoryServer) dbUpdate(request *cogmentAPI.RegisterRequest, token string) (ServiceID, string, error) {
	if !request.Permanent {
		return 0, "", nil
	}

	selectionIds, err := ds.db.SelectByDetails(request.Details)
	if err != nil {
		return 0, "", fmt.Errorf("Unable to inquire service for duplice verification [%w]", err)
	}

	if len(selectionIds) == 0 {
		return 0, "", nil
	}

	var updatedID ServiceID
	var secret string
	for _, id := range selectionIds {
		record, err := ds.db.SelectByID(id)
		if err != nil {
			continue
		}
		if !record.permanent {
			continue
		}
		if record.authenticationToken != token {
			continue
		}

		if updatedID != 0 {
			// This can happen if the new service is a partial match to multiple services
			// I.e. the existing services were different by properties not in the new service
			ds.db.Delete(id)
			log.WithFields(logrus.Fields{"service_id": id}).Info("Duplicate permanent service removed")
			continue
		}

		updatedRecord := DbRecord{
			lastHealthCheckTimestamp: utils.Timestamp(), // This is an indication that a permanent has been updated
			nbFailedHealthChecks:     0,
			permanent:                record.permanent,
			authenticationToken:      record.authenticationToken,
			secret:                   record.secret,
		}
		proto.Merge(&updatedRecord.endpoint, request.Endpoint)
		proto.Merge(&updatedRecord.details, request.Details)

		err = ds.db.Update(id, &updatedRecord)
		if err != nil {
			return 0, "", fmt.Errorf("Failed to update permanent service ID [%d] [%w]", id, err)
		}

		log.WithFields(logrus.Fields{"service_id": id}).Info("Permanent service updated")
		updatedID = id
		secret = record.secret
	}

	return updatedID, secret, nil
}

func (ds *DirectoryServer) dbRegister(request *cogmentAPI.RegisterRequest, token string) (ServiceID, string, error) {
	secret := utils.RandomString(secretLength)

	newRecord := DbRecord{
		lastHealthCheckTimestamp: 0,
		nbFailedHealthChecks:     0,
		permanent:                request.Permanent,
		authenticationToken:      token,
		secret:                   secret,
	}
	proto.Merge(&newRecord.endpoint, request.Endpoint)
	proto.Merge(&newRecord.details, request.Details)

	err := ds.db.Insert(&newRecord)
	if err != nil {
		return 0, "", err
	}

	log.WithFields(logrus.Fields{
		"service_id": newRecord.id,
		"type":       newRecord.details.Type,
		"permanent":  newRecord.permanent,
		"token":      len(token) > 0,
	}).Info("New registered service")
	log.WithFields(logrus.Fields{
		"service_id": newRecord.id,
		"token":      token,
		"secret":     secret,
		"endpoint":   newRecord.endpoint.String(),
		"details":    newRecord.details.String(),
	}).Debug("New service")

	if !healthCheck(&newRecord, newRecord.registerTimestamp) {
		log.WithFields(logrus.Fields{"service_id": newRecord.id}).Debug("Failed initial health check")
	}

	return newRecord.id, newRecord.secret, nil
}

func (ds *DirectoryServer) dbDeregister(request *cogmentAPI.DeregisterRequest, token string) error {
	id := ServiceID(request.ServiceId)

	record, err := ds.db.SelectByID(id)
	if err != nil {
		return err
	} else if record.authenticationToken != token || record.secret != request.Secret {
		log.WithFields(logrus.Fields{
			"service_id":     id,
			"db_token":       record.authenticationToken,
			"request_token":  token,
			"db_secret":      record.secret,
			"request_secret": request.Secret,
		}).Debug("Authentication failure")
		return fmt.Errorf("Authentication failure")
	}

	ds.db.Delete(id)
	log.WithFields(logrus.Fields{"service_id": id}).Info("Service deregistered")

	return nil
}

func (ds *DirectoryServer) dbInquire(request *cogmentAPI.InquireRequest, token string) (*sortableRecords, error) {
	var requestDetails *cogmentAPI.ServiceDetails
	var ids *[]ServiceID

	switch inquiry := request.Inquiry.(type) {
	case *cogmentAPI.InquireRequest_ServiceId:
		ids = &[]ServiceID{ServiceID(inquiry.ServiceId)}
	case *cogmentAPI.InquireRequest_Details:
		requestDetails = inquiry.Details
		selectionIds, err := ds.db.SelectByDetails(requestDetails)
		if err != nil {
			return nil, fmt.Errorf("Unable to inquire service details [%w]", err)
		}
		ids = &selectionIds
	case nil:
		return nil, fmt.Errorf("Empty request")
	default:
		return nil, fmt.Errorf("Unknown request type [%T]", inquiry)
	}

	var matches sortableRecords
	for _, id := range *ids {
		record, err := ds.db.SelectByID(id)
		if err != nil {
			if requestDetails == nil {
				log.WithFields(logrus.Fields{"service_id": id}).Debug("Inquiry for unknown service")
			}
			// else: the service was removed since we got the ID

		} else if record.authenticationToken != token {
			logAuth := log.WithFields(logrus.Fields{
				"service_id":    id,
				"db_token":      record.authenticationToken,
				"request_token": token,
			})

			if requestDetails != nil {
				logAuth = logAuth.WithFields(logrus.Fields{
					"request_details": requestDetails.String(),
				})
			}
			logAuth.Debug("Authentication failure")

		} else if record.nbFailedHealthChecks == 0 {
			matches = append(matches, record)
		}
	}

	sort.Sort(matches)

	return &matches, nil
}

// gRPC interface
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

			updatedServiceID, updatedServiceSecret, err := ds.dbUpdate(request, token)
			if err != nil {
				reply.Status = cogmentAPI.RegisterReply_FAILED
				reply.ErrorMsg = fmt.Sprintf("Could not update service in database [%s]", err.Error())
				log.WithFields(logrus.Fields{"error": err}).Debug("Failed to update database")
			} else if updatedServiceID != 0 {
				reply.Status = cogmentAPI.RegisterReply_OK
				reply.ServiceId = uint64(updatedServiceID)
				reply.Secret = updatedServiceSecret
			} else {

				newID, newSecret, err := ds.dbRegister(request, token)
				if err != nil {
					reply.Status = cogmentAPI.RegisterReply_FAILED
					reply.ErrorMsg = fmt.Sprintf("Could not add service to database [%s]", err.Error())
					log.WithFields(logrus.Fields{"error": err}).Debug("Failed to insert in database")
				} else {
					reply.Status = cogmentAPI.RegisterReply_OK
					reply.ServiceId = uint64(newID)
					reply.Secret = newSecret
				}
			}
		}

		err = inOutStream.Send(&reply)
		if err != nil {
			return err
		}
	}

	return nil
}

// gRPC interface
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

		reply := cogmentAPI.DeregisterReply{}
		id := request.ServiceId

		err = ds.dbDeregister(request, token)
		if err != nil {
			reply.Status = cogmentAPI.DeregisterReply_FAILED
			reply.ErrorMsg = fmt.Sprintf("Failed to deregister service ID [%d]: [%s]", id, err.Error())
			log.WithFields(logrus.Fields{"error": err}).Debug("Failed to deregister from database")
		} else {
			reply.Status = cogmentAPI.DeregisterReply_OK
		}

		err = inOutStream.Send(&reply)
		if err != nil {
			return err
		}
	}

	return nil
}

// gRPC interface
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

	matches, err := ds.dbInquire(request, token)
	if err != nil {
		return err
	}

	for _, record := range *matches {
		data := cogmentAPI.FullServiceData{
			Endpoint:  &record.endpoint,
			ServiceId: uint64(record.id),
			Details:   &record.details,
			Permanent: record.permanent,
		}
		reply := cogmentAPI.InquireReply{Data: &data}

		err := outStream.Send(&reply)
		if err != nil {
			return err
		}
	}

	return nil
}

// gRPC interface
func (ds *DirectoryServer) Version(context.Context, *cogmentAPI.VersionRequest) (*cogmentAPI.VersionInfo, error) {
	// TODO
	return &cogmentAPI.VersionInfo{}, nil
}

func RegisterDirectoryServer(grpcServer grpc.ServiceRegistrar) (*DirectoryServer, error) {
	server := DirectoryServer{}
	server.db = MakeMemoryDb()

	cogmentAPI.RegisterDirectorySPServer(grpcServer, &server)

	return &server, nil
}
