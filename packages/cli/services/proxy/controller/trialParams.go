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
)

// Wrapper around `cogmentAPI.TrialParams` to provide full JSON serialization/deserialization
type TrialParams struct {
	*cogmentAPI.TrialParams
	spec *trialspec.Manager
}

func NewTrialParams(
	spec *trialspec.Manager,
) *TrialParams {
	return &TrialParams{
		TrialParams: &cogmentAPI.TrialParams{},
		spec:        spec,
	}
}

type rawActorParams struct {
	Name                     string            `json:"name,omitempty"`
	ActorClass               string            `json:"actor_class,omitempty"`
	Endpoint                 endpoint.Endpoint `json:"endpoint,omitempty"`
	Implementation           string            `json:"implementation,omitempty"`
	Config                   *json.RawMessage  `json:"config,omitempty"`
	InitialConnectionTimeout float32           `json:"initial_connection_timeout,omitempty"`
	ResponseTimeout          float32           `json:"response_timeout,omitempty"`
	Optional                 bool              `json:"optional,omitempty"`
	DefaultAction            *json.RawMessage  `json:"default_action,omitempty"`
}

type rawDatalogParams struct {
	Endpoint      endpoint.Endpoint `json:"endpoint,omitempty"`
	ExcludeFields []string          `json:"exclude_fields,omitempty"`
}

type rawEnvironmentParams struct {
	Config         *json.RawMessage  `json:"config,omitempty"`
	Name           string            `json:"name,omitempty"`
	Endpoint       endpoint.Endpoint `json:"endpoint,omitempty"`
	Implementation string            `json:"implementation,omitempty"`
}

type rawTrialParams struct {
	Config        *json.RawMessage  `json:"config,omitempty"`
	Properties    map[string]string `json:"properties,omitempty"`
	MaxSteps      uint32            `json:"max_steps,omitempty"`
	MaxInactivity uint32            `json:"max_inactivity,omitempty"`
	// int64 / uint64 values are serialized/deserialized as string to be aligned with the behavior of protobuf
	NbBufferedTicks int64                `json:"nb_buffered_ticks,omitempty,string"`
	Datalog         rawDatalogParams     `json:"datalog,omitempty"`
	Environment     rawEnvironmentParams `json:"environment,omitempty"`
	Actors          []*rawActorParams    `json:"actors,omitempty"`
}

func (trialParams *TrialParams) UnmarshalJSON(b []byte) error {
	// Unmarshalling everything, stopping at the protobuf values
	var rawTrial rawTrialParams
	if err := json.Unmarshal(b, &rawTrial); err != nil {
		return err
	}

	if rawTrial.Config != nil {
		trialConfig, err := trialParams.spec.NewTrialConfig()
		if err != nil {
			return err
		}

		if err := json.Unmarshal(*rawTrial.Config, trialConfig); err != nil {
			return err
		}

		serializedTrialConfig, err := trialConfig.MarshalProto()
		if err != nil {
			return err
		}

		trialParams.TrialConfig = &cogmentAPI.SerializedMessage{
			Content: serializedTrialConfig,
		}
	} else {
		trialParams.TrialConfig = &cogmentAPI.SerializedMessage{}
	}
	trialParams.Properties = rawTrial.Properties
	trialParams.MaxSteps = rawTrial.MaxSteps
	trialParams.MaxInactivity = rawTrial.MaxInactivity
	trialParams.NbBufferedTicks = rawTrial.NbBufferedTicks

	if rawTrial.Datalog.Endpoint.IsValid() {
		trialParams.Datalog = &cogmentAPI.DatalogParams{
			Endpoint:      rawTrial.Datalog.Endpoint.MarshalString(),
			ExcludeFields: rawTrial.Datalog.ExcludeFields,
		}
	}

	trialParams.Environment = &cogmentAPI.EnvironmentParams{}

	environmentConfig, err := trialParams.spec.NewEnvironmentConfig()
	if err != nil {
		return err
	}

	if rawTrial.Environment.Config != nil {
		if err := json.Unmarshal(*rawTrial.Environment.Config, environmentConfig); err != nil {
			return err
		}
	}

	serializedEnvironmentConfig, err := environmentConfig.MarshalProto()
	if err != nil {
		return err
	}

	trialParams.Environment.Config = &cogmentAPI.SerializedMessage{
		Content: serializedEnvironmentConfig,
	}

	trialParams.Environment.Name = rawTrial.Environment.Name
	trialParams.Environment.Endpoint = rawTrial.Environment.Endpoint.MarshalString()
	trialParams.Environment.Implementation = rawTrial.Environment.Implementation

	trialParams.Actors = make([]*cogmentAPI.ActorParams, 0)
	for _, rawActor := range rawTrial.Actors {
		actor := &cogmentAPI.ActorParams{}

		actorConfig, err := trialParams.spec.NewActorConfig(rawActor.ActorClass)
		if err != nil {
			return err
		}

		// Actor config can be undefined
		if actorConfig != nil {
			if rawActor.Config != nil {
				if err := json.Unmarshal(*rawActor.Config, actorConfig); err != nil {
					return err
				}
			}

			serializedActorConfig, err := actorConfig.MarshalProto()
			if err != nil {
				return err
			}

			actor.Config = &cogmentAPI.SerializedMessage{
				Content: serializedActorConfig,
			}
		} else {
			actor.Config = &cogmentAPI.SerializedMessage{}
		}

		defaultAction, err := trialParams.spec.NewAction(rawActor.ActorClass)
		if err != nil {
			return err
		}

		if rawActor.DefaultAction != nil {
			if err := json.Unmarshal(*rawActor.DefaultAction, defaultAction); err != nil {
				return err
			}
		}

		serializedDefaultAction, err := defaultAction.MarshalProto()
		if err != nil {
			return err
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

		trialParams.Actors = append(trialParams.Actors, actor)
	}
	return nil
}
