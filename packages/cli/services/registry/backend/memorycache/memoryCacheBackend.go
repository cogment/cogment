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

package memorycache

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"sync"
	"time"

	"github.com/cogment/cogment/services/registry/backend"
	lru "github.com/hashicorp/golang-lru"
)

type VersionCacheConfiguration struct {
	MaxItems int
}

var DefaultVersionCacheConfiguration = VersionCacheConfiguration{
	MaxItems: 100,
}

type memoryCacheBackend struct {
	archive                        backend.Backend
	modelsLatestVersionNumberMutex sync.RWMutex
	modelsLatestVersionNumber      map[string]uint
	versionCache                   *lru.Cache
	versionCacheConfiguration      VersionCacheConfiguration
}

type cachedVersion struct {
	ModelID           string
	VersionNumber     uint
	CreationTimestamp time.Time
	Archived          bool
	DataHash          string
	Data              []byte
	UserData          map[string]string
}

type memoryCacheKey struct {
	modelID       string
	versionNumber uint
}

func (b *memoryCacheBackend) deleteModelVersions(modelID string) {
	for _, key := range b.versionCache.Keys() {
		if key.(memoryCacheKey).modelID == modelID {
			b.versionCache.Remove(key)
		}
	}
}

func serializeCachedVersion(version cachedVersion) []byte {
	var buffer bytes.Buffer
	enc := gob.NewEncoder(&buffer)
	err := enc.Encode(version)
	if err != nil {
		panic(err)
	}
	return buffer.Bytes()
}

func deserializeCachedVersion(serializedVersion interface{}) cachedVersion {
	var version cachedVersion
	reader := bytes.NewReader(serializedVersion.([]byte))
	dec := gob.NewDecoder(reader)
	err := dec.Decode(&version)
	if err != nil {
		panic(err)
	}
	return version
}

func CreateBackend(
	versionCacheConfiguration VersionCacheConfiguration,
	archive backend.Backend,
) (backend.Backend, error) {
	cache, err := lru.New(versionCacheConfiguration.MaxItems)
	if err != nil {
		return nil, fmt.Errorf("unable to create memory cache backend: %w", err)
	}
	b := memoryCacheBackend{
		archive:                        archive,
		modelsLatestVersionNumberMutex: sync.RWMutex{},
		modelsLatestVersionNumber:      make(map[string]uint),
		versionCache:                   cache,
		versionCacheConfiguration:      versionCacheConfiguration,
	}
	return &b, nil

}

func (b *memoryCacheBackend) Destroy() {
}

func (b *memoryCacheBackend) retrieveCachedModelLatestVersionNumber(modelID string) (uint, bool) {
	b.modelsLatestVersionNumberMutex.RLock()
	defer b.modelsLatestVersionNumberMutex.RUnlock()
	latestVersionNumber, ok := b.modelsLatestVersionNumber[modelID]
	return latestVersionNumber, ok
}

func (b *memoryCacheBackend) deleteCachedModelLatestVersionNumber(modelID string, predicate func(uint) bool) {
	latestVersionNumber, ok := b.retrieveCachedModelLatestVersionNumber(modelID)
	if ok && predicate(latestVersionNumber) {
		b.modelsLatestVersionNumberMutex.Lock()
		defer b.modelsLatestVersionNumberMutex.Unlock()
		delete(b.modelsLatestVersionNumber, modelID)
	}
}

func (b *memoryCacheBackend) updateCachedModelLatestVersionNumber(modelID string, versionNumber uint) uint {
	latestVersionNumber, ok := b.retrieveCachedModelLatestVersionNumber(modelID)
	if !ok || latestVersionNumber < versionNumber {
		b.modelsLatestVersionNumberMutex.Lock()
		defer b.modelsLatestVersionNumberMutex.Unlock()
		b.modelsLatestVersionNumber[modelID] = versionNumber
		return versionNumber
	}
	return latestVersionNumber
}

