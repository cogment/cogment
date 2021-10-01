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
	"github.com/cogment/cogment-model-registry/backend"
)

type memoryCacheBackend struct {
	archive backend.Backend
}

func CreateBackend(archive backend.Backend) (backend.Backend, error) {
	b := memoryCacheBackend{
		archive: archive,
	}

	return &b, nil
}

func (b *memoryCacheBackend) Destroy() {
}

func (b *memoryCacheBackend) CreateOrUpdateModel(modelInfo backend.ModelInfo) (backend.ModelInfo, error) {
	return b.archive.CreateOrUpdateModel(modelInfo)
}

func (b *memoryCacheBackend) HasModel(modelID string) (bool, error) {
	return b.archive.HasModel(modelID)
}

func (b *memoryCacheBackend) DeleteModel(modelID string) error {
	return b.archive.DeleteModel(modelID)
}

func (b *memoryCacheBackend) ListModels(offset int, limit int) ([]string, error) {
	return b.archive.ListModels(offset, limit)
}

func (b *memoryCacheBackend) CreateOrUpdateModelVersion(modelID string, versionArgs backend.VersionArgs) (backend.VersionInfo, error) {
	return b.archive.CreateOrUpdateModelVersion(modelID, versionArgs)
}

func (b *memoryCacheBackend) RetrieveModelVersionData(modelID string, versionNumber int) ([]byte, error) {
	return b.archive.RetrieveModelVersionData(modelID, versionNumber)
}

func (b *memoryCacheBackend) RetrieveModelVersionInfo(modelID string, versionNumber int) (backend.VersionInfo, error) {
	return b.archive.RetrieveModelVersionInfo(modelID, versionNumber)
}

func (b *memoryCacheBackend) RetrieveModelInfo(modelID string) (backend.ModelInfo, error) {
	return b.archive.RetrieveModelInfo(modelID)
}

func (b *memoryCacheBackend) DeleteModelVersion(modelID string, versionNumber int) error {
	return b.archive.DeleteModelVersion(modelID, versionNumber)
}

func (b *memoryCacheBackend) ListModelVersionInfos(modelID string, offset int, limit int) ([]backend.VersionInfo, error) {
	return b.archive.ListModelVersionInfos(modelID, offset, limit)
}
