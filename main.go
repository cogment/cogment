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
	"net"

	"github.com/spf13/viper"

	"github.com/cogment/cogment-trial-datastore/backend/memoryBackend"
	"github.com/cogment/cogment-trial-datastore/grpcservers"
	"github.com/cogment/cogment-trial-datastore/version"
	log "github.com/sirupsen/logrus"
)

func main() {
	viper.AutomaticEnv()
	viper.SetDefault("PORT", 9000)
	viper.SetDefault("GRPC_REFLECTION", false)
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("MEMORY_STORAGE_MAX_SAMPLE_SIZE", memoryBackend.DefaultMaxSampleSize)
	viper.SetEnvPrefix("COGMENT_TRIAL_DATASTORE")

	logLevel, err := log.ParseLevel(viper.GetString("LOG_LEVEL"))
	if err != nil {
		expectedLevels := make([]string, 0)
		for _, level := range log.AllLevels {
			expectedLevels = append(expectedLevels, level.String())
		}
		log.Fatalf("invalid log level specified %q expecting one of %v", viper.GetString("LOG_LEVEL"), expectedLevels)
	}
	log.Infof("setting up log level to %q", logLevel.String())
	log.SetLevel(logLevel)

	backend, err := memoryBackend.CreateMemoryBackend(viper.GetUint32("MEMORY_STORAGE_MAX_SAMPLE_SIZE"))
	if err != nil {
		log.Fatalf("unable to create the memory backend: %v", err)
	}

	port := viper.GetInt("PORT")
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("unable to listen to tcp port %d: %v", port, err)
	}
	server := grpcservers.CreateGrpcServer(viper.GetBool("GRPC_REFLECTION"))
	err = grpcservers.RegisterTrialDatastoreServer(server, backend)
	if err != nil {
		log.Fatalf("%v", err)
	}
	err = grpcservers.RegisterDatalogServer(server, backend)
	if err != nil {
		log.Fatalf("%v", err)
	}
	log.WithField("port", port).WithField("version", version.Version).Info("Cogment Trial Datastore service starts...\n")
	err = server.Serve(listener)
	if err != nil {
		log.Fatalf("unexpected error while serving grpc services: %v", err)
	}
}
