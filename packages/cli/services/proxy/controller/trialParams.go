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
	"encoding/json"

	cogmentAPI "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/cogment/cogment/services/proxy/trialspec"
	"github.com/cogment/cogment/utils/endpoint"
	"gopkg.in/go-playground/validator.v8"
)

//nolint:lll
type TrialParams struct {
	Config        *json.RawMessage  `json:"config,omitempty" description:"Trial configuration, following the format defined in the spec file"`
	Properties    map[string]string `json:"properties,omitempty"`
	MaxSteps      uint32            `json:"max_steps,omitempty" description:"The maximum number of time steps (ticks) that the trial will run" default:"0"`
	MaxInactivity uint32            `json:"max_inactivity,omitempty" description:"The number of seconds of inactivity after which a trial will be hard terminated" default:"30"`
	// int64 / uint64 values are serialized/deserialized as string to be aligned with the behavior of protobuf
	NbBufferedTicks int64 `json:"nb_buffered_ticks,omitempty,string" description:"The number of ticks (steps) to buffer in the Orchestrator before sending the data to the datalog" validate:"omitempty,gt=1" format:"int64" default:"2"`

	Datalog     DatalogParams     `json:"datalog,omitempty"`
	Environment EnvironmentParams `json:"environment,omitempty" validate:"required"`

	Actors []*ActorParams `json:"actors,omitempty" validate:"gt=0"`
}

//nolint:lll
type DatalogParams struct {
	Endpoint      endpoint.Endpoint `json:"endpoint,omitempty" description:"Endpoint for the datalog service"`
	ExcludeFields []string          `json:"exclude_fields,omitempty" description:"List of fields to exclude from the data samples sent to the datalog service"`
}

//nolint:lll
type EnvironmentParams struct {
	Config         *json.RawMessage  `json:"config,omitempty" description:"Environment configuration, following the format defined in the spec file"`
	Name           string            `json:"name,omitempty" description:"The name of the environment" default:"env"`
	Endpoint       endpoint.Endpoint `json:"endpoint,omitempty" description:"Endpoint of the environment service" default:"cogment://discover/environment?__implementation=<implementation>"`
	Implementation string            `json:"implementation,omitempty" description:"The name of the implementation to run the environment." validate:"required"`
}

//nolint:lll
type ActorParams struct {
	Name                     string            `json:"name,omitempty" description:"The name of the actor (must be unique in the trial)" validate:"required"`
	ActorClass               string            `json:"actor_class,omitempty" description:"The name of the actor class" validate:"required"`
	Endpoint                 endpoint.Endpoint `json:"endpoint,omitempty" description:"Endpoint of the actor" default:"cogment://discover/actor?__actor_class=<actor_class>&__implementation=<implementation>"`
	Implementation           string            `json:"implementation,omitempty" description:"The name of the implementation to run this actor"`
	Config                   *json.RawMessage  `json:"config,omitempty" description:"Actor configuration, following the format defined in the spec file"`
	InitialConnectionTimeout float32           `json:"initial_connection_timeout,omitempty" description:"Maximum amount of time (in seconds) to wait for an actor to connect to a new trial, after which it is considered unavailable for the trial duration." default:"0" validate:"gte=0"`
	ResponseTimeout          float32           `json:"response_timeout,omitempty" description:"Maximum amount of time (in seconds) to wait for an actor to respond with an action after an observation is sent, after which it is considered unavailable." default:"0" validate:"gte=0"`
	Optional                 bool              `json:"optional,omitempty" description:"If set (true), the actor is optional." default:"false"`
	DefaultAction            *json.RawMessage  `json:"default_action,omitempty" description:"This is only relevant for optional actors (see optional). If set, and the actor is not available, the environment will receive this action."`
}

