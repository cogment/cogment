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

func Sync(from backend.Backend, to backend.Backend) error {
	// Let's import everything from 'from' to 'to'
	// ATM we don't care about removing what's already in 'from' storage
	modelIDs, err := from.ListModels(-1, -1)
	if err != nil {
		return fmt.Errorf("Error while syncing models: %w", err)
	}
	for _, modelID := range modelIDs {
		err := SyncModel(from, to, modelID)
		if err != nil {
			return fmt.Errorf("Error while syncing models: %w", err)
		}
	}
	return nil
}

func SyncModel(from backend.Backend, to backend.Backend, modelID string) error {
	_, err := to.CreateOrUpdateModel(backend.ModelInfo{ModelID: modelID})
	if err != nil {
		return fmt.Errorf("Error while syncing model %q: %w", modelID, err)

	}
	modelVersionInfos, err := from.ListModelVersionInfos(modelID, -1, -1)
	if err != nil {
		return fmt.Errorf("Error while syncing model %q: %w", modelID, err)
	}
	for _, modelVersionInfo := range modelVersionInfos {
		modelVersionData, err := from.RetrieveModelVersionData(modelID, modelVersionInfo.Number)
		if err != nil {
			return fmt.Errorf("Error while syncing model %q: %w", modelID, err)
		}
		_, err = to.CreateOrUpdateModelVersion(modelID, backend.VersionInfoArgs{
			VersionNumber: modelVersionInfo.Number,
			Data:          modelVersionData,
			Archive:       modelVersionInfo.Archive,
			Metadata:      modelVersionInfo.Metadata,
		})
		if err != nil {
			return fmt.Errorf("Error while syncing model %q: %w", modelID, err)
		}
	}
	return nil
}
