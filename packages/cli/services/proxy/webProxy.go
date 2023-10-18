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

package proxy

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/cogment/cogment/clients/directory"
	cogmentAPI "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/cogment/cogment/services/proxy/actor"
	"github.com/cogment/cogment/services/proxy/controller"
	"github.com/cogment/cogment/services/proxy/grpcservers"
	"github.com/cogment/cogment/services/proxy/httpserver"
	"github.com/cogment/cogment/services/proxy/trialspec"
	"github.com/cogment/cogment/services/utils"
	"github.com/cogment/cogment/utils/endpoint"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type Options struct {
	directory.RegistrationOptions
	OrchestratorEndpoint *endpoint.Endpoint
	WebPort              uint
	SpecFile             string
	Implementation       string
	Secret               string
	GrpcPort             uint
	GrpcReflection       bool
}

var DefaultOptions = Options{
	RegistrationOptions:  directory.DefaultRegistrationOptions,
	OrchestratorEndpoint: endpoint.MustParse("cogment://discover"),
	WebPort:              8080,
	SpecFile:             "cogment.yaml",
	Implementation:       "web",
	Secret:               "web_proxy_secret",
	GrpcPort:             0,
	GrpcReflection:       false,
}

func Run(ctx context.Context, options Options) error {
	switch options.OrchestratorEndpoint.Details.Type {
	case cogmentAPI.ServiceType_UNKNOWN_SERVICE:
		options.OrchestratorEndpoint.SetServiceType(cogmentAPI.ServiceType_TRIAL_LIFE_CYCLE_SERVICE)
	case cogmentAPI.ServiceType_TRIAL_LIFE_CYCLE_SERVICE:
		break
	default:
		return fmt.Errorf(
			"Orchestrator endpoint [%s] doesn't have a valid service type",
			options.OrchestratorEndpoint,
		)
	}

	orchestratorEndpoints, err := directory.InquireEndpoint(
		ctx,
		options.OrchestratorEndpoint,
		options.DirectoryEndpoint,
		options.DirectoryAuthToken,
	)
	if err != nil {
		return err
	}

	orchestratorEndpoint := orchestratorEndpoints[0]
	if len(orchestratorEndpoints) > 1 {
		log.WithField("orchestrator", orchestratorEndpoint).Warning(
			"More that one matching orchestrator found, picking the first",
		)
	}

	// Build a trial spec manager
	trialSpecManager, err := trialspec.NewFromFile(options.SpecFile)
	if err != nil {
		return err
	}

	// Build the controller
	controller, err := controller.NewController(orchestratorEndpoint, trialSpecManager)
	if err != nil {
		return err
	}

	// Build the actor manager
	actorManager, err := actor.NewManager(trialSpecManager)
	if err != nil {
		return err
	}

	// Build the gRPC server
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", options.GrpcPort))
	if err != nil {
		return fmt.Errorf("unable to listen to tcp port %d: %v", options.GrpcPort, err)
	}
	grpcPort, err := utils.ExtractPort(listener.Addr().String())
	if err != nil {
		return err
	}
	grpcServer := utils.NewGrpcServer(options.GrpcReflection)

	err = grpcservers.RegisterServiceActorServer(
		grpcServer,
		actorManager,
	)
	if err != nil {
		return err
	}

	// Build the http server
	httpServer, err := httpserver.New(options.WebPort, actorManager, controller, options.Secret)
	if err != nil {
		return err
	}

	group, ctx := errgroup.WithContext(ctx)

	// Register the actor to the directory
	for _, actorClass := range actorManager.Spec().Spec.ActorClasses {
		properties := map[string]string{}
		for key, value := range options.RegistrationOptions.DirectoryRegistrationProperties {
			properties[key] = value
		}
		properties[endpoint.ImplementationPropertyName] = options.Implementation
		properties[endpoint.ActorClassPropertyName] = actorClass.Name
		group.Go(func() error {
			log.WithFields(logrus.Fields{
				"actor_class":                 properties[endpoint.ActorClassPropertyName],
				"implementation":              properties[endpoint.ImplementationPropertyName],
				"directory_endpoint":          options.DirectoryEndpoint,
				"directory_registration_host": options.DirectoryRegistrationHost,
			}).Debug("Registering the actor to the directory")
			return directory.ManageRegistration(
				ctx,
				grpcPort,
				cogmentAPI.ServiceEndpoint_GRPC,
				cogmentAPI.ServiceType_ACTOR_SERVICE,
				directory.RegistrationOptions{
					DirectoryEndpoint:               options.DirectoryEndpoint,
					DirectoryAuthToken:              options.DirectoryAuthToken,
					DirectoryRegistrationHost:       options.DirectoryRegistrationHost,
					DirectoryRegistrationProperties: properties,
				},
			)
		})
	}

	// Start the gRPC server
	group.Go(func() error {
		log.WithField("grpc_port", grpcPort).Info("grpc server listening")
		err = grpcServer.Serve(listener)
		if err != nil {
			return fmt.Errorf("unexpected error while serving grpc services: %v", err)
		}
		return nil
	})

	// Start the http server
	group.Go(func() error {
		log.WithField("web_port", options.WebPort).Info("http server listening")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("unexpected error while serving http routes: %v", err)
		}
		return nil
	})

	group.Go(func() error {
		<-ctx.Done()
		log.Info("Gracefully stopping")

		log.Debug("Destroying the actor manager")
		actorManager.Destroy()

		log.Debug("Destroying the controller")
		controller.Destroy()

		stopGroup, stopCtx := errgroup.WithContext(context.Background())
		stopGroup.Go(func() error {
			log.Debug("Stopping the http server")
			stopCtx, cancel := context.WithTimeout(stopCtx, 5*time.Second)
			defer cancel()
			return httpServer.Shutdown(stopCtx)
		})

		stopGroup.Go(func() error {
			log.Debug("Stopping the grpc server")
			stopCtx, cancel := context.WithTimeout(stopCtx, 5*time.Second)
			defer cancel()
			return utils.StopGrpcServer(stopCtx, grpcServer)
		})

		err = stopGroup.Wait()
		if err != nil {
			log.WithField("error", err).Warning("Error while stopping")
		}
		return ctx.Err()
	})

	return group.Wait()
}