func (b *memoryCacheBackend) resolveModelVersionNumbers(modelID string, versionNumbers []int) ([]uint, error) {
	resolvedVersionNumbers := []uint{}
	latestVersionNumber := uint(0) // We might not need it
	for _, versionNumber := range versionNumbers {
		// Check if it's a positive version number (no check in this case it'll be done later)
		if versionNumber >= 0 {
			resolvedVersionNumbers = append(resolvedVersionNumbers, uint(versionNumber))
			continue
		}

		// We now need `latestVersionNumber` to be set
		if latestVersionNumber == uint(0) {
			var ok bool
			latestVersionNumber, ok = b.retrieveCachedModelLatestVersionNumber(modelID)
			if !ok {
				archivedLatestVersionNumber, err := b.archive.RetrieveModelLatestVersionNumber(modelID)
				if err != nil {
					return nil, err
				}
				latestVersionNumber = archivedLatestVersionNumber

				for _, key := range b.versionCache.Keys() {
					if key.(memoryCacheKey).modelID == modelID {
						versionNumber := key.(memoryCacheKey).versionNumber
						if versionNumber > latestVersionNumber {
							latestVersionNumber = versionNumber
						}
					}
				}
			}
		}

		// Let's actually resolve this version number
		nthToLastIndex := uint(-versionNumber - 1)

		if nthToLastIndex > latestVersionNumber+1 {
			// Appending 0 to notify that it doesn't exist
			resolvedVersionNumbers = append(resolvedVersionNumbers, uint(0))
			continue
		}

		resolvedVersionNumbers = append(resolvedVersionNumbers, uint(latestVersionNumber-nthToLastIndex))
	}

	return resolvedVersionNumbers, nil
}

func (b *memoryCacheBackend) CreateOrUpdateModel(modelArgs backend.ModelInfo) (backend.ModelInfo, error) {
	modelInfo, err := b.archive.CreateOrUpdateModel(modelArgs)
	if err != nil {
		// Something wrong happened, let's just clear the cache
		b.deleteCachedModelLatestVersionNumber(modelInfo.ModelID, func(uint) bool { return true })
		b.deleteModelVersions(modelInfo.ModelID)
		return backend.ModelInfo{}, err
	}

	return modelInfo, nil
}

func (b *memoryCacheBackend) HasModel(modelID string) (bool, error) {
	return b.archive.HasModel(modelID)
}

func (b *memoryCacheBackend) RetrieveModelInfo(modelID string) (backend.ModelInfo, error) {
	return b.archive.RetrieveModelInfo(modelID)
}

func (b *memoryCacheBackend) RetrieveModelLatestVersionNumber(modelID string) (uint, error) {
	versionsCount, err := b.archive.RetrieveModelLatestVersionNumber(modelID)
	if err == nil {
		b.updateCachedModelLatestVersionNumber(modelID, versionsCount)
	}
	return versionsCount, err
}

func (b *memoryCacheBackend) DeleteModel(modelID string) error {
	err := b.archive.DeleteModel(modelID)
	if err != nil {
		return err
	}
	b.deleteCachedModelLatestVersionNumber(modelID, func(uint) bool { return true })
	b.deleteModelVersions(modelID)
	return nil
}

func (b *memoryCacheBackend) ListModels(offset int, limit int) ([]backend.ModelInfo, error) {
	return b.archive.ListModels(offset, limit)
}

func (b *memoryCacheBackend) retrieveCachedModelVersion(modelID string, versionNumber uint) (cachedVersion, bool) {
	// Is the version cached?
	key := memoryCacheKey{modelID: modelID, versionNumber: versionNumber}
	item, ok := b.versionCache.Get(key)
	if !ok {
		return cachedVersion{}, false
	}
	return deserializeCachedVersion(item), true
}

func (b *memoryCacheBackend) updateCachedModelVersion(modelID string, versionNumber uint, version cachedVersion) {
	key := memoryCacheKey{modelID: modelID, versionNumber: versionNumber}
	b.versionCache.Add(key, serializeCachedVersion(version))
}

func (b *memoryCacheBackend) deleteCachedModelVersion(modelID string, versionNumber uint) {
	key := memoryCacheKey{modelID: modelID, versionNumber: versionNumber}
	b.versionCache.Remove(key)
}

