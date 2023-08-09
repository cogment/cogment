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
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const testDataDir string = "../../../testdata"

func TestNewTrialSpec(t *testing.T) {
	tsm, err := NewFromFile(path.Join(testDataDir, "cogment.yaml"))
	assert.NoError(t, err)
	assert.Equal(t, tsm.Spec.Trial.ConfigType, "flames.TrialConfig")
	assert.Equal(t, tsm.Spec.Environment.ConfigType, "foo.Data")
	assert.Len(t, tsm.Spec.ActorClasses, 2)
}

func TestFindMessageDescriptor(t *testing.T) {
	tsm, err := NewFromFile(path.Join(testDataDir, "cogment.yaml"))
	assert.NoError(t, err)

	fooDataDescriptor, err := tsm.FindMessageDescriptor("foo.Data")
	assert.NoError(t, err)
	assert.Equal(t, fooDataDescriptor.Fields().ByName("member").Kind(), protoreflect.StringKind)

	objectInfoDescriptor, err := tsm.FindMessageDescriptor("flames.ObjectInfo")
	assert.NoError(t, err)
	assert.Equal(t, objectInfoDescriptor.Fields().Len(), 3)

	_, err = tsm.FindMessageDescriptor("flames.CellState")
	assert.Error(t, err)
}

func TestNewTrialConfig(t *testing.T) {
	tsm, err := NewFromFile(path.Join(testDataDir, "cogment.yaml"))
	assert.NoError(t, err)

	trialConfig, err := tsm.NewTrialConfig()
	assert.NoError(t, err)
	assert.True(t, trialConfig.IsValid())
}

func TestNewEnvironmentConfig(t *testing.T) {
	tsm, err := NewFromFile(path.Join(testDataDir, "cogment.yaml"))
	assert.NoError(t, err)

	environmentConfig, err := tsm.NewEnvironmentConfig()
	assert.NoError(t, err)
	assert.True(t, environmentConfig.IsValid())
}

func TestNewActorConfig(t *testing.T) {
	tsm, err := NewFromFile(path.Join(testDataDir, "cogment.yaml"))
	assert.NoError(t, err)

	actorConfig, err := tsm.NewActorConfig("plane")
	assert.NoError(t, err)
	assert.Nil(t, actorConfig)

	actorConfig, err = tsm.NewActorConfig("ai_drone")
	assert.NoError(t, err)
	assert.True(t, actorConfig.IsValid())
}

func TestNewAction(t *testing.T) {
	tsm, err := NewFromFile(path.Join(testDataDir, "cogment.yaml"))
	assert.NoError(t, err)

	action, err := tsm.NewAction("plane")
	assert.NoError(t, err)
	assert.True(t, action.IsValid())

	_, err = tsm.NewAction("bob")
	assert.Error(t, err)
}

func TestNewObservation(t *testing.T) {
	tsm, err := NewFromFile(path.Join(testDataDir, "cogment.yaml"))
	assert.NoError(t, err)

	action, err := tsm.NewObservation("ai_drone")
	assert.NoError(t, err)
	assert.True(t, action.IsValid())

	_, err = tsm.NewObservation("alice")
	assert.Error(t, err)
}
