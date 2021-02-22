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

package configure

import (
	"bytes"
	"github.com/cogment/cogment-cli/helper"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConfigureRemoteCommand(t *testing.T) {
	viper.SetFs(afero.NewMemMapFs())
	viper.AddConfigPath("/tmp")
	viper.SetConfigName(".cogment")
	helper.CfgFile = "/tmp/.cogment.toml"

	const expectedToken = "ABCD12345"

	var stdin bytes.Buffer
	//1st call for URL then token
	stdin.Write([]byte("\n" + expectedToken + "\n"))

	err := runConfigureRemoteCmd("default", &stdin)
	assert.NoError(t, err)

	if err := viper.ReadInConfig(); err != nil {
		t.Fatal("Unable to read config file : ", err)
	}

	assert.Equal(t, expectedToken, viper.GetString("default.token"))
	assert.Equal(t, defaultApiUrl, viper.GetString("default.url"))
}
