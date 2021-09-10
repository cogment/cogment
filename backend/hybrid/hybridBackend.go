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

package hybrid

import (
	"fmt"

	"github.com/cogment/cogment-model-registry/backend"
)

type hybridBackend struct {
	transient backend.Backend
	archive   backend.Backend
}

// CreateBackend creates a new backend with that store everything in a transient backend and archive versions in an archive backend
func CreateBackend(transient backend.Backend, archive backend.Backend) (backend.Backend, error) {
	backend := hybridBackend{
		transient: transient,
		archive:   archive,
	}

	err := backend.sync()
	if err != nil {
		return nil, err
	}
	return &backend, nil
}

// Destroy terminates the underlying storage
func (b *hybridBackend) Destroy() {
	b.transient.Destroy()
	b.archive.Destroy()
}

func (b *hybridBackend) sync() error {
	// Let's import everything from the archive to the transient storage
	// ATM we don't care about removing what's already in the transient storage
	modelIDs, err := b.archive.ListModels(-1, -1)
	if err != nil {
		return fmt.Errorf("Error while syncing models from archive to transient: %w", err)
	}
	for _, modelID := range modelIDs {
		err := b.syncModel(modelID)
		if err != nil {
			return fmt.Errorf("Error while syncing models from archive to transient: %w", err)
		}
	}
	return nil
}

func (b *hybridBackend) syncModel(modelID string) error {
	err := b.transient.CreateModel(modelID)
	if err != nil {
		if _, ok := err.(*backend.ModelAlreadyExistsError); !ok {
			return fmt.Errorf("Error while syncing model %q from archive to transient: %w", modelID, err)
		}
	}
	modelVersionInfos, err := b.archive.ListModelVersionInfos(modelID, -1, -1)
	if err != nil {
		return fmt.Errorf("Error while syncing model %q from archive to transient: %w", modelID, err)
	}
	for _, modelVersionInfo := range modelVersionInfos {
		modelVersionData, err := b.archive.RetrieveModelVersionData(modelID, modelVersionInfo.Number)
		if err != nil {
			return fmt.Errorf("Error while syncing model %q from archive to transient: %w", modelID, err)
		}
		_, err = b.transient.CreateOrUpdateModelVersion(modelID, modelVersionInfo.Number, modelVersionData, modelVersionInfo.Archive)
		if err != nil {
			return fmt.Errorf("Error while syncing model %q from archive to transient: %w", modelID, err)
		}
	}
	return nil
}

// CreateModel creates a model with a given unique id in the backend
func (b *hybridBackend) CreateModel(modelID string) error {
	err := b.transient.CreateModel(modelID)
	if err != nil {
		return err
	}
	err = b.archive.CreateModel(modelID)
	if err != nil {
		if _, ok := err.(*backend.ModelAlreadyExistsError); !ok {
			// Error while replicating model to the archive storage
			// Rollbacking
			// Explicitely ignoring error here, there's nothing we can do about it.
			_ = b.transient.DeleteModel(modelID)
		}
		return err
	}
	return nil
}

// HasModel check if a model exists
func (b *hybridBackend) HasModel(modelID string) (bool, error) {
	// Transient is kept in sync from the archive, so it should have everything
	return b.transient.HasModel(modelID)
}

// DeleteModel deletes a model with a given id from the storage
func (b *hybridBackend) DeleteModel(modelID string) error {
	err := b.transient.DeleteModel(modelID)
	if err != nil {
		return err
	}
	err = b.archive.DeleteModel(modelID)
	if err != nil {
		if _, ok := err.(*backend.UnknownModelError); !ok {
			// Error while removing model from the archive storage
			// Rollacking (we can't completely rollback non archived model versions)
			// Explicitely ignoring errors here, there's nothing we can do about it.
			_ = b.syncModel(modelID)
		}
		return err
	}
	return nil
}

// ListModels list models ordered by id from the given offset index, it returns at most the given limit number of models
func (b *hybridBackend) ListModels(offset int, limit int) ([]string, error) {
	// Transient is kept in sync from the archive, so it should have everything
	return b.transient.ListModels(offset, limit)
}

// CreateOrUpdateModelVersion creates and store a new version for a model and returns its info, including the version number
func (b *hybridBackend) CreateOrUpdateModelVersion(modelID string, versionNumber int, data []byte, archive bool) (backend.VersionInfo, error) {
	existingVersionInfo := backend.VersionInfo{}
	existingVersionInfoFound := false
	if versionNumber > 0 {
		versionInfo, err := b.RetrieveModelVersionInfo(modelID, versionNumber)
		if err == nil {
			existingVersionInfoFound = true
			existingVersionInfo = versionInfo
		}
	}
	versionInfo, err := b.transient.CreateOrUpdateModelVersion(modelID, versionNumber, data, archive)
	if err != nil {
		return backend.VersionInfo{}, err
	}
	if archive {
		_, err := b.archive.CreateOrUpdateModelVersion(modelID, versionInfo.Number, data, archive)
		if err != nil {
			// Rollbacking
			// explicitely ignoring error here, there's nothing we can do about it.
			_ = b.transient.DeleteModelVersion(modelID, versionNumber)
			return backend.VersionInfo{}, err
		}
	}
	if existingVersionInfoFound && !archive && existingVersionInfo.Archive {
		// Delete this model that is no longer in the archive
		// explicitely ignoring error here, there's nothing we can do about it.
		_ = b.archive.DeleteModelVersion(modelID, existingVersionInfo.Number)
	}
	return versionInfo, nil
}

// RetrieveModelVersionData retrieves a given model version data
func (b *hybridBackend) RetrieveModelVersionData(modelID string, versionNumber int) ([]byte, error) {
	// Transient is kept in sync from the archive, so it should have everything
	return b.transient.RetrieveModelVersionData(modelID, versionNumber)
}

// RetrieveModelVersionInfo retrieves a given model version info
func (b *hybridBackend) RetrieveModelVersionInfo(modelID string, versionNumber int) (backend.VersionInfo, error) {
	// Transient is kept in sync from the archive, so it should have everything
	return b.transient.RetrieveModelVersionInfo(modelID, versionNumber)
}

// DeleteModelVersion deletes a given model version
func (b *hybridBackend) DeleteModelVersion(modelID string, versionNumber int) error {
	err := b.transient.DeleteModelVersion(modelID, versionNumber)
	if err != nil {
		return err
	}
	err = b.archive.DeleteModelVersion(modelID, versionNumber)
	if err != nil {
		if _, ok := err.(*backend.UnknownModelVersionError); ok {
			// Ignoring unknowne model version errors on the archive side as it is suppoed to miss non archived model versions
			return nil
		}
		// Error while removing model version from the archive storage
		// Rollacking (we can't completely rollback non archived model versions)
		// explicitely ignoring error here, there's nothing we can do about it.
		_ = b.syncModel(modelID)
		return err
	}
	return nil
}

// ListModelVersionInfos list the versions info of a model from the latest to the earliest from the given offset index, it returns at most the given limit number of versions
func (b *hybridBackend) ListModelVersionInfos(modelID string, offset int, limit int) ([]backend.VersionInfo, error) {
	// Transient is kept in sync from the archive, so it should have everything
	return b.transient.ListModelVersionInfos(modelID, offset, limit)
}
