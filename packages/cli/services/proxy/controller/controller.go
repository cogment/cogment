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
	"sync"
	"time"

	"github.com/cogment/cogment/clients/control"
	cogmentAPI "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/cogment/cogment/services/proxy/trialspec"
	"github.com/cogment/cogment/utils/endpoint"
)

type ActiveTrialInfo struct {
	TrialID    string                `json:"trial_id"`
	EnvName    string                `json:"env_name,omitempty"`
	State      cogmentAPI.TrialState `json:"state,omitempty"`
	Properties map[string]string     `json:"properties,omitempty"`
}

type Controller struct {
	orchestratorEndpoint *endpoint.Endpoint
	trialSpecManager     *trialspec.Manager
	workersContext       context.Context
	cancelWorkers        context.CancelFunc
	activeTrialsMutex    sync.RWMutex
	activeTrials         map[string]ActiveTrialInfo
}

func (controller *Controller) startWatchTrialsWorker(retryTimeout time.Duration) {
	observer := make(chan *cogmentAPI.TrialInfo)

	go func() {
		for {
			client, err := control.CreateClientWithInsecureEndpoint(
				controller.orchestratorEndpoint,
				"web_proxy_controller/watch_trials_worker",
			)
			if err != nil {
				log.WithField("error", err).Errorf("Error while connecting to the orchestrator, retrying in %v", retryTimeout)
			} else {
				err = client.WatchTrials(controller.workersContext, observer)
				if err == nil {
					log.Errorf("Unexpected watch trial termination, retrying in %v", retryTimeout)
				} else if err == context.Canceled {
					log.Debug("Watch trial canceled")
					break
				} else {
					log.WithField("error", err).Errorf("Error while watching trials, retrying in %v", retryTimeout)
				}
			}
			select {
			case <-time.After(retryTimeout):
				continue
			case <-controller.workersContext.Done():
				log.Debug("Watch trial canceled")
				break
			}
		}
	}()

	go func() {
		for trialInfo := range observer {
			controller.activeTrialsMutex.Lock()
			if trialInfo.State == cogmentAPI.TrialState_ENDED {
				delete(controller.activeTrials, trialInfo.TrialId)
			} else {
				controller.activeTrials[trialInfo.TrialId] = ActiveTrialInfo{
					TrialID:    trialInfo.TrialId,
					EnvName:    trialInfo.EnvName,
					State:      trialInfo.State,
					Properties: trialInfo.Properties,
				}
			}
			controller.activeTrialsMutex.Unlock()
		}
	}()
}

func NewController(orchestratorEndpoint *endpoint.Endpoint, trialSpecManager *trialspec.Manager) (*Controller, error) {
	workersContext, cancelWorkers := context.WithCancel(context.Background())
	controller := Controller{
		orchestratorEndpoint: orchestratorEndpoint,
		trialSpecManager:     trialSpecManager,
		workersContext:       workersContext,
		cancelWorkers:        cancelWorkers,
		activeTrialsMutex:    sync.RWMutex{},
		activeTrials:         map[string]ActiveTrialInfo{},
	}
	controller.startWatchTrialsWorker(2 * time.Second)
	return &controller, nil
}

func (controller *Controller) Destroy() {
	controller.cancelWorkers()
}

func (controller *Controller) Spec() *trialspec.Manager {
	return controller.trialSpecManager
}

// GetActiveTrials retrieves the current list of active trials
func (controller *Controller) GetActiveTrials() ([]ActiveTrialInfo, error) {
	infoList := []ActiveTrialInfo{}
	controller.activeTrialsMutex.RLock()
	defer controller.activeTrialsMutex.RUnlock()
	for _, info := range controller.activeTrials {
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

	client, err := control.CreateClientWithInsecureEndpoint(
		controller.orchestratorEndpoint,
		"web_proxy_controller/start_trial",
	)
	if err != nil {
		return "", err
	}

	return client.StartTrial(ctx, requestedTrialID, params)
}
