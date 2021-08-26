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

// CreateRouter create a http handler handling all the model CRUD routes
func CreateRouter(backend backend.Backend) http.Handler {
	router := mux.NewRouter()

	router.Path("/").Methods(http.MethodGet, http.MethodHead).HandlerFunc(welcome())
	router.Path("/models").Methods(http.MethodGet, http.MethodHead).HandlerFunc(listModels(backend))
	router.Path("/models").Methods(http.MethodPost).HandlerFunc(addModel(backend))
	router.Path("/models/{model_id:[a-zA-Z][a-zA-Z0-9-_]*}").Methods(http.MethodGet, http.MethodHead).HandlerFunc(listModelVersions(backend))
	router.Path("/models/{model_id:[a-zA-Z][a-zA-Z0-9-_]*}").Queries("archive", "{archive:true|false}").Methods(http.MethodPost).HandlerFunc(addModelVersion(backend))
	router.Path("/models/{model_id:[a-zA-Z][a-zA-Z0-9-_]*}").Methods(http.MethodPost).HandlerFunc(addModelVersion(backend))
	router.Path("/models/{model_id:[a-zA-Z][a-zA-Z0-9-_]*}/latest").Methods(http.MethodGet, http.MethodHead).HandlerFunc(getModelLatestVersion(backend))
	router.Path("/models/{model_id:[a-zA-Z][a-zA-Z0-9-_]*}/{model_version:[0-9]+}").Methods(http.MethodGet, http.MethodHead).HandlerFunc(getModelGivenVersion(backend))

	router.NotFoundHandler = notFound()

	return router
}
