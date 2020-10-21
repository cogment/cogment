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
	"fmt"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.com/cogment/cogment/cmd/configure"
	"gitlab.com/cogment/cogment/cmd/registry"
	"gitlab.com/cogment/cogment/helper"
	"log"
	"os"
	"path"
	"strconv"
)

//var cfgFile string
var Verbose, BetaFeatureEnabled bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cogment",
	Short: "A brief description of your application",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig, enableBetaFeatures)

	rootCmd.PersistentFlags().StringVar(&helper.CfgFile, "config", "", "config file (default is $HOME/.cogment.toml)")

	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "verbose output")

	rootCmd.PersistentFlags().String("app", "", "Application ID")
	viper.BindPFlag("app", rootCmd.PersistentFlags().Lookup("app"))

	rootCmd.PersistentFlags().BoolVar(&BetaFeatureEnabled, "enable_beta", false, "Enable beta features")
	viper.BindPFlag("enable_beta", rootCmd.PersistentFlags().Lookup("enable_beta"))

	rootCmd.AddCommand(registry.NewRegistryCommand(Verbose))
	rootCmd.AddCommand(configure.NewConfigureCmd())

}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	configName := ".cogment"

	if helper.CfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(helper.CfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".cogment" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(configName)

		helper.CfgFile = path.Join(home, configName+".toml")
	}

	viper.SetEnvPrefix("cog")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); Verbose && err != nil {
		log.Println(err)
	}

	if Verbose {
		log.Println("Using config file:", viper.ConfigFileUsed())
	}

}

func enableBetaFeatures() {
	value := viper.GetString("enable_beta")
	result, err := strconv.ParseBool(value)
	if err != nil {
		log.Printf("%s is not a valid COG_ENABLE_BETA value, can t be converted to bool\n", value)
	}

	if result {
		for _, cmd := range rootCmd.Commands() {
			cmd.Hidden = false
		}
	}

}
