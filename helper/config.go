package helper

import (
	"fmt"
	"github.com/spf13/viper"
)

var CfgFile string

func CurrentConfig(key string) string {
	remote := viper.GetString("remote")
	output := viper.GetString(fmt.Sprintf("%s.%s", remote, key))
	return output
}