func (trialParams *TrialParams) FullyUnmarshal(spec *trialspec.Manager) (*cogmentAPI.TrialParams, error) {
	validate := validator.New(&validator.Config{TagName: "validate"})
	err := validate.Struct(trialParams)
	if err != nil {
		return nil, err
	}

	pbTrialParams := &cogmentAPI.TrialParams{}

	if trialParams.Config != nil {
		trialConfig, err := spec.NewTrialConfig()
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(*trialParams.Config, trialConfig); err != nil {
			return nil, err
		}

		serializedTrialConfig, err := trialConfig.MarshalProto()
		if err != nil {
			return nil, err
		}

		pbTrialParams.TrialConfig = &cogmentAPI.SerializedMessage{
			Content: serializedTrialConfig,
		}
	} else {
		pbTrialParams.TrialConfig = &cogmentAPI.SerializedMessage{}
	}
	pbTrialParams.Properties = trialParams.Properties
	pbTrialParams.MaxSteps = trialParams.MaxSteps
	pbTrialParams.MaxInactivity = trialParams.MaxInactivity
	pbTrialParams.NbBufferedTicks = trialParams.NbBufferedTicks

	if trialParams.Datalog.Endpoint.IsValid() {
		pbTrialParams.Datalog = &cogmentAPI.DatalogParams{
			Endpoint:      trialParams.Datalog.Endpoint.MarshalString(),
			ExcludeFields: trialParams.Datalog.ExcludeFields,
		}
	}

	pbTrialParams.Environment = &cogmentAPI.EnvironmentParams{}

	environmentConfig, err := spec.NewEnvironmentConfig()
	if err != nil {
		return nil, err
	}

	if trialParams.Environment.Config != nil {
		if err := json.Unmarshal(*trialParams.Environment.Config, environmentConfig); err != nil {
			return nil, err
		}
	}

	serializedEnvironmentConfig, err := environmentConfig.MarshalProto()
	if err != nil {
		return nil, err
	}

	pbTrialParams.Environment.Config = &cogmentAPI.SerializedMessage{
		Content: serializedEnvironmentConfig,
	}

	pbTrialParams.Environment.Name = trialParams.Environment.Name
	pbTrialParams.Environment.Endpoint = trialParams.Environment.Endpoint.MarshalString()
	pbTrialParams.Environment.Implementation = trialParams.Environment.Implementation

	pbTrialParams.Actors = make([]*cogmentAPI.ActorParams, 0)
	for _, rawActor := range trialParams.Actors {
		actor := &cogmentAPI.ActorParams{}

		actorConfig, err := spec.NewActorConfig(rawActor.ActorClass)
		if err != nil {
			return nil, err
		}

		// Actor config can be undefined
		if actorConfig != nil {
			if rawActor.Config != nil {
				if err := json.Unmarshal(*rawActor.Config, actorConfig); err != nil {
					return nil, err
				}
			}

			serializedActorConfig, err := actorConfig.MarshalProto()
			if err != nil {
				return nil, err
			}

			actor.Config = &cogmentAPI.SerializedMessage{
				Content: serializedActorConfig,
			}
		} else {
			actor.Config = &cogmentAPI.SerializedMessage{}
		}

		defaultAction, err := spec.NewAction(rawActor.ActorClass)
		if err != nil {
			return nil, err
		}

		if rawActor.DefaultAction != nil {
			if err := json.Unmarshal(*rawActor.DefaultAction, defaultAction); err != nil {
				return nil, err
			}
		}

		serializedDefaultAction, err := defaultAction.MarshalProto()
		if err != nil {
			return nil, err
		}

		actor.DefaultAction = &cogmentAPI.SerializedMessage{
			Content: serializedDefaultAction,
		}

		actor.Name = rawActor.Name
		actor.ActorClass = rawActor.ActorClass
		actor.Endpoint = rawActor.Endpoint.MarshalString()
		actor.Implementation = rawActor.Implementation
		actor.InitialConnectionTimeout = rawActor.InitialConnectionTimeout
		actor.ResponseTimeout = rawActor.ResponseTimeout
		actor.Optional = rawActor.Optional

		pbTrialParams.Actors = append(pbTrialParams.Actors, actor)
	}
	return pbTrialParams, nil
}
