// Copyright 2021 Artificial Intelligence Redefined <dev+cogment@ai-r.com>
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
	"io/ioutil"
	"log"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cogment/cogment-cli/api"
)

func TestGeneratePythonSettings(t *testing.T) {
	dir, err := ioutil.TempDir("", "testgeneratepythonsettings")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	config, err := api.CreateProjectConfigFromYaml("../testdata/cogment.yaml")
	if err != nil {
		log.Fatal(err)
	}
	err = generate(config, []string{dir}, []string{})
	assert.Nil(t, err)

	assert.FileExists(t, path.Join(dir, "somedata_pb2.py"))
	assert.FileExists(t, path.Join(dir, "subdir/otherdata_pb2.py"))
	assert.FileExists(t, path.Join(dir, "cog_settings.py"))

	// TODO add a simple python script launch to check if the generated files are "importable" (this would require an easy to install cogment-py-sdk)
}