func (b *memoryCacheBackend) CreateOrUpdateModelVersion(
	modelID string,
	versionArgs backend.VersionArgs,
) (backend.VersionInfo, error) {
	// Let's compute the actual version number
	if versionArgs.VersionNumber == uint(0) {
		resolvedVersionNumbers, err := b.resolveModelVersionNumbers(modelID, []int{-1})
		if err != nil {
			return backend.VersionInfo{}, err
		}
		versionArgs.VersionNumber = resolvedVersionNumbers[0] + 1
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

func (b *memoryCacheBackend) doRetrieveModelVersionData(modelID string, versionNumber uint) ([]byte, error) {
	// Is the version cached?
	version, versionInCache := b.retrieveCachedModelVersion(modelID, versionNumber)
	if versionInCache {
		return version.Data, nil
	}
	versionData, err := b.archive.RetrieveModelVersionData(modelID, int(versionNumber))
	if err != nil {
		return nil, err
	}
	versionInfo, err := b.archive.RetrieveModelVersionInfo(modelID, int(versionNumber))
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

func (b *memoryCacheBackend) RetrieveModelVersionData(modelID string, versionNumber int) ([]byte, error) {
	resolvedVersionNumbers, err := b.resolveModelVersionNumbers(modelID, []int{versionNumber})
	if err != nil {
		return nil, err
	}
	resolvedVersionNumber := resolvedVersionNumbers[0]
	if resolvedVersionNumber == 0 {
		return nil, &backend.UnknownModelVersionError{ModelID: modelID, VersionNumber: versionNumber}
	}
	return b.doRetrieveModelVersionData(modelID, resolvedVersionNumbers[0])
}

func (b *memoryCacheBackend) doRetrieveModelVersionInfo(
	modelID string,
	versionNumber uint,
) (backend.VersionInfo, error) {
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
	versionInfo, err := b.archive.RetrieveModelVersionInfo(modelID, int(versionNumber))
	if err != nil {
		return backend.VersionInfo{}, err
	}
	b.updateCachedModelLatestVersionNumber(modelID, versionInfo.VersionNumber)
	return versionInfo, nil
}

func (b *memoryCacheBackend) RetrieveModelLastVersionInfo(modelID string) (backend.VersionInfo, error) {
	resolvedVersionNumbers, err := b.resolveModelVersionNumbers(modelID, []int{-1})
	if err != nil {
		return backend.VersionInfo{}, err
	}
	resolvedVersionNumber := resolvedVersionNumbers[0]
	if resolvedVersionNumber == 0 {
		return backend.VersionInfo{}, nil
	}
	versionInfo, err := b.doRetrieveModelVersionInfo(modelID, resolvedVersionNumber)
	if err != nil {
		if _, ok := err.(*backend.UnknownModelVersionError); ok {
			// Sending an error with the unresolved versionNumber for it to make sense to the user
			return backend.VersionInfo{}, &backend.UnknownModelVersionError{ModelID: modelID, VersionNumber: -1}
		}
		return backend.VersionInfo{}, err
	}
	return versionInfo, nil
}

func (b *memoryCacheBackend) RetrieveModelVersionInfo(modelID string, versionNumber int) (backend.VersionInfo, error) {
	resolvedVersionNumbers, err := b.resolveModelVersionNumbers(modelID, []int{versionNumber})
	if err != nil {
		return backend.VersionInfo{}, err
	}
	resolvedVersionNumber := resolvedVersionNumbers[0]
	if resolvedVersionNumber == 0 {
		return backend.VersionInfo{}, &backend.UnknownModelVersionError{ModelID: modelID, VersionNumber: versionNumber}
	}
	versionInfo, err := b.doRetrieveModelVersionInfo(modelID, resolvedVersionNumber)
	if err != nil {
		if _, ok := err.(*backend.UnknownModelVersionError); ok {
			// Sending an error with the unresolved versionNumber for it to make sense to the user
			return backend.VersionInfo{}, &backend.UnknownModelVersionError{ModelID: modelID, VersionNumber: versionNumber}
		}
		return backend.VersionInfo{}, err
	}
	return versionInfo, nil
}

func (b *memoryCacheBackend) doDeleteModelVersion(modelID string, versionNumber uint) error {
	// Delete from the archive model ignoring any error here
	_ = b.archive.DeleteModelVersion(modelID, int(versionNumber))
	b.deleteCachedModelVersion(modelID, versionNumber)
	// Delete the latest version number if it became "dirty"
	b.deleteCachedModelLatestVersionNumber(
		modelID,
		func(latestVersionNumber uint) bool { return versionNumber >= latestVersionNumber },
	)
	return nil
}

func (b *memoryCacheBackend) DeleteModelVersion(modelID string, versionNumber int) error {
	// Let's compute the actual version number
	resolvedVersionNumbers, err := b.resolveModelVersionNumbers(modelID, []int{versionNumber})
	if err != nil {
		return err
	}
	return b.doDeleteModelVersion(modelID, resolvedVersionNumbers[0])
}

func (b *memoryCacheBackend) ListModelVersionInfos(
	modelID string,
	initialVersionNumber uint,
	limit int,
) ([]backend.VersionInfo, error) {
	if initialVersionNumber == 0 {
		initialVersionNumber = 1
	}
	resolvedVersionNumbers, err := b.resolveModelVersionNumbers(modelID, []int{int(initialVersionNumber), -1})
	if err != nil {
		return nil, err
	}
	initialVersionNumber = resolvedVersionNumbers[0]
	latestVersionNumber := resolvedVersionNumbers[1]
	versions := []backend.VersionInfo{}
	for versionNumber := initialVersionNumber; versionNumber <= latestVersionNumber; versionNumber++ {
		versionInfo, err := b.RetrieveModelVersionInfo(modelID, int(versionNumber))
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
