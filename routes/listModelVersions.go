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
	"github.com/gorilla/mux"
)

func listModelVersions(backend backend.Backend) http.HandlerFunc {
	return errorHandler(func(w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		modelID := vars["model_id"]

		// Retrieve the models from db
		modelVersionInfos, err := backend.ListModelVersionInfos(modelID, 0, -1)
		if err != nil {
			return err
		}

		// Encode the response
		w.WriteHeader(http.StatusOK)
		err = encodeJSONResponse(w, &modelVersionInfos)
		if err != nil {
			return err
		}
		return nil
	})
}
