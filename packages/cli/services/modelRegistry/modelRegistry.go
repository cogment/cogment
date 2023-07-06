// Copyright 2023 AI Redefined Inc. <dev+cogment@ai-r.com>
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
	"context"
	"fmt"
	"net"

	"github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/cogment/cogment/services/modelRegistry/backend"
	"github.com/cogment/cogment/services/modelRegistry/backend/fileSystem"
	"github.com/cogment/cogment/services/modelRegistry/backend/memoryCache"
	"github.com/cogment/cogment/services/modelRegistry/grpcservers"
	"github.com/cogment/cogment/services/utils"
	"golang.org/x/sync/errgroup"
)

// 4MB seems to be the maximum size gRPC can manage
const maxGrpcMessageLength int = 4 * 1024 * 1024

type Options struct {
	utils.DirectoryRegistrationOptions
	Port                     uint
	GrpcReflection           bool
	ArchiveDir               string
	CacheMaxItems            int
	SentVersionDataChunkSize int
}

var DefaultOptions = Options{
	DirectoryRegistrationOptions: utils.DefaultDirectoryRegistrationOptions,
	Port:                         9002,
	GrpcReflection:               false,
	ArchiveDir:                   ".cogment/model_registry",
	CacheMaxItems:                memoryCache.DefaultVersionCacheConfiguration.MaxItems,
	SentVersionDataChunkSize:     maxGrpcMessageLength,
}

func Run(ctx context.Context, options Options) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", options.Port))
	if err != nil {
		return fmt.Errorf("unable to listen to tcp port %d: %v", options.Port, err)
	}
	port, err := utils.ExtractPort(listener.Addr().String())
	if err != nil {
		return err
	}
	server := utils.NewGrpcServer(options.GrpcReflection)

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

		versionCacheConfiguration := memoryCache.VersionCacheConfiguration{MaxItems: options.CacheMaxItems}
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

	log.WithField("port", port).Info("server listening")

	group, ctx := errgroup.WithContext(ctx)

	group.Go(func() error {
		err = server.Serve(listener)
		if err != nil {
			return fmt.Errorf("unexpected error while serving grpc services: %v", err)
		}
		return nil
	})

	group.Go(func() error {
		<-ctx.Done()
		log.Info("gracefully stopping the server")
		server.GracefulStop()
		return ctx.Err()
	})

	group.Go(func() error {
		return utils.ManageDirectoryRegistration(
			ctx,
			port,
			api.ServiceEndpoint_GRPC,
			api.ServiceType_MODEL_REGISTRY_SERVICE,
			options.DirectoryRegistrationOptions,
		)
	})

	return group.Wait()
}
