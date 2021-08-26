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

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/spf13/viper"

	"github.com/cogment/model-registry/backend/db"
	"github.com/cogment/model-registry/backend/fs"
	"github.com/cogment/model-registry/backend/hybrid"
	"github.com/cogment/model-registry/routes"
)

func main() {
	viper.AutomaticEnv()
	viper.SetDefault("PORT", 80)
	viper.SetDefault("ARCHIVE_DIR", "/data")
	viper.SetEnvPrefix("COGMENT_MODEL_REGISTRY")

	dbBackend, err := db.CreateBackend()
	if err != nil {
		log.Printf("Unable to create the database backend: %v\n", err)
		os.Exit(-1)
	}
	defer dbBackend.Destroy()

	fsBackend, err := fs.CreateBackend(viper.GetString("ARCHIVE_DIR"))
	if err != nil {
		log.Printf("Unable to create the archive filesystem backend: %v\n", err)
		os.Exit(-1)
	}
	defer fsBackend.Destroy()

	backend, err := hybrid.CreateBackend(dbBackend, fsBackend)
	if err != nil {
		log.Printf("Unable to create the backend: %v\n", err)
		os.Exit(-1)
	}
	defer backend.Destroy()

	router := routes.CreateRouter(backend)
	port := viper.GetString("PORT")
	log.Printf("Model registry service starts on port %s...\n", port)
	err = http.ListenAndServe(fmt.Sprintf(":%s", port), handlers.CombinedLoggingHandler(os.Stdout, router))
	if err != nil {
		log.Printf("Error while running the http server: %v\n", err)
		os.Exit(-1)
	}
}
