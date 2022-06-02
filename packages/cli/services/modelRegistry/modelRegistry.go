// Copyright 2022 AI Redefined Inc. <dev+cogment@ai-r.com>
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

package modelRegistry

import (
	"fmt"
	"net"

	"github.com/cogment/cogment/services/modelRegistry/backend"
	"github.com/cogment/cogment/services/modelRegistry/backend/fileSystem"
	"github.com/cogment/cogment/services/modelRegistry/backend/memoryCache"
	"github.com/cogment/cogment/services/modelRegistry/grpcservers"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Options struct {
	Port                     uint
	GrpcReflection           bool
	ArchiveDir               string
	CacheMaxItems            int
	SentVersionDataChunkSize int
}

var DefaultOptions = Options{
	Port:                     9002,
	GrpcReflection:           false,
	ArchiveDir:               ".cogment/model_registry",
	CacheMaxItems:            memoryCache.DefaultVersionCacheConfiguration.MaxItems,
	SentVersionDataChunkSize: 1024 * 1024 * 5,
}

func Run(options Options) error {
	log.Info("initializing the model registry...")
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", options.Port))
	if err != nil {
		return fmt.Errorf("unable to listen to tcp port %d: %v", options.Port, err)
	}
	var opts []grpc.ServerOption
	server := grpc.NewServer(opts...)
	modelRegistryServer, err := grpcservers.RegisterModelRegistryServer(
		server,
		options.SentVersionDataChunkSize,
	)
	if err != nil {
		return err
	}

	var archiveBackend backend.Backend
	var backend backend.Backend

	go func() {
		archiveBackend, err = fileSystem.CreateBackend(options.ArchiveDir)
		if err != nil {
			log.Fatalf("unable to create the archive filesystem backend: %v", err)
		}
		log.WithField("path", options.ArchiveDir).Info("filesystem backend created for archived model versions\n")

		versionCacheConfiguration := memoryCache.DefaultVersionCacheConfiguration
		backend, err = memoryCache.CreateBackend(versionCacheConfiguration, archiveBackend)
		if err != nil {
			log.Fatalf("unable to create the backend: %v", err)
		}

		modelRegistryServer.SetBackend(backend)
	}()

	defer func() {
		if backend != nil {
			backend.Destroy()
		}
		if archiveBackend != nil {
			archiveBackend.Destroy()
		}
	}()

	if options.GrpcReflection {
		reflection.Register(server)
		log.Printf("gRPC reflection registered")
	}

	log.WithField("port", options.Port).Info("model registry service starts...\n")
	err = server.Serve(listener)
	if err != nil {
		return fmt.Errorf("unexpected error while serving grpc services: %v", err)
	}

	return nil
}
