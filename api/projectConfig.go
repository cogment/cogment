// Copyright 2021 AI Redefined Inc. <dev+cogment@ai-r.com>
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
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/imdario/mergo"
	"github.com/jinzhu/copier"
	"github.com/markbates/pkger"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"

	"github.com/cogment/cogment-cli/helper"
)

var logger *zap.SugaredLogger

const SettingsFilenamePy = "cog_settings.py"
const SettingsFilenameJs = "CogSettings.ts"

var ValidProjectFilenames = []string{
	"cogment.yaml",
	"cogment.yml",
}

// ProtoAliasFromProtoPath convert the path to a .proto file to an unique alias like
// path/to/data.proto => path_to_data_pb
func ProtoAliasFromProtoPath(path string) string {
	fileName := strings.Split(path, ".")[0]
	return strings.ReplaceAll(fileName, "/", "_") + "_pb"
}

// TODO: have this look up the existance of web-client and web-client/tsconfig.json
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
func CreateDefaultProjectConfig() (*ProjectConfig, error) {
	yamlFile, err := pkger.Open("/api/default_cogment.yaml")
	if err != nil {
		// The default cogment.yaml file should be part of the package if it is not there, it's a huge problem
		yamlFile.Close()
		return nil, err
	}

	yamlFileStats, err := yamlFile.Stat()
	if err != nil {
		yamlFile.Close()
		return nil, err
	}

	yamlContent := make([]byte, yamlFileStats.Size())
	_, err = yamlFile.Read(yamlContent)
	if err != nil {
		yamlFile.Close()
		return nil, err
	}

	defaultConfig, err := createProjectConfigFromYamlContent(yamlContent)
	if err != nil {
		yamlFile.Close()
		return nil, err
	}

	return defaultConfig, nil
}

// ExtendDefaultProjectConfig extends the default project configuration with the given config
//
// the given config is left untouched.
func ExtendDefaultProjectConfig(config *ProjectConfig) (*ProjectConfig, error) {
	defaultConfig, err := CreateDefaultProjectConfig()
	if err != nil {
		return nil, err
	}

	extendedConfig := ProjectConfig{}
	err = copier.Copy(&extendedConfig, &config)
	if err != nil {
		return nil, err
	}

	err = mergo.Merge(&extendedConfig, defaultConfig)
	if err != nil {
		return nil, err
	}

	return &extendedConfig, nil
}

// GetProjectConfigPathFromProjectPath returns the path to an existing project configuration file in a given directory
func GetProjectConfigPathFromProjectPath(projectPath string) (string, error) {
	for _, projectConfigName := range ValidProjectFilenames {
		projectConfigPath := path.Join(projectPath, projectConfigName)
		if _, err := os.Stat(projectConfigPath); os.IsNotExist(err) {
			logger.Debugf("No configuration file found at %s", projectConfigPath)
			continue
		}
		logger.Debugf("Configuration file found at %s", projectConfigPath)
		return projectConfigPath, nil
	}

	return "", fmt.Errorf("unable to find any valid cogment project files at %s", projectPath)
}

// CreateProjectConfigFromProjectPath creates a new instance of ProjectConfig from a given directory.
// It will lookup `cogment.yaml`, `cogment.yml`
func CreateProjectConfigFromProjectPath(projectPath string) (*ProjectConfig, error) {
	projectConfigPath, err := GetProjectConfigPathFromProjectPath(projectPath)
	if err != nil {
		return nil, err
	}

	return CreateProjectConfigFromYaml(projectConfigPath)
}

// CreateProjectConfigFromYaml creates a new instance of ProjectConfig from a given `cogment.yaml` file
func CreateProjectConfigFromYaml(filename string) (*ProjectConfig, error) {
	yamlContent, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	loadedConfig, err := createProjectConfigFromYamlContent(yamlContent)
	if err != nil {
		return nil, err
	}
	loadedConfig.ProjectConfigPath = filename
	return ExtendDefaultProjectConfig(loadedConfig)
}

// ProjectConfig describes the root configuration of a cogment app, as loaded from a `cogment.yaml` file.
type ProjectConfig struct {
	ActorClasses      []*ActorClass `yaml:"actor_classes"`
	CliVersion        string
	Commands          map[string]string
	Components        ComponentsConfigurations
	Environment       *Environment
	Import            Import
	ProjectConfigPath string
	ProjectName       string
	Trial             *Trial
	TrialParams       *TrialParams `yaml:"trial_params"`
	Typescript        bool
	WebClient         bool
}

