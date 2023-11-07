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
	"strconv"
	"sync"
	"time"

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
const (
	unserviceableLoad   = 255
	acceptableLoadRange = 10
)

type ServerParameters struct {
	RegistrationLag uint
	LoadBalancing   bool
	CheckOnInquire  bool
	ForcePermanent  bool
}

type DirectoryServer struct {
	cogmentAPI.UnimplementedDirectorySPServer
	params ServerParameters
	db     *MemoryDB
}

type sortType int

const (
	none sortType = iota
	timestamp
	machineLoad
)

type resultRecords struct {
	List    []*DbRecord
	sortVar sortType
}

func (sr *resultRecords) Add(record *DbRecord) { sr.List = append(sr.List, record) }
func (sr *resultRecords) Len() int             { return len(sr.List) }
func (sr *resultRecords) Swap(i, j int)        { sr.List[i], sr.List[j] = sr.List[j], sr.List[i] }
func (sr *resultRecords) Less(i, j int) bool {
	switch sr.sortVar {
	case timestamp:
		// Sort in reverse: from most recent (larger number) to less recent (smaller number).
		// So the "Less" function effectively becomes a "More" function.
		return sr.List[i].RegisterTimestamp > sr.List[j].RegisterTimestamp

	case machineLoad:
		// From small loads to larger loads
		return sr.List[i].MachineLoad < sr.List[j].MachineLoad

	default:
		return sr.List[i].Sid < sr.List[j].Sid
	}
}
func (sr *resultRecords) SortByTime() {
	sr.sortVar = timestamp
	sort.Sort(sr)
}
func (sr *resultRecords) SortByLoad() {
	sr.sortVar = machineLoad
	sort.Sort(sr)
}
func (sr *resultRecords) Shuffle() { rand.Shuffle(sr.Len(), sr.Swap) }

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

	permanent := request.Permanent
	if ds.params.ForcePermanent {
		permanent = true
	}

	var updatedID ServiceID
	var secret string
	for _, id := range selectionIds {
		record, err := ds.db.SelectByID(id)
		if err != nil {
			continue
		}
		if !permanent {
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

	permanent := request.Permanent
	if ds.params.ForcePermanent {
		permanent = true
	}

	newRecord := DbRecord{
		LastHealthCheckTimestamp: 0,
		NbFailedHealthChecks:     0,
		Permanent:                permanent,
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

	if !healthCheck(&newRecord, newRecord.RegisterTimestamp, defaultHealthCheckNbRounds) {
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

func (ds *DirectoryServer) dbInquire(request *cogmentAPI.InquireRequest, token string) (*resultRecords, error) {
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

	matches := resultRecords{}
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
			matches.Add(record)
		}
	}

	if ds.params.CheckOnInquire {
		now := utils.Timestamp()
		var group sync.WaitGroup
		var mutex sync.Mutex
		healthyList := make([]*DbRecord, 0, matches.Len())

		group.Add(matches.Len())
		for _, record := range matches.List {
			go func(rec *DbRecord) {
				defer group.Done()

				if healthCheck(rec, now, 1) {
					mutex.Lock()
					defer mutex.Unlock()
					healthyList = append(healthyList, rec)
				}
			}(record)
		}

		group.Wait()
		matches.List = healthyList
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
			serviceType > cogmentAPI.ServiceType_DIRECTORY_SERVICE) &&
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

	for count := ds.params.RegistrationLag; matches.Len() == 0 && count > 0; count-- {
		time.Sleep(1 * time.Second)

		matches, err = ds.dbInquire(request, token)
		if err != nil {
			return err
		}
	}

	if matches.Len() == 0 {
		return nil
	}

	if !ds.params.LoadBalancing {
		matches.SortByTime()
	} else {
		matches.SortByLoad()

		for lastServiceableIndex := matches.Len() - 1; lastServiceableIndex >= 0; lastServiceableIndex-- {
			if matches.List[lastServiceableIndex].MachineLoad < unserviceableLoad {
				matches.List = matches.List[:lastServiceableIndex+1]
				break
			}
		}
		if matches.Len() == 0 {
			return nil
		}

		lowestLoad := matches.List[0].MachineLoad
		maxLowLoad := lowestLoad + acceptableLoadRange
		for lowLoadIndex := 1; lowLoadIndex < matches.Len(); lowLoadIndex++ {
			if matches.List[lowLoadIndex].MachineLoad > maxLowLoad {
				matches.List = matches.List[:lowLoadIndex]
				break
			}
		}

		matches.Shuffle()
	}

	for _, record := range matches.List {
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

// gRPC interface
func (ds *DirectoryServer) Status(_ context.Context, request *cogmentAPI.StatusRequest,
) (*cogmentAPI.StatusReply, error) {
	reply := cogmentAPI.StatusReply{}

	if len(request.Names) == 0 {
		return &reply, nil
	}
	reply.Statuses = make(map[string]string)

	// We purposefully don't scan for "*" ahead of time to allow explicit values before.
	all := false
	for _, name := range request.Names {
		if name == "*" {
			all = true
		}

		if all || name == "overall_load" {
			reply.Statuses["overall_load"] = "0"
		}

		if all || name == "nb_entries" {
			reply.Statuses["nb_entries"] = strconv.Itoa(len(ds.db.data))
		}

		if all || name == "load_balancing" {
			if ds.params.LoadBalancing {
				reply.Statuses["load_balancing"] = "enabled"
			} else {
				reply.Statuses["load_balancing"] = "disabled"
			}
		}

		if all {
			break
		}
	}

	return &reply, nil
}

func RegisterDirectoryServer(grpcServer grpc.ServiceRegistrar, persistenceFilename string,
	parameters ServerParameters) (*DirectoryServer, error) {
	server := DirectoryServer{
		db:     MakeMemoryDb(persistenceFilename),
		params: parameters,
	}

	cogmentAPI.RegisterDirectorySPServer(grpcServer, &server)

	return &server, nil
}
