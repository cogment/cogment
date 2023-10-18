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

package registry

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/cogment/cogment/clients/directory"
	cogmentAPI "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/cogment/cogment/services/registry/backend"
	"github.com/cogment/cogment/services/registry/backend/files"
	"github.com/cogment/cogment/services/registry/backend/memorycache"
	"github.com/cogment/cogment/services/registry/grpcservers"
	"github.com/cogment/cogment/services/utils"
	"golang.org/x/sync/errgroup"
)

// 4MB seems to be the maximum size gRPC can manage. To be safe we take 2MB.
const maxGrpcMessageLength int = 2 * 1024 * 1024

type Options struct {
	directory.RegistrationOptions
	Port                     uint
	GrpcReflection           bool
	ArchiveDir               string
	CacheMaxItems            int
	SentVersionDataChunkSize int
}

var DefaultOptions = Options{
	RegistrationOptions:      directory.DefaultRegistrationOptions,
	Port:                     9002,
	GrpcReflection:           false,
	ArchiveDir:               ".cogment/model_registry",
	CacheMaxItems:            memorycache.DefaultVersionCacheConfiguration.MaxItems,
	SentVersionDataChunkSize: maxGrpcMessageLength,
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
		archiveBackend, err = files.CreateBackend(options.ArchiveDir)
		if err != nil {
			log.Fatalf("unable to create the archive filesystem backend: %v", err)
		}
		log.WithField("path", options.ArchiveDir).Info("filesystem backend created for archived model versions\n")

		versionCacheConfiguration := memorycache.VersionCacheConfiguration{MaxItems: options.CacheMaxItems}
		backend, err = memorycache.CreateBackend(versionCacheConfiguration, archiveBackend)
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
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = utils.StopGrpcServer(stopCtx, server)
		return ctx.Err()
	})

	group.Go(func() error {
		return directory.ManageRegistration(
			ctx,
			port,
			cogmentAPI.ServiceEndpoint_GRPC,
			cogmentAPI.ServiceType_MODEL_REGISTRY_SERVICE,
			options.RegistrationOptions,
		)
	})

	return group.Wait()
}
