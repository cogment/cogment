package configure

import (
	"bytes"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"gitlab.com/cogment/cogment/helper"
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
