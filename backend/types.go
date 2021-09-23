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

type ModelInfo struct {
	ModelID  string            `json:"modelId" yaml:"model_id"`
	Metadata map[string]string `json:"metadata" `
}

// VersionInfo describes the informations (metadata) for a particular version of a model
type VersionInfo struct {
	ModelID   string            `json:"modelId" yaml:"model_id"`
	CreatedAt time.Time         `json:"createdAt" yaml:"created_at"`
	Number    int               `json:"number"`
	Archive   bool              `json:"archive"`
	Hash      string            `json:"hash"`
	Metadata  map[string]string `json:"metadata"`
}

type VersionInfoArgs struct {
	VersionNumber int               `json:"versionNumber" yaml:"version_number"`
	Data          []byte            `json:"data"`
	Archive       bool              `json:"archive"`
	Metadata      map[string]string `json:"metadata"`
}

// Backend defines the interface for a model registry backend
type Backend interface {
	Destroy()

	CreateOrUpdateModel(modelInfo ModelInfo) (ModelInfo, error)
	RetrieveModelInfo(modelID string) (ModelInfo, error)
	HasModel(modelID string) (bool, error)
	DeleteModel(modelID string) error
	ListModels(offset int, limit int) ([]string, error)

	CreateOrUpdateModelVersion(modelID string, versionInfoArgs VersionInfoArgs) (VersionInfo, error)
	RetrieveModelVersionInfo(modelID string, versionNumber int) (VersionInfo, error)
	RetrieveModelVersionData(modelID string, versionNumber int) ([]byte, error)
	DeleteModelVersion(modelID string, versionNumber int) error
	ListModelVersionInfos(modelID string, offset int, limit int) ([]VersionInfo, error)
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
