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

package cmd

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"gitlab.com/cogment/cogment/api"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"testing"
)

var expectedConfig = api.ProjectConfig{
	ActorClasses: []*api.ActorClass{
		&api.ActorClass{Id: "master"},
		&api.ActorClass{Id: "smart"},
		&api.ActorClass{Id: "dumb"},
	},
	TrialParams: &api.TrialParams{
		Actors: []*api.Actor{
			&api.Actor{ActorClass: "master", Endpoint: "human"},
			&api.Actor{ActorClass: "smart", Endpoint: "grpc://smart:9000"},
			&api.Actor{ActorClass: "dumb", Endpoint: "grpc://dumb:9000"},
		},
	},
}

func TestCreateProjectConfig(t *testing.T) {

	input := []string{
		"master", // master client actor name
		"2",      // Number of agent actor types
		"smart",  // Agent actor type 1 name
		"1",      // Number of agent 'smart' instances
		"dumb",   // Agent actor type 2 name
		"1",      // Number of agent 'dumb' instances
	}

	var stdin bytes.Buffer
	stdin.Write([]byte(strings.Join(input, "\n") + "\n"))

	config, err := createProjectConfigFromReader(&stdin)

	assert.Nil(t, err)
	assert.Equal(t, expectedConfig, *config)
}

func TestCreateProjectConfigWindows(t *testing.T) {

	input := []string{
		"master", // master client actor name
		"2",      // Number of agent actor types
		"smart",  // Agent actor type 1 name
		"1",      // Number of agent 'smart' instances
		"dumb",   // Agent actor type 2 name
		"1",      // Number of agent 'dumb' instances
	}

	var stdin bytes.Buffer
	stdin.Write([]byte(strings.Join(input, "\r\n") + "\r\n"))

	config, err := createProjectConfigFromReader(&stdin)

	assert.Nil(t, err)
	assert.Equal(t, expectedConfig, *config)
}

// comment out for now
// func TestCreateProjectConfigWhitespace(t *testing.T) {

// 	input := []string{
// 		"2",            // nb of actor types
// 		" player red",   // name 1st
// 		"1",            // nb ai
// 		"1",            // nb human
// 		"player white ", // name 2nd
// 		"1",            // nb ai
// 		"0",            // nb human
// 	}

// 	var stdin bytes.Buffer
// 	stdin.Write([]byte(strings.Join(input, "\n") + "\n"))

// 	config, err := createProjectConfigFromReader(&stdin)

// 	assert.Nil(t, err)
// 	assert.Equal(t, expectedConfig, *config)
// }

func TestCreateProjectFiles(t *testing.T) {

	dir, err := ioutil.TempDir("", "testcreateprojectfiles")

	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	expectedConfig.ProjectName = "testit"

	err = createProjectFiles(dir, &expectedConfig)

	assert.NoError(t, err)
	assert.FileExists(t, path.Join(dir, "agents", "smart", "main.py"))
	assert.FileExists(t, path.Join(dir, "agents", "smart", "Dockerfile"))
	assert.FileExists(t, path.Join(dir, "agents", "dumb", "main.py"))
	assert.FileExists(t, path.Join(dir, "agents", "dumb", "Dockerfile"))

	assert.FileExists(t, path.Join(dir, "clients", "main.py"))
	assert.FileExists(t, path.Join(dir, "clients", "Dockerfile"))

	assert.FileExists(t, path.Join(dir, "envs", "main.py"))
	assert.FileExists(t, path.Join(dir, "envs", "Dockerfile"))

	assert.FileExists(t, path.Join(dir, "orchestrator", "Dockerfile"))

	assert.FileExists(t, path.Join(dir, ".gitignore"))
	assert.FileExists(t, path.Join(dir, "cogment.yaml"))
	assert.FileExists(t, path.Join(dir, "data.proto"))
	assert.FileExists(t, path.Join(dir, "docker-compose.yaml"))
	assert.FileExists(t, path.Join(dir, "README.md"))

	assert.FileExists(t, path.Join(dir, "data_pb2.py"))
	assert.FileExists(t, path.Join(dir, "cog_settings.py"))

}
