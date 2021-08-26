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
	"fmt"
	"net/http"
	"time"

	"github.com/cogment/model-registry/backend"
	"github.com/gorilla/mux"
)

type model struct {
	ModelID                string    `json:"modelId"`
	LatestVersionCreatedAt time.Time `json:"latestVersionCreatedAt,omitempty"`
	LatestVersionNumber    int       `json:"latestVersionNumber,omitempty"`
}

type modelCreationRequest struct {
	ModelID string `json:"modelId,omitempty"`
}

func addModel(backend backend.Backend) http.HandlerFunc {
	return errorHandler(func(w http.ResponseWriter, r *http.Request) error {
		// Retrieve the parameters
		params := modelCreationRequest{}
		err := decodeJSONBody(w, r, &params)
		if err != nil {
			return err
		}

		// Add the model to the DB
		err = backend.CreateModel(params.ModelID)
		if err != nil {
			return NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Unable to create model %q: %s", params.ModelID, err), err)
		}

		// Encode the response
		w.WriteHeader(http.StatusCreated)
		err = encodeJSONResponse(w, &model{ModelID: params.ModelID})
		if err != nil {
			return err
		}
		return nil
	})
}

func addModelVersion(backend backend.Backend) http.HandlerFunc {
	return errorHandler(func(w http.ResponseWriter, r *http.Request) error {
		// Retrieve the query parameters
		vars := mux.Vars(r)
		modelID := vars["model_id"]
		archive := false
		if archiveQueryValue, isArchiveQueryDefined := vars["archive"]; isArchiveQueryDefined {
			archive = archiveQueryValue == "true"
		}

		// Retrieve the model blob
		modelBlob, err := decodeBlobBody(w, r)
		if err != nil {
			return err
		}

		// Add the model version to the DB
		version, err := backend.CreateOrUpdateModelVersion(modelID, -1, modelBlob, archive)
		if err != nil {
			return NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Unable to create a version for model %q: %s", modelID, err), err)
		}

		// Encode the response
		w.WriteHeader(http.StatusCreated)
		err = encodeJSONResponse(w, &version)
		if err != nil {
			return err
		}
		return nil
	})
}
