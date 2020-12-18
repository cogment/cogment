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
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bradleyjkemp/cupaloy"
	"github.com/stretchr/testify/assert"
	"gitlab.com/cogment/cogment/api"
)

var expectedConfig = api.ExtendDefaultProjectConfig(&api.ProjectConfig{
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
})

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
	assert.Equal(t, *expectedConfig, *config)
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
	assert.Equal(t, *expectedConfig, *config)
}

func TestCreateProjectFiles(t *testing.T) {

	dir, err := ioutil.TempDir("", "testcreateprojectfiles")

	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	config := api.ExtendDefaultProjectConfig(&api.ProjectConfig{
		ProjectName: "testit",
		Components: api.ComponentsConfigurations{
			Orchestrator: api.OrchestratorConfiguration{Version: "v1.0"},
		},
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
	})

	err = createProjectFiles(dir, config)
	assert.NoError(t, err)

	generatedFiles := []string{}
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		assert.NoError(t, err)
		if !info.IsDir() {
			relativePath, err := filepath.Rel(dir, path)
			assert.NoError(t, err)
			generatedFiles = append(generatedFiles, relativePath)

			t.Run(strings.ReplaceAll(relativePath, "/", "-"), func(t *testing.T) {
				fileContent, err := ioutil.ReadFile(path)
				assert.NoError(t, err)
				// Check each file against the previously generated snapshot
				cupaloy.SnapshotT(t, fileContent)
			})
		}
		return nil
	})
	// Check the generate file lists against the previous snapshot
	cupaloy.SnapshotT(t, generatedFiles)
}
