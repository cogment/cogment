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
	"strconv"

	"github.com/cogment/model-registry/backend"
	"github.com/gorilla/mux"
)

func getModelLatestVersion(backend backend.Backend) http.HandlerFunc {
	return errorHandler(func(w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		modelID := vars["model_id"]

		// Retrieve the version from db
		data, err := backend.RetrieveModelVersionData(modelID, -1)
		if err != nil {
			return NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Unable to retrieve the latest version of model %q: %s", modelID, err), err)
		}

		// Encode the response
		w.WriteHeader(http.StatusOK)
		err = encodeBlobResponse(w, data)
		if err != nil {
			return err
		}
		return nil
	})
}

func getModelGivenVersion(backend backend.Backend) http.HandlerFunc {
	return errorHandler(func(w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		modelID := vars["model_id"]
		// We can ignore the error as the validity of "model_version" is checked by the router itself
		versionNumber, _ := strconv.ParseInt(vars["model_version"], 10, 0)

		// Retrieve the version from db
		data, err := backend.RetrieveModelVersionData(modelID, int(versionNumber))
		if err != nil {
			return NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Unable to retrieve the version \"%d\" of model %q: %s", versionNumber, modelID, err), err)
		}

		// Encode the response
		w.WriteHeader(http.StatusOK)
		err = encodeBlobResponse(w, data)
		if err != nil {
			return err
		}
		return nil
	})
}
