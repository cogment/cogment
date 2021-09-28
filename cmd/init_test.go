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

package cmd

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bradleyjkemp/cupaloy"
	"github.com/stretchr/testify/assert"

	"github.com/cogment/cogment-cli/api"
	"github.com/cogment/cogment-cli/helper"
)

var expectedConfig, err = api.ExtendDefaultProjectConfig(&api.ProjectConfig{
	ActorClasses: []*api.ActorClass{
		&api.ActorClass{Name: "master"},
		&api.ActorClass{Name: "smart"},
		&api.ActorClass{Name: "dumb"},
	},
	TrialParams: &api.TrialParams{
		Actors: []*api.TrialActor{
			&api.TrialActor{Name: "client_actor", ActorClass: "master", Implementation: "client_actor_impl", Endpoint: "client"},
			&api.TrialActor{Name: "smart_smart_impl_1", ActorClass: "smart", Implementation: "smart_impl", Endpoint: "grpc://smart-impl:9000"},
			&api.TrialActor{Name: "dumb_dumb_impl_1", ActorClass: "dumb", Implementation: "dumb_impl", Endpoint: "grpc://dumb-impl:9000"},
		},
	},
	WebClient:  true,
	Typescript: true,
})

func TestCreateProjectConfig(t *testing.T) {

	input := []string{
		"3",          // 3 actor classes
		"master",     // class #1
		"0",          // 0 service actor implementation of "master"
		"Y",          // A client actor implementation of "master"
		"smart",      // class #2
		"1",          // 1 service actor implementation of "smart"
		"smart_impl", // Named "smart_impl"
		"1",          // 1 actor using this implementation
		"dumb",       // class #3
		"1",          // 1 service actor implementation of "dumb"
		"dumb_impl",  // Named "dumb_impl"
		"1",          // 1 actor using this implementation
		"Y",          // Create a web client
		"Y",          // Web client is in typescript
	}

	var stdin bytes.Buffer
	stdin.Write([]byte(strings.Join(input, "\n") + "\n"))

	config, err := createProjectConfigFromReader(&stdin)

	assert.Nil(t, err)
	assert.Equal(t, *expectedConfig, *config)
}

func TestCreateProjectConfigWindows(t *testing.T) {

	input := []string{
		"3",          // 3 actor classes
		"master",     // class #1
		"0",          // 0 service actor implementation of "master"
		"Y",          // A client actor implementation of "master"
		"smart",      // class #2
		"1",          // 1 service actor implementation of "smart"
		"smart_impl", // Named "smart_impl"
		"1",          // 1 actor using this implementation
		"dumb",       // class #3
		"1",          // 1 service actor implementation of "dumb"
		"dumb_impl",  // Named "dumb_impl"
		"1",          // 1 actor using this implementation
		"Y",          // Create a web client
		"Y",          // Web client is in typescript
	}

	var stdin bytes.Buffer
	stdin.Write([]byte(strings.Join(input, "\r\n") + "\r\n"))

	config, err := createProjectConfigFromReader(&stdin)

	assert.Nil(t, err)
	assert.Equal(t, *expectedConfig, *config)
}

func TestCreateProjectFiles(t *testing.T) {

	dir, err := ioutil.TempDir("", "TestCreateProjectFiles")

	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			log.Fatalf("Failed to clean up temporary init files: %v", err)
		}
	}()

	config, err := api.ExtendDefaultProjectConfig(&api.ProjectConfig{
		ProjectName:       "testit",
		ProjectConfigPath: path.Join(dir, "cogment.yaml"),
		Components: api.ComponentsConfigurations{
			Orchestrator: helper.VersionInfo{Version: "v1.0"},
			Python:       helper.VersionInfo{Version: "1.0"},
			Metrics:      helper.VersionInfo{Version: "2.0"},
			Dashboard:    helper.VersionInfo{Version: "v1.5-beta"},
			Javascript:   helper.VersionInfo{Version: "1.25.5"},
		},
		ActorClasses: []*api.ActorClass{
			&api.ActorClass{Name: "master"},
			&api.ActorClass{Name: "smart"},
			&api.ActorClass{Name: "dumb"},
		},
		TrialParams: &api.TrialParams{
			Actors: []*api.TrialActor{
				&api.TrialActor{Name: "human", ActorClass: "master", Implementation: "human", Endpoint: "client"},
				&api.TrialActor{Name: "ai_1", ActorClass: "smart", Implementation: "smart_impl", Endpoint: "grpc://smart:9000"},
				&api.TrialActor{Name: "ai_2", ActorClass: "dumb", Implementation: "dumb_impl", Endpoint: "grpc://dumb:9000"},
			},
		},
	})
	assert.NoError(t, err)

	err = createProjectFiles(config)
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
				// Skipping generated content check on files generated by protoc
				if !strings.HasSuffix(path, "_pb2.py") {
					cupaloy.SnapshotT(t, fileContent)
				}
			})
		}
		return nil
	})
	// Check the generate file lists against the previous snapshot
	cupaloy.SnapshotT(t, generatedFiles)
}

func TestCreateProjectFilesDashes(t *testing.T) {

	dir, err := ioutil.TempDir("", "TestCreateProjectFilesDashes")

	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			log.Fatalf("Failed to clean up temporary init files: %v", err)
		}
	}()

	config, err := api.ExtendDefaultProjectConfig(&api.ProjectConfig{
		ProjectName:       "a-test-project-with-dashes",
		ProjectConfigPath: path.Join(dir, "cogment.yaml"),
		Components: api.ComponentsConfigurations{
			Orchestrator: helper.VersionInfo{Version: "v1.0"},
		},
		ActorClasses: []*api.ActorClass{
			&api.ActorClass{Name: "smart"},
			&api.ActorClass{Name: "dumb"},
		},
		TrialParams: &api.TrialParams{
			Actors: []*api.TrialActor{
				&api.TrialActor{ActorClass: "smart", Endpoint: "grpc://smart:9000"},
				&api.TrialActor{ActorClass: "dumb", Endpoint: "grpc://dumb:9000"},
			},
		},
	})
	assert.NoError(t, err)

	err = createProjectFiles(config)
	assert.NoError(t, err)
}
