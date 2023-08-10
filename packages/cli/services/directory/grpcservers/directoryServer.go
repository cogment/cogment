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

package grpcservers

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"sort"
	"time"

	cogmentAPI "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/cogment/cogment/utils"
	"github.com/cogment/cogment/utils/endpoint"

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
	registrationLag uint
	db              *MemoryDB
}

type sortableRecords []*DbRecord

func (sr sortableRecords) Len() int      { return len(sr) }
func (sr sortableRecords) Swap(i, j int) { sr[i], sr[j] = sr[j], sr[i] }

// sortableRecordsByRegistrationTimestamp implements `sort.Interface` for `sortableRecords`
// To sort in reverse: from most recent (larger number) to less recent (smaller number).
// So the "Less" function is actually a "More".
type sortableRecordsByRegistrationTimestamp struct{ sortableRecords }

func (sr sortableRecordsByRegistrationTimestamp) Less(i, j int) bool {
	return sr.sortableRecords[i].RegisterTimestamp > sr.sortableRecords[j].RegisterTimestamp
}

func (ds *DirectoryServer) dbUpdate(request *cogmentAPI.RegisterRequest, token string) (ServiceID, string, error) {
	if !request.Permanent {
		return 0, "", nil
	}

	selectionIds, err := ds.db.SelectByDetails(request.Details)
	if err != nil {
		return 0, "", fmt.Errorf("Unable to inquire service for duplicate verification [%w]", err)
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
		if !record.Permanent {
			continue
		}
		if record.AuthenticationToken != token {
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
			LastHealthCheckTimestamp: utils.Timestamp(), // This is an indication that a permanent has been updated
			NbFailedHealthChecks:     0,
			Permanent:                record.Permanent,
			AuthenticationToken:      record.AuthenticationToken,
			Secret:                   record.Secret,
		}
		proto.Merge(&updatedRecord.Endpoint, request.Endpoint)
		proto.Merge(&updatedRecord.Details, request.Details)

		err = ds.db.Update(id, &updatedRecord)
		if err != nil {
			return 0, "", fmt.Errorf("Failed to update permanent service ID [%d] [%w]", id, err)
		}

		log.WithFields(logrus.Fields{"service_id": id}).Info("Permanent service updated")
		updatedID = id
		secret = record.Secret
	}

	return updatedID, secret, nil
}

func (ds *DirectoryServer) dbRegister(request *cogmentAPI.RegisterRequest, token string) (ServiceID, string, error) {
	secret := utils.RandomString(secretLength)

	newRecord := DbRecord{
		LastHealthCheckTimestamp: 0,
		NbFailedHealthChecks:     0,
		Permanent:                request.Permanent,
		AuthenticationToken:      token,
		Secret:                   secret,
	}
	proto.Merge(&newRecord.Endpoint, request.Endpoint)
	proto.Merge(&newRecord.Details, request.Details)

	err := ds.db.Insert(&newRecord)
	if err != nil {
		return 0, "", err
	}

	log.WithFields(logrus.Fields{
		"service_id": newRecord.Sid,
		"type":       newRecord.Details.Type,
		"permanent":  newRecord.Permanent,
		"token":      len(token) > 0,
	}).Info("New registered service")
	log.WithFields(logrus.Fields{
		"service_id": newRecord.Sid,
		"token":      token,
		"secret":     secret,
		"endpoint":   newRecord.Endpoint.String(),
		"details":    newRecord.Details.String(),
	}).Debug("New service")

	if !healthCheck(&newRecord, newRecord.RegisterTimestamp) {
		log.WithFields(logrus.Fields{"service_id": newRecord.Sid}).Debug("Failed initial health check")
	}

	return newRecord.Sid, newRecord.Secret, nil
}

func (ds *DirectoryServer) dbDeregister(request *cogmentAPI.DeregisterRequest, token string) error {
	id := ServiceID(request.ServiceId)

	record, err := ds.db.SelectByID(id)
	if err != nil {
		return err
	} else if record.AuthenticationToken != token || record.Secret != request.Secret {
		log.WithFields(logrus.Fields{
			"service_id":     id,
			"db_token":       record.AuthenticationToken,
			"request_token":  token,
			"db_secret":      record.Secret,
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
	var orderBy string

	switch inquiry := request.Inquiry.(type) {
	case *cogmentAPI.InquireRequest_ServiceId:
		ids = &[]ServiceID{ServiceID(inquiry.ServiceId)}
		orderBy = endpoint.OrderByRegistration
	case *cogmentAPI.InquireRequest_Details:
		requestDetails = inquiry.Details

		// Deal with the "order_by" property
		var hasOrderBy bool
		if orderBy, hasOrderBy = requestDetails.Properties[endpoint.OrderByPropertyName]; !hasOrderBy {
			orderBy = endpoint.OrderByRegistration
		}
		delete(requestDetails.Properties, endpoint.OrderByPropertyName)

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

		} else if record.AuthenticationToken != token {
			logAuth := log.WithFields(logrus.Fields{
				"service_id":    id,
				"db_token":      record.AuthenticationToken,
				"request_token": token,
			})

			if requestDetails != nil {
				logAuth = logAuth.WithFields(logrus.Fields{
					"request_details": requestDetails.String(),
				})
			}
			logAuth.Debug("Authentication failure")

		} else if record.NbFailedHealthChecks == 0 {
			matches = append(matches, record)
		}
	}

	switch orderBy {
	case endpoint.OrderByRegistration:
		sort.Sort(sortableRecordsByRegistrationTimestamp{matches})
	case endpoint.OrderByRandom:
		rand.Shuffle(len(matches), matches.Swap)
	default:
		return nil, fmt.Errorf("Unknown value [%s] for property [%s]", orderBy, endpoint.OrderByPropertyName)
	}

	return &matches, nil
}

func (ds *DirectoryServer) SaveDatabase() error {
	return ds.db.Save()
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

	for count := ds.registrationLag; matches.Len() == 0 && count > 0; count-- {
		time.Sleep(1 * time.Second)

		matches, err = ds.dbInquire(request, token)
		if err != nil {
			return err
		}
	}

	for _, record := range *matches {
		data := cogmentAPI.FullServiceData{
			Endpoint:  &record.Endpoint,
			ServiceId: uint64(record.Sid),
			Details:   &record.Details,
			Permanent: record.Permanent,
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

func RegisterDirectoryServer(grpcServer grpc.ServiceRegistrar, regLag uint, persistenceFilename string,
) (*DirectoryServer, error) {
	server := DirectoryServer{
		db:              MakeMemoryDb(persistenceFilename),
		registrationLag: regLag,
	}

	cogmentAPI.RegisterDirectorySPServer(grpcServer, &server)

	return &server, nil
}
