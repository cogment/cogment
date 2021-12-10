// Copyright 2021 AI Redefined Inc. <dev+cogment@ai-r.com>
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

package memoryCache

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/cogment/cogment-model-registry/backend"
	ccache "github.com/karlseguin/ccache/v2"
)

type VersionCacheConfiguration struct {
	MaxSize         int
	VersionsToPrune int
	Expiration      time.Duration
}

var DefaultVersionCacheConfiguration = VersionCacheConfiguration{
	MaxSize:         1024 * 1024 * 1024, // 1GB
	VersionsToPrune: 50,
	Expiration:      1 * time.Hour,
}

type memoryCacheBackend struct {
	archive                        backend.Backend
	modelsLatestVersionNumberMutex sync.RWMutex
	modelsLatestVersionNumber      map[string]int
	versionCache                   *ccache.LayeredCache
	versionCacheConfiguration      VersionCacheConfiguration
}

type cachedVersion struct {
	ModelID           string
	VersionNumber     int
	CreationTimestamp time.Time
	Archived          bool
	DataHash          string
	Data              []byte
	UserData          map[string]string
}

type sizedBuffer struct {
	buffer []byte
}

func (s *sizedBuffer) Size() int64 {
	return int64(len(s.buffer))
}

func serializeCachedVersion(version cachedVersion) *sizedBuffer {
	var buffer bytes.Buffer
	enc := gob.NewEncoder(&buffer)
	err := enc.Encode(version)
	if err != nil {
		panic(err)
	}
	return &sizedBuffer{buffer: buffer.Bytes()}
}

func deserializeCachedVersion(serializedVersion interface{}) cachedVersion {
	var version cachedVersion
	reader := bytes.NewReader(serializedVersion.(*sizedBuffer).buffer)
	dec := gob.NewDecoder(reader)
	err := dec.Decode(&version)
	if err != nil {
		panic(err)
	}
	return version
}

func CreateBackend(versionCacheConfiguration VersionCacheConfiguration, archive backend.Backend) (backend.Backend, error) {
	b := memoryCacheBackend{
		archive:                        archive,
		modelsLatestVersionNumberMutex: sync.RWMutex{},
		modelsLatestVersionNumber:      make(map[string]int),
		versionCache:                   ccache.Layered(ccache.Configure().MaxSize(int64(versionCacheConfiguration.MaxSize)).ItemsToPrune(uint32(versionCacheConfiguration.VersionsToPrune))),
		versionCacheConfiguration:      versionCacheConfiguration,
	}

	return &b, nil
}

func (b *memoryCacheBackend) Destroy() {
}

func (b *memoryCacheBackend) retrieveCachedModelLatestVersionNumber(modelID string) (int, bool) {
	b.modelsLatestVersionNumberMutex.RLock()
	defer b.modelsLatestVersionNumberMutex.RUnlock()
	latestVersionNumber, ok := b.modelsLatestVersionNumber[modelID]
	return latestVersionNumber, ok
}

func (b *memoryCacheBackend) deleteCachedModelLatestVersionNumber(modelID string, predicate func(int) bool) {
	latestVersionNumber, ok := b.retrieveCachedModelLatestVersionNumber(modelID)
	if ok && predicate(latestVersionNumber) {
		b.modelsLatestVersionNumberMutex.Lock()
		defer b.modelsLatestVersionNumberMutex.Unlock()
		delete(b.modelsLatestVersionNumber, modelID)
	}
}

func (b *memoryCacheBackend) updateCachedModelLatestVersionNumber(modelID string, versionNumber int) int {
	latestVersionNumber, ok := b.retrieveCachedModelLatestVersionNumber(modelID)
	if !ok || latestVersionNumber < versionNumber {
		b.modelsLatestVersionNumberMutex.Lock()
		defer b.modelsLatestVersionNumberMutex.Unlock()
		b.modelsLatestVersionNumber[modelID] = versionNumber
		return versionNumber
	}
	return latestVersionNumber
}

func (b *memoryCacheBackend) retrieveModelLatestVersionNumber(modelID string) (int, error) {
	latestVersionNumber, ok := b.retrieveCachedModelLatestVersionNumber(modelID)
	if ok {
		return latestVersionNumber, nil
	}
	modelInfo, err := b.archive.RetrieveModelInfo(modelID)
	if err != nil {
		return 0, err
	}
	latestVersionNumber = modelInfo.LatestVersionNumber
	b.versionCache.ForEachFunc(modelID, func(versionNumberStr string, item *ccache.Item) bool {
		versionNumber, err := strconv.Atoi(versionNumberStr)
		if err != nil {
			panic(err)
		}
		if versionNumber > latestVersionNumber {
			latestVersionNumber = versionNumber
		}
		return true // Means that the iteration should continue
	})
	return latestVersionNumber, nil
}