// ComponentsConfigurations describes the configuration of the cogment components
type ComponentsConfigurations struct {
	Orchestrator helper.VersionInfo
	Python       helper.VersionInfo
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
	Name   string
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
	Actors []*TrialActor
}

type TrialActor struct {
	Name           string
	ActorClass     string `yaml:"actor_class"`
	Endpoint       string
	Implementation string
	Config         map[string]interface{}
}

func (p *ProjectConfig) CountActorsByActorClass(name string) (countAi, countHuman int) {
	countAi = 0
	countHuman = 0

	for _, actor := range p.TrialParams.Actors {
		if actor.ActorClass != name {
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

var grpcHostRegex = regexp.MustCompile(`grpc://([A-Za-z0-9-]+):[0-9]+`)

// EnvironmentServiceName is the name used for environment service
var EnvironmentServiceName = "environment"

// ClientServiceName is the name used for client service
var ClientServiceName = "client"

// ComputeTrialActorServiceName computes the service name for a given trial actor
func ComputeTrialActorServiceName(actor *TrialActor) string {
	matches := grpcHostRegex.FindStringSubmatch(actor.Endpoint)
	if len(matches) != 0 && matches[1] != "" {
		// We assume the name of the service is the name of the grpc host
		return matches[1]
	}
	return ClientServiceName
}

// ClientActorServiceEndpoint is the endpoint used for client actors
var ClientActorServiceEndpoint = ClientServiceName

// ActorService represents an actor service:
//	- its unique name,
//	- its endpoint, and
//	- the actor implementations it hosts
type ActorService struct {
	Name            string
	Endpoint        string
	Implementations []ActorImplementation
}

// byServiceName implements sort.Interface for []ActorService based on the Name field.
type byServiceName []ActorService

func (a byServiceName) Len() int           { return len(a) }
func (a byServiceName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byServiceName) Less(i, j int) bool { return a[i].Name < a[j].Name }

// ActorImplementation represents an actor implementation:
//	- its unique name, and
//	- the actor classes it implements.
type ActorImplementation struct {
	Name         string
	ActorClasses []string
}

// byImplementationName implements sort.Interface for []ActorImplementation based on the Name field.
type byImplementationName []ActorImplementation

func (a byImplementationName) Len() int           { return len(a) }
func (a byImplementationName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byImplementationName) Less(i, j int) bool { return a[i].Name < a[j].Name }

func (p *ProjectConfig) listImplementations(service *ActorService) []ActorImplementation {
	var implMap = map[string]ActorImplementation{}
	for _, actor := range p.TrialParams.Actors {
		if actor.Endpoint == service.Endpoint {
			impl, implListed := implMap[actor.Implementation]
			if !implListed {
				implMap[actor.Implementation] = ActorImplementation{
					Name:         actor.Implementation,
					ActorClasses: []string{actor.ActorClass},
				}
			} else {
				classListed := false
				for _, class := range impl.ActorClasses {
					if class == actor.ActorClass {
						classListed = true
						break
					}
				}
				if !classListed {
					impl.ActorClasses = append(impl.ActorClasses, actor.ActorClass)
				}
			}
		}
	}
	var implList = []ActorImplementation{}
	for _, impl := range implMap {
		implList = append(implList, impl)
	}
	sort.Sort(byImplementationName(implList))
	return implList
}

// ListServiceActorServices lists the service actor services used by the default trial
func (p *ProjectConfig) ListServiceActorServices() []ActorService {
	var serviceMap = map[string]ActorService{}
	for _, actor := range p.TrialParams.Actors {
		serviceName := ComputeTrialActorServiceName(actor)
		if serviceName != ClientServiceName {
			// It's a service actor
			_, serviceListed := serviceMap[serviceName]
			if !serviceListed {
				service := ActorService{
					Name:            serviceName,
					Endpoint:        actor.Endpoint,
					Implementations: []ActorImplementation{},
				}
				service.Implementations = p.listImplementations(&service)
				serviceMap[serviceName] = service
			}
		}
	}
	var serviceList = []ActorService{}
	for _, service := range serviceMap {
		serviceList = append(serviceList, service)
	}
	sort.Sort(byServiceName(serviceList))
	return serviceList
}

// ListClientActorImplementations lists the client actor implementations from the default trial
func (p *ProjectConfig) ListClientActorImplementations() []ActorImplementation {
	service := ActorService{
		Name:            ClientServiceName,
		Endpoint:        ClientActorServiceEndpoint,
		Implementations: []ActorImplementation{},
	}
	return p.listImplementations(&service)
}

func init() {
	logger = helper.GetSugarLogger([]string{"api"})
}
