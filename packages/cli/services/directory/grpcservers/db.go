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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	cogmentAPI "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/cogment/cogment/utils"
)

type ServiceID uint64

type DbRecord struct {
	// Internal DB data (set/maintained by the DB)
	Sid               ServiceID // Redundant but simplifies code
	RegisterTimestamp uint64

	// Internal service data (set/maintained by the directory and not
	// consistently persisted)
	LastHealthCheckTimestamp uint64
	NbFailedHealthChecks     uint // since the last successful health check

	// External service data
	Permanent           bool
	AuthenticationToken string
	Secret              string

	Endpoint cogmentAPI.ServiceEndpoint
	Details  cogmentAPI.ServiceDetails
}

type MemoryDB struct {
	data  map[ServiceID]*DbRecord
	mutex sync.Mutex

	// We will need to add indexes when/if efficiency is needed
	// At least for the service type, maybe for special properties (class_name, implementation)

	persistenceFile *os.File
}

func MakeMemoryDb(filename string) *MemoryDB {
	// We may need to initialize indexes and other data here eventually
	md := MemoryDB{
		data:  make(map[ServiceID]*DbRecord),
		mutex: sync.Mutex{},
	}

	if len(filename) > 0 {
		file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0600)
		if err != nil {
			log.WithField("filename", filename).Warn("Could not open/create persistence file - ", err)
		} else {
			md.persistenceFile = file

			err = md.fromFile()
			if err != nil {
				log.WithField("filename", filename).Warn("Could not load persistence file - ", err)
			} else {
				log.Info("Loaded [", len(md.data), "] entries from persistence file [", filename, "]")
			}
		}
	} else {
		log.Info("No persistence file")
	}

	return &md
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

// Everything provided must match what is in the DB
func (md *MemoryDB) SelectByDetails(inquiryDetails *cogmentAPI.ServiceDetails) ([]ServiceID, error) {
	md.mutex.Lock()
	defer md.mutex.Unlock()

	var inquiryHasType bool = inquiryDetails.Type != cogmentAPI.ServiceType_UNKNOWN_SERVICE

	inquiryPropertyFilter := utils.NewPropertiesFilter(inquiryDetails.Properties)

	var result []ServiceID
	for id, record := range md.data {
		if inquiryHasType && record.Details.Type != inquiryDetails.Type {
			continue
		}

		if inquiryPropertyFilter.Selects(record.Details.Properties) {
			result = append(result, id)
		}
	}

	return result, nil
}

func (md *MemoryDB) Insert(record *DbRecord) error {
	md.mutex.Lock()
	defer md.mutex.Unlock()

	newID := md.getNewServiceID()
	record.Sid = newID
	record.RegisterTimestamp = utils.Timestamp()
	md.data[newID] = record

	err := md.toFile()
	if err != nil {
		log.Warn("Could not save new record to persistence file - ", err)
	}

	return nil
}

func (md *MemoryDB) Update(id ServiceID, record *DbRecord) error {
	md.mutex.Lock()
	defer md.mutex.Unlock()

	oldRecord, found := md.data[id]
	if !found {
		return fmt.Errorf("Unknown ID [%d] to update", id)
	}

	record.Sid = id
	record.RegisterTimestamp = oldRecord.RegisterTimestamp
	md.data[id] = record

	err := md.toFile()
	if err != nil {
		log.Warn("Could not update record in persistence file - ", err)
	}

	return nil
}

func (md *MemoryDB) Delete(id ServiceID) {
	md.mutex.Lock()
	defer md.mutex.Unlock()

	delete(md.data, id)

	err := md.toFile()
	if err != nil {
		log.Warn("Could not remove record in persistence file - ", err)
	}
}

func (md *MemoryDB) Save() error {
	md.mutex.Lock()
	defer md.mutex.Unlock()

	return md.toFile()
}

func (md *MemoryDB) fromFile() error {
	if md.persistenceFile != nil {
		_, err := md.persistenceFile.Seek(0, 0)
		if err != nil {
			return err
		}

		fileData, err := io.ReadAll(md.persistenceFile)
		if err != nil {
			return err
		}
		if len(fileData) == 0 {
			return nil
		}

		if len(md.data) != 0 {
			return fmt.Errorf("Can only read persistent data in a new database")
		}

		err = json.Unmarshal(fileData, &md.data)
		if err != nil {
			md.data = make(map[ServiceID]*DbRecord)
			return err
		}

		for _, record := range md.data {
			record.LastHealthCheckTimestamp = 0
			record.NbFailedHealthChecks = 1
		}
	}

	return nil
}

// This persistence system (i.e. re-save everything for every change) is workable only
// for a very small database with relatively rare changes (which is our use case).
func (md *MemoryDB) toFile() error {
	if md.persistenceFile != nil {
		fileData, err := json.Marshal(md.data)
		if err != nil {
			return err
		}

		endOffset, err := md.persistenceFile.WriteAt(fileData, 0)
		if err != nil {
			return err
		}

		err = md.persistenceFile.Truncate(int64(endOffset))
		if err != nil {
			return err
		}
	}

	return nil
}