func (b *memoryCacheBackend) CreateOrUpdateModel(modelInfo backend.ModelInfo) (backend.ModelInfo, error) {
	modelInfo, err := b.archive.CreateOrUpdateModel(modelInfo)
	if err != nil {
		// Something wrong happened, let's just clear the cache
		b.deleteCachedModelLatestVersionNumber(modelInfo.ModelID, func(int) bool { return true })
		b.versionCache.DeleteAll(modelInfo.ModelID)
		return backend.ModelInfo{}, err
	}
	modelInfo.LatestVersionNumber = b.updateCachedModelLatestVersionNumber(modelInfo.ModelID, modelInfo.LatestVersionNumber)

	return modelInfo, nil
}

func (b *memoryCacheBackend) HasModel(modelID string) (bool, error) {
	return b.archive.HasModel(modelID)
}

func (b *memoryCacheBackend) DeleteModel(modelID string) error {
	err := b.archive.DeleteModel(modelID)
	if err != nil {
		return err
	}
	b.deleteCachedModelLatestVersionNumber(modelID, func(int) bool { return true })
	b.versionCache.DeleteAll(modelID)
	return nil
}

func (b *memoryCacheBackend) ListModels(offset int, limit int) ([]string, error) {
	return b.archive.ListModels(offset, limit)
}

func (b *memoryCacheBackend) retrieveCachedModelVersion(modelID string, versionNumber int) (cachedVersion, bool) {
	versionNumberStr := fmt.Sprintf("%d", versionNumber)
	// Is the version cached?
	cachedItem := b.versionCache.Get(modelID, versionNumberStr)
	if cachedItem == nil || cachedItem.Expired() {
		return cachedVersion{}, false
	}
	return deserializeCachedVersion(cachedItem.Value()), true
}

func (b *memoryCacheBackend) updateCachedModelVersion(modelID string, versionNumber int, version cachedVersion) {
	versionNumberStr := fmt.Sprintf("%d", versionNumber)
	b.versionCache.Set(modelID, versionNumberStr, serializeCachedVersion(version), b.versionCacheConfiguration.Expiration)
}

func (b *memoryCacheBackend) deleteCachedModelVersion(modelID string, versionNumber int) {
	versionNumberStr := fmt.Sprintf("%d", versionNumber)
	b.versionCache.Delete(modelID, versionNumberStr)
}

func (b *memoryCacheBackend) CreateOrUpdateModelVersion(modelID string, versionArgs backend.VersionArgs) (backend.VersionInfo, error) {
	// Let's compute the actual version number
	if versionArgs.VersionNumber < 0 {
		latestVersionNumber, err := b.retrieveModelLatestVersionNumber(modelID)
		if err != nil {
			return backend.VersionInfo{}, err
		}
		versionArgs.VersionNumber = latestVersionNumber + 1
	}

	var versionInfo backend.VersionInfo
	if versionArgs.Archived {
		var err error
		versionInfo, err = b.archive.CreateOrUpdateModelVersion(modelID, versionArgs)
		if err != nil {
			return backend.VersionInfo{}, err
		}
	} else {
		versionInfo = backend.VersionInfo{
			ModelID:           modelID,
			VersionNumber:     versionArgs.VersionNumber,
			CreationTimestamp: versionArgs.CreationTimestamp,
			Archived:          versionArgs.Archived,
			DataHash:          versionArgs.DataHash,
			DataSize:          len(versionArgs.Data),
			UserData:          versionArgs.UserData,
		}
	}
	// Add the version to the cache
	b.updateCachedModelVersion(modelID, versionInfo.VersionNumber, cachedVersion{
		ModelID:           versionInfo.ModelID,
		VersionNumber:     versionInfo.VersionNumber,
		CreationTimestamp: versionInfo.CreationTimestamp,
		Archived:          versionInfo.Archived,
		DataHash:          versionInfo.DataHash,
		Data:              versionArgs.Data,
		UserData:          versionInfo.UserData,
	})
	// Update the latest version number if needed
	b.updateCachedModelLatestVersionNumber(modelID, versionInfo.VersionNumber)
	return versionInfo, nil
}

