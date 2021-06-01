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

package deployment

import (
	"fmt"
	"github.com/cogment/cogment-cli/compose"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
)

type PortDefinition struct {
	Port   uint16 `json:"port"`
	Public bool   `json:"public"`
}

type Service struct {
	Image       string            `json:"image"`
	Environment map[string]string `json:"environment,omitempty"`
	Ports       []*PortDefinition `json:"ports,omitempty"`
	Entrypoint  string            `json:"entrypoint,omitempty"`
	Command     string            `json:"command,omitempty"`
	pushImage   bool
}

func NewService() *Service {
	return &Service{
		Environment: make(map[string]string),
	}
}

func (s *Service) setPushImage(value bool) {
	s.pushImage = value
}

func (s *Service) getPushImage() bool {
	return s.pushImage
}

type DeploymentManifest struct {
	Force    bool                `json:"force"`
	Services map[string]*Service `json:"services"`
}

func NewDeploymentManifest() *DeploymentManifest {
	return &DeploymentManifest{
		Services: make(map[string]*Service),
	}
}

func CreateManifestFromCompose(filename string, services []string) (*DeploymentManifest, error) {
	compose, err := filterServicesFromCompose(filename, services)
	if err != nil {
		return nil, err
	}

	m := NewDeploymentManifest()

	for name, svc := range compose.Services {

		s := NewService()

		s.Entrypoint = svc.XCogment.Entrypoint
		s.Command = svc.XCogment.Command

		//fmt.Println(prettyPrint(svc))

		// Parse environment
		for k, v := range svc.XCogment.Environment {
			s.Environment[k] = v
		}

		// Parse ports
		for _, v := range svc.XCogment.Ports {
			s.Ports = append(s.Ports, &PortDefinition{
				Port:   v.Port,
				Public: v.Public,
			})
		}

		// Parse image
		image := svc.Image
		if image == "" {
			return nil, fmt.Errorf("Service %s must define an image value in docker-compose", name)
		}
		s.Image = image

		if _, ok := svc.Build.(string); ok {
			s.setPushImage(true)
		} else if _, ok := svc.Build.(map[interface{}]interface{}); ok {
			s.setPushImage(true)
		}

		m.Services[name] = s
	}

	//fmt.Println(helper.PrettyPrint(m))
	return m, nil
}

// return only services with a x-cogment key
func filterServicesFromCompose(filename string, services []string) (*compose.Manifest, error) {
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}

	manifest := compose.Manifest{}
	err = yaml.Unmarshal(yamlFile, &manifest)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	//Check that all services exist
	for _, svc := range services {
		if _, ok := manifest.Services[svc]; ok == false {
			return nil, fmt.Errorf("%s doesn't exist in compose", svc)
		}
	}

	for name, svc := range manifest.Services {
		toDelete := true

		if svc.XCogment != nil {
			toDelete = false
		}

		if toDelete == false && len(services) > 0 {
			toDelete = true
			for _, svc := range services {
				if svc == name {
					toDelete = false
				}
			}
		}

		if toDelete == true {
			delete(manifest.Services, name)
		}

	}

	//fmt.Printf("%#v", manifest)
	//println(prettyPrint(manifest))
	//fmt.Printf("Format %T", manifest.Services["env"].Environment)

	return &manifest, nil
}
