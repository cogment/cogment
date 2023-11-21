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

package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/cogment/cogment/clients/control"
	"github.com/cogment/cogment/clients/directory"
	cogmentAPI "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/cogment/cogment/services/proxy/trialspec"
	"github.com/cogment/cogment/utils/endpoint"
	"golang.org/x/sync/errgroup"
)

type Controller struct {
	orchestratorEndpoint *endpoint.Endpoint
	directoryEndpoint    *endpoint.Endpoint
	directoryAuthToken   string
	trialSpecManager     *trialspec.Manager
}

func NewController(
	orchestratorEndpoint *endpoint.Endpoint,
	directoryEndpoint *endpoint.Endpoint,
	directoryAuthToken string,
	trialSpecManager *trialspec.Manager,
) (*Controller, error) {
	controller := Controller{
		orchestratorEndpoint: orchestratorEndpoint,
		directoryEndpoint:    directoryEndpoint,
		directoryAuthToken:   directoryAuthToken,
		trialSpecManager:     trialSpecManager,
	}
	return &controller, nil
}

func (controller *Controller) createClient(ctx context.Context, userID string) (*control.Client, error) {
	// TODO caching the inquiry result ?
	orchestratorEndpoints, err := directory.InquireEndpoint(
		ctx,
		controller.orchestratorEndpoint,
		controller.directoryEndpoint,
		controller.directoryAuthToken,
	)
	if err != nil {
		return nil, fmt.Errorf("Error while inquiring the orchestrator's lifecycle endpoint: %w", err)
	}

	orchestratorEndpoint := orchestratorEndpoints[0]
	if len(orchestratorEndpoints) > 1 {
		log.WithField("orchestrator", orchestratorEndpoint).Warning(
			"More that one matching orchestrator found, picking the first",
		)
	}

	return control.CreateClientWithInsecureEndpoint(
		orchestratorEndpoint,
		userID,
	)
}

func (controller *Controller) Destroy() {
	// NOTHING
}

func (controller *Controller) Spec() *trialspec.Manager {
	return controller.trialSpecManager
}

type TrialInfo struct {
	TrialID    string                `json:"trial_id" description:"The trial identifier"`
	EnvName    string                `json:"env_name,omitempty" description:"Name of the environment"`
	State      cogmentAPI.TrialState `json:"state,omitempty" description:"Current state of the trial"`
	Properties map[string]string     `json:"properties,omitempty"`
}

// WatchTrials retrieves the current list of active trials
func (controller *Controller) WatchTrials(ctx context.Context, out chan<- TrialInfo) error {
	client, err := controller.createClient(ctx, "web_proxy_controller/watch_active_trials")
	if err != nil {
		return err
	}
	observer := make(chan *cogmentAPI.TrialInfo)
	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		defer close(observer)
		return client.WatchTrials(ctx, observer)
	})
	group.Go(func() error {
		for trialInfo := range observer {
			activeTrialInfo := TrialInfo{
				TrialID:    trialInfo.TrialId,
				EnvName:    trialInfo.EnvName,
				State:      trialInfo.State,
				Properties: trialInfo.Properties,
			}
			select {
			case out <- activeTrialInfo:
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	})
	return group.Wait()
}

// GetActiveTrials retrieves the current list of active trials.
// In practice, it listens to trial activities until the timeout is reached and then return the active trials.
func (controller *Controller) GetActiveTrials(
	ctx context.Context,
	inactivityTimeout time.Duration,
) ([]TrialInfo, error) {
	client, err := controller.createClient(ctx, "web_proxy_controller/get_active_trials")
	if err != nil {
		return nil, err
	}

	observer := make(chan *cogmentAPI.TrialInfo)
	activeTrials := map[string]TrialInfo{}

	ctx, cancel := context.WithCancelCause(ctx)
	inactivityTimeoutCause := fmt.Errorf("inactivity timeout reached")

	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		defer close(observer)
		return client.WatchTrials(ctx, observer)
	})
	group.Go(func() error {
		for {
			select {
			case trialInfo := <-observer:
				if trialInfo.State == cogmentAPI.TrialState_ENDED {
					delete(activeTrials, trialInfo.TrialId)
				} else {
					activeTrials[trialInfo.TrialId] = TrialInfo{
						TrialID:    trialInfo.TrialId,
						EnvName:    trialInfo.EnvName,
						State:      trialInfo.State,
						Properties: trialInfo.Properties,
					}
				}
			case <-time.After(inactivityTimeout):
				cancel(inactivityTimeoutCause)
				return nil
			}
		}
	})

	err = group.Wait()

	if err != nil && context.Cause(ctx) != inactivityTimeoutCause {
		return nil, err
	}

	infoList := []TrialInfo{}
	for _, info := range activeTrials {
		infoList = append(infoList, info)
	}
	return infoList, nil
}

// StartTrial starts a new trial
func (controller *Controller) StartTrial(
	ctx context.Context,
	requestedTrialID string,
	params *cogmentAPI.TrialParams,
) (string, error) {
	// Forbidding any grpc endpoint
	if params.Environment != nil && params.Environment.Endpoint != "" {
		environmentEndpoint, err := endpoint.Parse(params.Environment.Endpoint)
		if err != nil {
			return "", err
		}
		if environmentEndpoint.Category == endpoint.GrpcEndpoint {
			return "", fmt.Errorf("web proxy controller doesn't support starting trial referring to grpc endpoints")
		}
	}
	if params.Datalog != nil && params.Datalog.Endpoint != "" {
		datalogEndpoint, err := endpoint.Parse(params.Datalog.Endpoint)
		if err != nil {
			return "", err
		}
		if datalogEndpoint.Category == endpoint.GrpcEndpoint {
			return "", fmt.Errorf("web proxy controller doesn't support starting trial referring to grpc endpoints")
		}
	}

	for _, actorParams := range params.Actors {
		if actorParams != nil && actorParams.Endpoint != "" {
			actorEndpoint, err := endpoint.Parse(actorParams.Endpoint)
			if err != nil {
				return "", err
			}
			if actorEndpoint.Category == endpoint.GrpcEndpoint {
				return "", fmt.Errorf("web proxy controller doesn't support starting trial referring to grpc endpoints")
			}
		}
	}

	client, err := controller.createClient(ctx, "web_proxy_controller/start_trial")
	if err != nil {
		return "", err
	}
	if err != nil {
		return "", err
	}

	return client.StartTrial(ctx, requestedTrialID, params)
}