func (b *memoryCacheBackend) RetrieveModelVersionData(modelID string, versionNumber int) ([]byte, error) {
	// Let's compute the actual version number
	if versionNumber < 0 {
		latestVersionNumber, err := b.retrieveModelLatestVersionNumber(modelID)
		if err != nil {
			return nil, err
		}

		versionNumber = latestVersionNumber
	}
	// Is the version cached?
	version, versionInCache := b.retrieveCachedModelVersion(modelID, versionNumber)
	if versionInCache {
		return version.Data, nil
	}
	versionData, err := b.archive.RetrieveModelVersionData(modelID, versionNumber)
	if err != nil {
		return nil, err
	}
	versionInfo, err := b.archive.RetrieveModelVersionInfo(modelID, versionNumber)
	if err == nil {
		// Version info was properly retrieved, let's put everything in cache
		b.updateCachedModelVersion(modelID, versionInfo.VersionNumber, cachedVersion{
			ModelID:           version.ModelID,
			VersionNumber:     version.VersionNumber,
			CreationTimestamp: versionInfo.CreationTimestamp,
			Archived:          versionInfo.Archived,
			DataHash:          versionInfo.DataHash,
			Data:              versionData,
			UserData:          versionInfo.UserData,
		})
		b.updateCachedModelLatestVersionNumber(modelID, versionInfo.VersionNumber)
	}
	return versionData, nil
}

func (b *memoryCacheBackend) RetrieveModelVersionInfo(modelID string, versionNumber int) (backend.VersionInfo, error) {
	// Let's compute the actual version number
	if versionNumber < 0 {
		latestVersionNumber, err := b.retrieveModelLatestVersionNumber(modelID)
		if err != nil {
			return backend.VersionInfo{}, err
		}

		versionNumber = latestVersionNumber
	}
	// Is the version cached?
	version, versionInCache := b.retrieveCachedModelVersion(modelID, versionNumber)
	if versionInCache {
		return backend.VersionInfo{
			ModelID:           modelID,
			VersionNumber:     versionNumber,
			CreationTimestamp: version.CreationTimestamp,
			Archived:          version.Archived,
			DataHash:          version.DataHash,
			DataSize:          len(version.Data),
			UserData:          version.UserData,
		}, nil
	}
	versionInfo, err := b.archive.RetrieveModelVersionInfo(modelID, versionNumber)
	if err == nil {
		b.updateCachedModelLatestVersionNumber(modelID, versionInfo.VersionNumber)
	}
	return versionInfo, err
}

func (b *memoryCacheBackend) RetrieveModelInfo(modelID string) (backend.ModelInfo, error) {
	modelInfo, err := b.archive.RetrieveModelInfo(modelID)
	if err == nil {
		b.updateCachedModelLatestVersionNumber(modelID, modelInfo.LatestVersionNumber)
	}
	return modelInfo, err
}

func (b *memoryCacheBackend) DeleteModelVersion(modelID string, versionNumber int) error {
	// Let's compute the actual version number
	if versionNumber < 0 {
		latestVersionNumber, err := b.retrieveModelLatestVersionNumber(modelID)
		if err != nil {
			return err
		}

		versionNumber = latestVersionNumber
	}
	// Delete from the archive model ignoring any error here
	_ = b.archive.DeleteModelVersion(modelID, versionNumber)
	b.deleteCachedModelVersion(modelID, versionNumber)
	// Delete the latest version number if it became "dirty"
	b.deleteCachedModelLatestVersionNumber(modelID, func(latestVersionNumber int) bool { return versionNumber >= latestVersionNumber })
	return nil
}

func (b *memoryCacheBackend) ListModelVersionInfos(modelID string, initialVersionNumber int, limit int) ([]backend.VersionInfo, error) {
	latestVersionNumber, err := b.retrieveModelLatestVersionNumber(modelID)
	if err != nil {
		return nil, err
	}
	versions := []backend.VersionInfo{}
	if initialVersionNumber < 1 {
		initialVersionNumber = 1
	}
	for versionNumber := initialVersionNumber; versionNumber <= latestVersionNumber; versionNumber++ {
		versionInfo, err := b.RetrieveModelVersionInfo(modelID, versionNumber)
		if err != nil {
			// skip the version if it is unknown.
			if _, ok := err.(*backend.UnknownModelVersionError); ok {
				continue
			}
			return []backend.VersionInfo{}, err
		}
		versions = append(versions, versionInfo)
		if limit > 0 && len(versions) >= limit {
			break
		}
	}
	return versions, nil
}
