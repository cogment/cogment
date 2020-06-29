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
		&api.ActorClass{Id: "player red"},
		&api.ActorClass{Id: "player white"},
	},
	TrialParams: &api.TrialParams{
		Actors: []*api.Actor{
			&api.Actor{ActorClass: "player red", Endpoint: "grpc://player-red:9000"},
			&api.Actor{ActorClass: "player red", Endpoint: "human"},
			&api.Actor{ActorClass: "player white", Endpoint: "grpc://player-white:9000"},
		},
	},
}

func TestCreateProjectConfig(t *testing.T) {

	input := []string{
		"2",            // nb of actor types
		"player red",   // name 1st
		"1",            // nb ai
		"1",            // nb human
		"player white", // name 2nd
		"1",            // nb ai
		"0",            // nb human
	}

	var stdin bytes.Buffer
	stdin.Write([]byte(strings.Join(input, "\n") + "\n"))

	config, err := createProjectConfigFromReader(&stdin)

	assert.Nil(t, err)
	assert.Equal(t, expectedConfig, *config)
}

func TestCreateProjectConfigWindows(t *testing.T) {

	input := []string{
		"2",            // nb of actor types
		"player red",   // name 1st
		"1",            // nb ai
		"1",            // nb human
		"player white", // name 2nd
		"1",            // nb ai
		"0",            // nb human
	}

	var stdin bytes.Buffer
	stdin.Write([]byte(strings.Join(input, "\r\n") + "\r\n"))

	config, err := createProjectConfigFromReader(&stdin)

	assert.Nil(t, err)
	assert.Equal(t, expectedConfig, *config)
}

func TestCreateProjectConfigWhitespace(t *testing.T) {

	input := []string{
		"2",            // nb of actor types
		" player red",   // name 1st
		"1",            // nb ai
		"1",            // nb human
		"player white ", // name 2nd
		"1",            // nb ai
		"0",            // nb human
	}

	var stdin bytes.Buffer
	stdin.Write([]byte(strings.Join(input, "\n") + "\n"))

	config, err := createProjectConfigFromReader(&stdin)

	assert.Nil(t, err)
	assert.Equal(t, expectedConfig, *config)
}

func TestCreateProjectFiles(t *testing.T) {

	dir, err := ioutil.TempDir("", "TestCreateProjectFiles")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	err = createProjectFiles(dir, &expectedConfig)

	assert.NoError(t, err)
	assert.FileExists(t, path.Join(dir, "agents", "player_red", "main.py"))
	assert.FileExists(t, path.Join(dir, "agents", "player_red", "Dockerfile"))
	assert.FileExists(t, path.Join(dir, "agents", "player_white", "main.py"))
	assert.FileExists(t, path.Join(dir, "agents", "player_white", "Dockerfile"))

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
