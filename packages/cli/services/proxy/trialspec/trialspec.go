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

package trialspec

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"
	"google.golang.org/protobuf/reflect/protoreflect"
	"gopkg.in/yaml.v2"
)

type Import struct {
	Proto []string
}

type Trial struct {
	ConfigType string `yaml:"config_type"`
}

type Environment struct {
	ConfigType string `yaml:"config_type"`
}

type Space struct {
	Space string
}

type ActorClass struct {
	Name        string
	ConfigType  string `yaml:"config_type"`
	Action      Space
	Observation Space
}

type TrialSpec struct {
	Import       Import
	Trial        Trial
	Environment  Environment
	ActorClasses []*ActorClass `yaml:"actor_classes"`
}

type Manager struct {
	Spec       TrialSpec
	protoFiles linker.Files
}

func NewFromFile(specFile string) (*Manager, error) {
	specFileContent, err := os.ReadFile(specFile)
	if err != nil {
		return nil, err
	}

	spec := TrialSpec{}
	err = yaml.Unmarshal(specFileContent, &spec)
	if err != nil {
		return nil, err
	}

	specDir := path.Dir(specFile)

	compiler := protocompile.Compiler{
		Resolver: &protocompile.SourceResolver{
			ImportPaths: []string{specDir},
		},
	}

	protoFiles, err := compiler.Compile(context.Background(), spec.Import.Proto...)
	if err != nil {
		return nil, err
	}

	return &Manager{
		Spec:       spec,
		protoFiles: protoFiles,
	}, nil
}

func (tsm *Manager) FindMessageDescriptor(messageName string) (protoreflect.MessageDescriptor, error) {
	fullName := protoreflect.FullName(messageName)
	if !fullName.IsValid() {
		return nil, fmt.Errorf("%q is not a valid protobuf message name", messageName)
	}

	for _, protoFile := range tsm.protoFiles {
		descriptor := protoFile.FindDescriptorByName(fullName)
		if descriptor != nil {
			messageDescriptor, ok := descriptor.(protoreflect.MessageDescriptor)
			if !ok {
				return nil, fmt.Errorf("%q is not a message name", fullName)
			}
			return messageDescriptor, nil
		}
	}
	return nil, fmt.Errorf("Can't find descriptor %q", fullName)
}

func (tsm *Manager) NewMessage(messageName string) (*DynamicPbMessage, error) {
	descriptor, err := tsm.FindMessageDescriptor(messageName)
	if err != nil {
		return nil, err
	}

	message := NewMessageFromDescriptor(descriptor)
	return message, nil
}

func (tsm *Manager) NewTrialConfig() (*DynamicPbMessage, error) {
	messageType := tsm.Spec.Trial.ConfigType
	if messageType == "" {
		return nil, fmt.Errorf("No Trial config type defined")
	}
	return tsm.NewMessage(messageType)
}

func (tsm *Manager) NewEnvironmentConfig() (*DynamicPbMessage, error) {
	messageType := tsm.Spec.Environment.ConfigType
	if messageType == "" {
		return nil, fmt.Errorf("No Environment config type defined")
	}
	return tsm.NewMessage(messageType)

}

func (tsm *Manager) FindActorClass(actorClassName string) (*ActorClass, bool) {
	for _, actorClass := range tsm.Spec.ActorClasses {
		if actorClass.Name == actorClassName {
			return actorClass, true
		}
	}
	return nil, false
}

func (tsm *Manager) NewActorConfig(actorClassName string) (*DynamicPbMessage, error) {
	actorClass, ok := tsm.FindActorClass(actorClassName)
	if !ok {
		return nil, fmt.Errorf("%q is not a known actor class", actorClassName)
	}
	messageType := actorClass.ConfigType
	if messageType == "" {
		return nil, nil // nolint
	}
	return tsm.NewMessage(messageType)
}

func (tsm *Manager) NewAction(actorClassName string) (*DynamicPbMessage, error) {
	actorClass, ok := tsm.FindActorClass(actorClassName)
	if !ok {
		return nil, fmt.Errorf("%q is not a known actor class", actorClassName)
	}
	messageType := actorClass.Action.Space
	if messageType == "" {
		return nil, fmt.Errorf("No action space type defined for actor class %q", actorClassName)
	}
	return tsm.NewMessage(messageType)
}

func (tsm *Manager) NewObservation(actorClassName string) (*DynamicPbMessage, error) {
	actorClass, ok := tsm.FindActorClass(actorClassName)
	if !ok {
		return nil, fmt.Errorf("%q is not a known actor class", actorClassName)
	}
	messageType := actorClass.Observation.Space
	if messageType == "" {
		return nil, fmt.Errorf("No observation space type defined for actor class %q", actorClassName)
	}
	return tsm.NewMessage(messageType)
}
