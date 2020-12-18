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

import (
	"io/ioutil"
	"strings"

	"github.com/imdario/mergo"
	"github.com/jinzhu/copier"
	"github.com/markbates/pkger"
	"gopkg.in/yaml.v2"
)

// ProtoAliasFromProtoPath convert the path to a .proto file to an unique alias like
// path/to/data.proto => path_to_data_pb
func ProtoAliasFromProtoPath(path string) string {
	fileName := strings.Split(path, ".")[0]
	return strings.ReplaceAll(fileName, "/", "_") + "_pb"
}

func createProjectConfigFromYamlContent(yamlContent []byte) (*ProjectConfig, error) {
	config := ProjectConfig{}
	err := yaml.Unmarshal(yamlContent, &config)
	if err != nil {
		return nil, err
	}

	for _, protoPath := range config.Import.Proto {
		config.Import.ProtoAlias = append(config.Import.ProtoAlias, ProtoAliasFromProtoPath(protoPath))
	}

	return &config, nil
}

// CreateDefaultProjectConfig creates a project with the defaults defined in "/api/default_cogment.yaml"
func CreateDefaultProjectConfig() *ProjectConfig {
	yamlFile, err := pkger.Open("/api/default_cogment.yaml")
	if err != nil {
		// The default cogment.yaml file should be part of the package if it is not there, it's a huge problem
		panic(err)
	}
	defer yamlFile.Close()

	yamlFileStats, err := yamlFile.Stat()
	if err != nil {
		panic(err)
	}

	yamlContent := make([]byte, yamlFileStats.Size())
	yamlFile.Read(yamlContent)
	defaultConfig, err := createProjectConfigFromYamlContent(yamlContent)
	if err != nil {
		panic(err)
	}

	return defaultConfig
}

// ExtendDefaultProjectConfig extends the default project configuration with the given config
//
// the given config is left untouched.
func ExtendDefaultProjectConfig(config *ProjectConfig) *ProjectConfig {
	defaultConfig := CreateDefaultProjectConfig()
	extendedConfig := ProjectConfig{}
	copier.Copy(&extendedConfig, &config)
	mergo.Merge(&extendedConfig, defaultConfig)
	return &extendedConfig
}

// CreateProjectConfigFromYaml creates a new instance of ProjectConfig from a given `cogment.yaml` file
func CreateProjectConfigFromYaml(filename string) (*ProjectConfig, error) {
	yamlContent, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	loadedCondfig, err := createProjectConfigFromYamlContent(yamlContent)
	if err != nil {
		return nil, err
	}
	return ExtendDefaultProjectConfig(loadedCondfig), nil
}

// ProjectConfig describes the root configuration of a cogment app, as loaded from a `cogment.yaml` file.
type ProjectConfig struct {
	Components   ComponentsConfigurations
	Import       Import
	Commands     map[string]string
	Trial        *Trial
	Environment  *Environment
	ActorClasses []*ActorClass `yaml:"actor_classes"`
	TrialParams  *TrialParams  `yaml:"trial_params"`
	ProjectName  string
	CliVersion   string
}

// ComponentsConfigurations describes the configuration of the cogment components
type ComponentsConfigurations struct {
	Orchestrator OrchestratorConfiguration
}

// OrchestratorConfiguration is the configuration of the orchestrator
// Image is its docker image
// Version is the version of its docker image
type OrchestratorConfiguration struct {
	Image   string
	Version string
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
