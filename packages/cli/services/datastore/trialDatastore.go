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

package datastore

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/cogment/cogment/clients/directory"
	"github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/cogment/cogment/services/datastore/backend"
	"github.com/cogment/cogment/services/datastore/backend/bolt"
	"github.com/cogment/cogment/services/datastore/backend/memory"
	"github.com/cogment/cogment/services/datastore/grpcservers"
	"github.com/cogment/cogment/services/utils"
	"golang.org/x/sync/errgroup"
)

type StorageType int

const (
	Memory StorageType = iota
	File
)

type Options struct {
	directory.RegistrationOptions
	Storage                     StorageType
	Port                        uint
	CustomListener              net.Listener
	GrpcReflection              bool
	MemoryStorageMaxSamplesSize uint32
	FileStoragePath             string
}

var DefaultOptions = Options{
	RegistrationOptions:         directory.DefaultRegistrationOptions,
	Storage:                     Memory,
	Port:                        9003,
	CustomListener:              nil,
	GrpcReflection:              false,
	MemoryStorageMaxSamplesSize: memory.DefaultMaxSampleSize,
	FileStoragePath:             ".cogment/trial_datastore.db",
}

func Run(ctx context.Context, options Options) error {
	var trialDatastoreBackend backend.Backend
	switch options.Storage {
	case File:
		log.WithField("path", options.FileStoragePath).Info("using a file storage backend")
		var err error
		trialDatastoreBackend, err = bolt.CreateBoltBackend(options.FileStoragePath)
		if err != nil {
			return fmt.Errorf("unable to create the bolt backend: %w", err)
		}
	case Memory:
		log.Info("using an in-memory storage")
		var err error
		trialDatastoreBackend, err = memory.CreateMemoryBackend(options.MemoryStorageMaxSamplesSize)
		if err != nil {
			return fmt.Errorf("unable to create the bolt backend: %w", err)
		}
	}

	server := utils.NewGrpcServer(options.GrpcReflection)
	err := grpcservers.RegisterTrialDatastoreServer(server, trialDatastoreBackend)
	if err != nil {
		return err
	}
	err = grpcservers.RegisterDatalogServer(server, trialDatastoreBackend)
	if err != nil {
		return err
	}

	var listener net.Listener
	var port uint
	if options.CustomListener != nil {
		listener = options.CustomListener
		port, err = utils.ExtractPort(listener.Addr().String())
		if err != nil {
			log.Info("server listening to custom listener")
		} else {
			log.WithField("port", port).Info("server listening")
		}
	} else {
		listener, err = net.Listen("tcp", fmt.Sprintf(":%d", options.Port))
		if err != nil {
			return fmt.Errorf("unable to listen to tcp port %d: %w", options.Port, err)
		}

		port, err = utils.ExtractPort(listener.Addr().String())
		if err != nil {
			return err
		}

		log.WithField("port", port).Info("server listening")
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
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = utils.StopGrpcServer(stopCtx, server)
		return ctx.Err()
	})

	group.Go(func() error {
		err := directory.ManageRegistration(
			ctx,
			port,
			api.ServiceEndpoint_GRPC,
			api.ServiceType_DATASTORE_SERVICE,
			options.RegistrationOptions,
		)
		if err != nil {
			return err
		}

		return directory.ManageRegistration(
			ctx,
			port,
			api.ServiceEndpoint_GRPC,
			api.ServiceType_DATALOG_SERVICE,
			options.RegistrationOptions,
		)
	})

	return group.Wait()
}
