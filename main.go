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

	"github.com/cogment/cogment-activity-logger/backend"
	"github.com/cogment/cogment-activity-logger/grpcservers"
)

func main() {
	viper.AutomaticEnv()
	viper.SetDefault("PORT", 9000)
	viper.SetDefault("TRIAL_SAMPLE_CAPACITY", 1000)
	viper.SetDefault("GRPC_REFLECTION", false)
	viper.SetEnvPrefix("COGMENT_ACTIVITY_LOGGER")

	backend, err := backend.CreateMemoryBackend(viper.GetUint("TRIAL_SAMPLE_CAPACITY"))
	if err != nil {
		log.Fatalf("unable to create the memory backend: %v", err)
	}

	port := viper.GetInt("PORT")
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("unable to listen to tcp port %d: %v", port, err)
	}
	var opts []grpc.ServerOption
	server := grpc.NewServer(opts...)
	err = grpcservers.RegisterActivityLoggerServer(server, backend)
	if err != nil {
		log.Fatalf("%v", err)
	}
	err = grpcservers.RegisterLogExporterServer(server, backend)
	if err != nil {
		log.Fatalf("%v", err)
	}
	if viper.GetBool("GRPC_REFLECTION") {
		reflection.Register(server)
		log.Printf("gRPC reflection registered")
	}
	log.Printf("Cogment Activity Logger service starts on port %d...\n", port)
	err = server.Serve(listener)
	if err != nil {
		log.Fatalf("unexpected error while serving grpc services: %v", err)
	}
}
