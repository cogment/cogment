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

package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListServiceActorServices(t *testing.T) {
	config := &ProjectConfig{
		ActorClasses: []*ActorClass{
			{Name: "dumb"},
			{Name: "dumber"},
		},
		TrialParams: &TrialParams{
			Actors: []*TrialActor{
				{Name: "one", ActorClass: "dumb", Implementation: "my_first_impl", Endpoint: "grpc://dumb-actor:9000"},
				{Name: "two", ActorClass: "dumb", Implementation: "my_first_impl", Endpoint: "grpc://dumb-actor:9000"},
				{Name: "three", ActorClass: "dumb", Implementation: "my_second_impl", Endpoint: "grpc://dumb-actor:9000"},
				{Name: "four", ActorClass: "dumber", Implementation: "my_third_impl", Endpoint: "grpc://dumber-actor:9000"},
			},
		},
	}

	actorServices := config.ListServiceActorServices()
	assert.Equal(t, []ActorService{
		{
			Name:     "dumb-actor",
			Endpoint: "grpc://dumb-actor:9000",
			Implementations: []ActorImplementation{
				{
					Name:         "my_first_impl",
					ActorClasses: []string{"dumb"},
				},
				{
					Name:         "my_second_impl",
					ActorClasses: []string{"dumb"},
				},
			},
		},
		{
			Name:     "dumber-actor",
			Endpoint: "grpc://dumber-actor:9000",
			Implementations: []ActorImplementation{
				{
					Name:         "my_third_impl",
					ActorClasses: []string{"dumber"},
				},
			},
		},
	}, actorServices)
}
