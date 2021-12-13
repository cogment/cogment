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

package backend

import (
	"fmt"
	"time"
)

// ModelInfo describes the informations (metadata) for a particular model
type ModelInfo struct {
	ModelID             string
	UserData            map[string]string
	LatestVersionNumber uint // Latest version number, 0 means that no version exists yet.
}

// VersionInfo describes the informations (metadata) for a particular version of a model
type VersionInfo struct {
	ModelID           string
	VersionNumber     uint
	CreationTimestamp time.Time
	Archived          bool
	DataHash          string
	DataSize          int
	UserData          map[string]string
}

// ModelArgs represents the arguments to create or update a model
type ModelArgs struct {
	ModelID  string
	UserData map[string]string
}

// VersionArgs represents the arguments to create or update a version
type VersionArgs struct {
	VersionNumber     uint // Set to 0 to create a new version
	CreationTimestamp time.Time
	Archived          bool
	DataHash          string
	Data              []byte
	UserData          map[string]string
}

// Backend defines the interface for a model registry backend
type Backend interface {
	Destroy()

	CreateOrUpdateModel(modelArgs ModelArgs) (ModelInfo, error)
	RetrieveModelInfo(modelID string) (ModelInfo, error)
	HasModel(modelID string) (bool, error)
	DeleteModel(modelID string) error
	ListModels(offset int, limit int) ([]string, error)

	CreateOrUpdateModelVersion(modelID string, versionArgs VersionArgs) (VersionInfo, error)
	RetrieveModelVersionInfo(modelID string, versionNumber int) (VersionInfo, error)
	RetrieveModelVersionData(modelID string, versionNumber int) ([]byte, error)
	DeleteModelVersion(modelID string, versionNumber int) error
	ListModelVersionInfos(modelID string, initialVersionNumber uint, limit int) ([]VersionInfo, error)
}

// UnknownModelError is raised when trying to operate on an unknown model
type UnknownModelError struct {
	ModelID string
}

func (e *UnknownModelError) Error() string {
	return fmt.Sprintf("no model %q found", e.ModelID)
}

// UnknownModelVersionError is raised when trying to operate on an unknown model version
type UnknownModelVersionError struct {
	ModelID       string
	VersionNumber int
}

func (e *UnknownModelVersionError) Error() string {
	if e.VersionNumber <= 0 {
		return fmt.Sprintf("model %q doesn't have any version yet", e.ModelID)
	}
	return fmt.Sprintf(`no version "%d" for model %q found`, e.VersionNumber, e.ModelID)
}
