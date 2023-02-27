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
)

type ServiceID uint64

type DbRecord struct {
	// Internal DB data (set/maintained by the DB)
	id                ServiceID // Redundant but simplifies code
	registerTimestamp uint64

	// Internal service data (set/maintained by the directory)
	lastHealthCheckTimestamp uint64
	nbFailedHealthChecks     uint // since the last successful health check

	// External service data
	permanent           bool
	authenticationToken string
	secret              string

	endpoint cogmentAPI.ServiceEndpoint
	details  cogmentAPI.ServiceDetails
}

type MemoryDB struct {
	data  map[ServiceID]*DbRecord
	mutex sync.Mutex

	// We will need to add indexes when/if efficiency is needed
	// At least for the service type, maybe for special properties (class_name, implementation)
}

func MakeMemoryDb() *MemoryDB {
	// We may need to initialize indexes and other data here eventually
	return &MemoryDB{
		data:  make(map[ServiceID]*DbRecord),
		mutex: sync.Mutex{},
	}
}

func (md *MemoryDB) getNewServiceID() ServiceID {
	id := ServiceID(utils.RandomUint())
	for ; ; id = ServiceID(utils.RandomUint()) {
		if _, exist := md.data[id]; id != 0 && !exist {
			break
		}
	}

	return id
}

func (md *MemoryDB) SelectAllIDs() ([]ServiceID, error) {
	md.mutex.Lock()
	defer md.mutex.Unlock()

	var ids []ServiceID
	for id := range md.data {
		ids = append(ids, id)
	}

	return ids, nil
}

func (md *MemoryDB) SelectByID(id ServiceID) (*DbRecord, error) {
	md.mutex.Lock()
	defer md.mutex.Unlock()

	record, exist := md.data[id]
	if !exist {
		return nil, fmt.Errorf("Unknown service ID [%d]", uint64(id))
	}

	return record, nil
}

func (md *MemoryDB) SelectByDetails(inquiryDetails *cogmentAPI.ServiceDetails) ([]ServiceID, error) {
	// Everything provided must match what is in the DB
	md.mutex.Lock()
	defer md.mutex.Unlock()

	var inquiryHasType bool = inquiryDetails.Type != cogmentAPI.ServiceType_UNKNOWN_SERVICE

	inquiryPropertyFilter := utils.NewPropertiesFilter(inquiryDetails.Properties)

	var result []ServiceID
	for id, record := range md.data {
		if inquiryHasType && record.details.Type != inquiryDetails.Type {
			continue
		}

		if inquiryPropertyFilter.Selects(record.details.Properties) {
			result = append(result, id)
		}
	}

	return result, nil
}

func (md *MemoryDB) Insert(record *DbRecord) error {
	md.mutex.Lock()
	defer md.mutex.Unlock()

	newID := md.getNewServiceID()
	record.id = newID
	record.registerTimestamp = utils.Timestamp()
	md.data[newID] = record

	return nil
}

func (md *MemoryDB) Update(id ServiceID, record *DbRecord) error {
	md.mutex.Lock()
	defer md.mutex.Unlock()

	oldRecord, found := md.data[id]
	if !found {
		return fmt.Errorf("Unknown ID [%d] to update", id)
	}

	record.id = id
	record.registerTimestamp = oldRecord.registerTimestamp
	md.data[id] = record

	return nil
}

func (md *MemoryDB) Delete(id ServiceID) {
	delete(md.data, id)
}
