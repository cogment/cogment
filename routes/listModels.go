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

package routes

import (
	"net/http"

	"github.com/cogment/model-registry/backend"
)

func listModels(b backend.Backend) http.HandlerFunc {
	return errorHandler(func(w http.ResponseWriter, r *http.Request) error {
		// Retrieve the models from db
		modelIDs, err := b.ListModels(0, -1)
		if err != nil {
			return err
		}

		models := []model{}
		for _, modelID := range modelIDs {
			modelLatestVersionInfo, err := b.RetrieveModelVersionInfo(modelID, -1)
			if _, ok := err.(*backend.UnknownModelVersionError); ok {
				models = append(models, model{
					ModelID: modelID,
				})
			} else if err != nil {
				return err
			} else {
				models = append(models, model{
					ModelID:                modelID,
					LatestVersionCreatedAt: modelLatestVersionInfo.CreatedAt,
					LatestVersionNumber:    modelLatestVersionInfo.Number,
				})
			}
		}

		// Encode the response
		w.WriteHeader(http.StatusOK)
		err = encodeJSONResponse(w, &models)
		if err != nil {
			return err
		}
		return nil
	})
}
