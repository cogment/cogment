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

package trialDatastore

import (
	"context"
	"fmt"
	"net"

	"github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/cogment/cogment/services/trialDatastore/backend"
	"github.com/cogment/cogment/services/trialDatastore/backend/boltBackend"
	"github.com/cogment/cogment/services/trialDatastore/backend/memoryBackend"
	"github.com/cogment/cogment/services/trialDatastore/grpcservers"
	"github.com/cogment/cogment/services/utils"
	"golang.org/x/sync/errgroup"
)

type StorageType int

const (
	Memory StorageType = iota
	File
)

type Options struct {
	utils.DirectoryRegistrationOptions
	Storage                     StorageType
	Port                        uint
	CustomListener              net.Listener
	GrpcReflection              bool
	MemoryStorageMaxSamplesSize uint32
	FileStoragePath             string
}

var DefaultOptions = Options{
	DirectoryRegistrationOptions: utils.DefaultDirectoryRegistrationOptions,
	Storage:                      Memory,
	Port:                         9003,
	CustomListener:               nil,
	GrpcReflection:               false,
	MemoryStorageMaxSamplesSize:  memoryBackend.DefaultMaxSampleSize,
	FileStoragePath:              ".cogment/trial_datastore.db",
}

func Run(ctx context.Context, options Options) error {
	var backend backend.Backend
	switch options.Storage {
	case File:
		log.WithField("path", options.FileStoragePath).Info("using a file storage backend")
		var err error
		backend, err = boltBackend.CreateBoltBackend(options.FileStoragePath)
		if err != nil {
			return fmt.Errorf("unable to create the bolt backend: %w", err)
		}
	case Memory:
		log.Info("using an in-memory storage")
		var err error
		backend, err = memoryBackend.CreateMemoryBackend(options.MemoryStorageMaxSamplesSize)
		if err != nil {
			return fmt.Errorf("unable to create the bolt backend: %w", err)
		}
	}

	server := utils.NewGrpcServer(options.GrpcReflection)
	err := grpcservers.RegisterTrialDatastoreServer(server, backend)
	if err != nil {
		return err
	}
	err = grpcservers.RegisterDatalogServer(server, backend)
	if err != nil {
		return err
	}
	var listener net.Listener
	if options.CustomListener != nil {
		listener = options.CustomListener
		log.Info("server listening")
	} else {
		var err error
		listener, err = net.Listen("tcp", fmt.Sprintf(":%d", options.Port))
		if err != nil {
			return fmt.Errorf("unable to listen to tcp port %d: %w", options.Port, err)
		}
		log.WithField("port", options.Port).Info("server listening")
	}

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
			options.Port,
			api.ServiceEndpoint_GRPC,
			api.ServiceType_DATASTORE_SERVICE,
			options.DirectoryRegistrationOptions,
		)
	})

	group.Go(func() error {
		return utils.ManageDirectoryRegistration(
			ctx,
			options.Port,
			api.ServiceEndpoint_GRPC,
			api.ServiceType_DATALOG_SERVICE,
			options.DirectoryRegistrationOptions,
		)
	})

	return group.Wait()
}
