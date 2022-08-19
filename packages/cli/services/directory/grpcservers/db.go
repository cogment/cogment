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
	"fmt"
	"sync"

	cogmentAPI "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/cogment/cogment/utils"
	"google.golang.org/protobuf/proto"
)

const secretLength = 5

type dbRecord struct {
	authenticationToken string
	secret              string

	endpoint cogmentAPI.ServiceEndpoint
	details  cogmentAPI.ServiceDetails
}

type ServiceID uint64

type MemoryDB struct {
	data  map[ServiceID]*dbRecord
	mutex sync.Mutex

	// We will need to add indexes when/if efficiency is needed
	// At least for the service type, maybe for special properties (class_name, implementation)
}

func MakeMemoryDb() *MemoryDB {
	// We may need to initialize indexes and other data here eventually
	return &MemoryDB{
		data:  make(map[ServiceID]*dbRecord),
		mutex: sync.Mutex{},
	}
}

func (md *MemoryDB) getNewServiceID() ServiceID {
	id := ServiceID(utils.RandomUint())
	for ; ; id = ServiceID(utils.RandomUint()) {
		if _, exist := md.data[id]; !exist {
			break
		}
	}

	return id
}

func (md *MemoryDB) SelectSecretsByID(id ServiceID) (string, string, error) {
	md.mutex.Lock()
	defer md.mutex.Unlock()

	record, exist := md.data[id]
	if !exist {
		return "", "", fmt.Errorf("Unknown service ID [%d]", uint64(id))
	}

	return record.secret, record.authenticationToken, nil
}

func (md *MemoryDB) SelectByID(id ServiceID) (*cogmentAPI.ServiceEndpoint, *cogmentAPI.ServiceDetails, error) {
	md.mutex.Lock()
	defer md.mutex.Unlock()

	record, exist := md.data[id]
	if !exist {
		return nil, nil, fmt.Errorf("Unknown service ID [%d]", uint64(id))
	}

	return &record.endpoint, &record.details, nil
}

func (md *MemoryDB) SelectByDetails(inquiryDetails *cogmentAPI.ServiceDetails) ([]ServiceID, error) {
	// Everything provided must match what is in the DB
	md.mutex.Lock()
	defer md.mutex.Unlock()

	var inquiryHasType bool = inquiryDetails.Type != cogmentAPI.ServiceType_UNKNOWN_SERVICE

	var result []ServiceID
	for id, record := range md.data {
		if inquiryHasType && record.details.Type != inquiryDetails.Type {
			continue
		}

		var propertyMatch = true
		for paramPropName, paramPropVal := range inquiryDetails.Properties {
			dbPropVal, exist := record.details.Properties[paramPropName]
			if !exist {
				propertyMatch = false
				break
			}
			if paramPropVal != dbPropVal {
				propertyMatch = false
				break
			}
		}

		if propertyMatch {
			result = append(result, id)
		}
	}

	return result, nil
}

func (md *MemoryDB) Insert(token string, endpoint *cogmentAPI.ServiceEndpoint, details *cogmentAPI.ServiceDetails,
) (ServiceID, string, error) {
	// Should we check for duplicates? What would be considered a duplicate?
	// Not to slow registration down, we could have recurring background duplication checks.
	md.mutex.Lock()
	defer md.mutex.Unlock()

	newID := md.getNewServiceID()
	secret := utils.RandomString(secretLength)

	newRecord := dbRecord{}
	newRecord.authenticationToken = token
	newRecord.secret = secret
	proto.Merge(&newRecord.endpoint, endpoint)
	proto.Merge(&newRecord.details, details)
	md.data[newID] = &newRecord // Lint complaining about copy locks

	return newID, secret, nil
}

func (md *MemoryDB) Delete(id ServiceID) {
	delete(md.data, id)
}

func (md *MemoryDB) Update(id ServiceID, secret string, token string, authenticationToken string,
	endpoint *cogmentAPI.ServiceEndpoint, details *cogmentAPI.ServiceDetails) error {
	panic("Update: Not implemented")
}
