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
	"net"

	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/cogment/cogment-model-registry/backend/db"
	"github.com/cogment/cogment-model-registry/backend/fs"
	"github.com/cogment/cogment-model-registry/backend/hybrid"
	"github.com/cogment/cogment-model-registry/grpcserver"
)

func main() {
	viper.AutomaticEnv()
	viper.SetDefault("PORT", 9000)
	viper.SetDefault("ARCHIVE_DIR", ".cogment_model_registry")
	viper.SetDefault("SENT_MODEL_VERSION_DATA_CHUNK_SIZE", 1024*1024*5) // Default chunk size is 5 MB
	viper.SetDefault("GRPC_REFLECTION", false)
	viper.SetEnvPrefix("COGMENT_MODEL_REGISTRY")

	dbBackend, err := db.CreateBackend()
	if err != nil {
		log.Fatalf("unable to create the database backend: %v", err)
	}
	defer dbBackend.Destroy()

	fsBackend, err := fs.CreateBackend(viper.GetString("ARCHIVE_DIR"))
	if err != nil {
		log.Fatalf("unable to create the archive filesystem backend: %v", err)
	}
	defer fsBackend.Destroy()

	backend, err := hybrid.CreateBackend(dbBackend, fsBackend)
	if err != nil {
		log.Fatalf("unable to create the backend: %v", err)
	}
	defer backend.Destroy()

	port := viper.GetInt("PORT")
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("unable to listen to tcp port %d: %v", port, err)
	}
	var opts []grpc.ServerOption
	server := grpc.NewServer(opts...)
	err = grpcserver.RegisterServer(server, backend, viper.GetInt("SENT_MODEL_VERSION_DATA_CHUNK_SIZE"))
	if err != nil {
		log.Fatalf("%v", err)
	}

	if viper.GetBool("GRPC_REFLECTION") {
		reflection.Register(server)
		log.Printf("gRPC reflection registered")
	}

	log.Printf("Cogment Model Registry service starts on port %d...\n", port)
	err = server.Serve(listener)
	if err != nil {
		log.Fatalf("unexpected error while serving grpc services: %v", err)
	}
}
