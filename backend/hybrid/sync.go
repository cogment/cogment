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
	"context"
	"fmt"

	"github.com/cogment/cogment-model-registry/backend"
	"golang.org/x/sync/errgroup"
)

func Sync(ctx context.Context, from backend.Backend, to backend.Backend) error {
	// Let's import everything from 'from' to 'to'
	// ATM we don't care about removing what's already in 'from' storage
	modelIDs, err := from.ListModels(-1, -1)
	if err != nil {
		return fmt.Errorf("Error while syncing models: %w", err)
	}
	g, ctx := errgroup.WithContext(ctx) // New error group and associated context
	for _, modelID := range modelIDs {
		modelID := modelID // Create a new 'modelID' that gets captured by the goroutine's closure https://golang.org/doc/faq#closures_and_goroutines
		g.Go(func() error {
			err := SyncModel(ctx, from, to, modelID)
			if err != nil {
				return fmt.Errorf("Error while syncing models: %w", err)
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}

func SyncModel(ctx context.Context, from backend.Backend, to backend.Backend, modelID string) error {
	_, err := to.CreateOrUpdateModel(backend.ModelInfo{ModelID: modelID})
	if err != nil {
		return fmt.Errorf("Error while syncing model %q: %w", modelID, err)

	}
	modelVersionInfos, err := from.ListModelVersionInfos(modelID, -1, -1)
	if err != nil {
		return fmt.Errorf("Error while syncing model %q: %w", modelID, err)
	}
	g, _ := errgroup.WithContext(ctx) // New error group and associated context
	for _, modelVersionInfo := range modelVersionInfos {
		modelVersionInfo := modelVersionInfo // Create a new 'modelID' that gets captured
		g.Go(func() error {
			modelVersionData, err := from.RetrieveModelVersionData(modelID, modelVersionInfo.VersionNumber)
			if err != nil {
				return fmt.Errorf("Error while syncing model %q: %w", modelID, err)
			}
			_, err = to.CreateOrUpdateModelVersion(modelID, backend.VersionArgs{
				VersionNumber:     modelVersionInfo.VersionNumber,
				CreationTimestamp: modelVersionInfo.CreationTimestamp,
				Archived:          modelVersionInfo.Archived,
				DataHash:          modelVersionInfo.DataHash,
				Data:              modelVersionData,
				UserData:          modelVersionInfo.UserData,
			})
			if err != nil {
				return fmt.Errorf("Error while syncing model %q: %w", modelID, err)
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}
