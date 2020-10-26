// Copyright 2020 Artificial Intelligence Redefined <dev+cogment@ai-r.com>
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

type ProjectConfig struct {
	Import       *Import
	Commands     map[string]string
	Trial        *Trial
	Environment  *Environment
	ActorClasses []*ActorClass `yaml:"actor_classes"`
	TrialParams  *TrialParams  `yaml:"trial_params"`
	ProjectName  string
	CliVersion   string
}

type Environment struct {
	ConfigType string `yaml:"config_type"`
}

type Import struct {
	Proto      []string
	ProtoAlias []string
	Python     []string
	Javascript []string
}

type Trial struct {
	ConfigType string   `yaml:"config_type"`
	PreHooks   []string `yaml:"pre_hooks"`
}

type Observation struct {
	Space        string
	Delta        string
	DeltaApplyFn *DeltaApplyFn `yaml:"delta_apply_fn"`
}

type DeltaApplyFn struct {
	Python     string
	Javascript string
}

type ActorClass struct {
	Id     string
	Action struct {
		Space string
	}
	Observation *Observation
	ConfigType  string `yaml:"config_type"`
}

type TrialParams struct {
	Environment struct {
		Endpoint string
		Config   map[string]interface{}
	}
	Actors []*Actor
}

type Actor struct {
	ActorClass string `yaml:"actor_class"`
	Endpoint   string
	Config     map[string]interface{}
}

func (p *ProjectConfig) CountActorsByActorClass(id string) (countAi, countHuman int) {
	countAi = 0
	countHuman = 0

	for _, actor := range p.TrialParams.Actors {
		if actor.ActorClass != id {
			continue
		}

		if actor.Endpoint == "human" {
			countHuman += 1
		} else {
			countAi += 1
		}
	}

	return countAi, countHuman
}

func (p *ProjectConfig) HasAiByActorClass(id string) bool {
	countAi, _ := p.CountActorsByActorClass(id)
	return countAi > 0
}

func (p *ProjectConfig) HasHumanByActorClass(id string) bool {
	_, countHuman := p.CountActorsByActorClass(id)
	return countHuman > 0
}
