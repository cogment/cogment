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

package services

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cogment/cogment/cmd/services/utils"
	"github.com/cogment/cogment/services/proxy"
	"github.com/cogment/cogment/utils/endpoint"
	"github.com/cogment/cogment/version"
)

// proxyViper represents the configuration of the web_proxy command
var proxyViper = viper.New()

const proxyWebPortKey = "web_port"
const proxyWebPortEnv = "COGMENT_WEB_PROXY_WEB_PORT"
const proxyOrchestratorEndpointKey = "orchestrator_endpoint"
const proxyOrchestratorEndpointEnv = "COGMENT_ORCHESTRATOR_ENDPOINT"
const proxyGrpcPortKey = "port"
const proxyGrpcPortEnv = "COGMENT_WEB_PROXY_PORT"
const proxySpecFileKey = "spec_file"
const proxySpecFileEnv = "COGMENT_SPEC_FILE"
const proxyImplementationKey = "implementation"
const proxyImplementationEnv = "COGMENT_WEB_PROXY_IMPLEMENTATION"
const proxySecretKey = "secret"
const proxySecretEnv = "COGMENT_WEB_PROXY_SECRET"

// proxyCmd represents the web_proxy
var proxyCmd = &cobra.Command{
	Use:     "web_proxy",
	Aliases: []string{"proxy"},
	Short:   "Run the web proxy",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _args []string) error {
		err := configureLog(servicesViper)
		if err != nil {
			return err
		}

		log.WithFields(logrus.Fields{
			"version": version.Version,
			"hash":    version.Hash,
		}).Info("starting the web proxy service")

		log.Warn("The web proxy is currently in preview, API and features might evolve in the future.")

		directoryOptions, err := utils.GetDirectoryRegistrationOptions(proxyViper)
		if err != nil {
			return err
		}

		orchestratorEndpoint, err := endpoint.Parse(proxyViper.GetString(proxyOrchestratorEndpointKey))
		if err != nil {
			return err
		}

		options := proxy.Options{
			RegistrationOptions:  directoryOptions,
			WebPort:              proxyViper.GetUint(proxyWebPortKey),
			GrpcPort:             proxyViper.GetUint(proxyGrpcPortKey),
			OrchestratorEndpoint: orchestratorEndpoint,
			SpecFile:             proxyViper.GetString(proxySpecFileKey),
			Implementation:       proxyViper.GetString(proxyImplementationKey),
			Secret:               proxyViper.GetString(proxySecretKey),
		}

		ctx := utils.ContextWithUserTermination(context.Background())

		err = proxy.Run(ctx, options)
		if err != nil {
			if err == context.Canceled {
				log.Info("interrupted by user")
				return nil
			}
			return err
		}
		return nil
	},
}

func init() {
	proxyViper.SetDefault(proxyWebPortKey, proxy.DefaultOptions.WebPort)
	_ = proxyViper.BindEnv(proxyWebPortKey, proxyWebPortEnv)
	proxyCmd.Flags().Uint(
		proxyWebPortKey,
		proxyViper.GetUint(proxyWebPortKey),
		"The http port to listen on",
	)

	proxyViper.SetDefault(proxyGrpcPortKey, proxy.DefaultOptions.GrpcPort)
	_ = proxyViper.BindEnv(proxyGrpcPortKey, proxyGrpcPortEnv)
	proxyCmd.Flags().Uint(
		proxyGrpcPortKey,
		proxyViper.GetUint(proxyGrpcPortKey),
		"The gRPC port to listen on",
	)

	proxyViper.SetDefault(proxyOrchestratorEndpointKey, proxy.DefaultOptions.OrchestratorEndpoint)
	_ = proxyViper.BindEnv(proxyOrchestratorEndpointKey, proxyOrchestratorEndpointEnv)
	proxyCmd.Flags().String(
		proxyOrchestratorEndpointKey,
		viper.GetString(proxyOrchestratorEndpointKey),
		"Orchestrator service gRPC endpoint",
	)

	proxyViper.SetDefault(proxySpecFileKey, proxy.DefaultOptions.SpecFile)
	_ = proxyViper.BindEnv(proxySpecFileKey, proxySpecFileEnv)
	proxyCmd.Flags().String(
		proxySpecFileKey,
		viper.GetString(proxySpecFileKey),
		"File name of the trial spec followed by the managed actors",
	)

	proxyViper.SetDefault(proxyImplementationKey, proxy.DefaultOptions.Implementation)
	_ = proxyViper.BindEnv(proxyImplementationKey, proxyImplementationEnv)
	proxyCmd.Flags().String(
		proxyImplementationKey,
		viper.GetString(proxyImplementationKey),
		"Implementation name followed by the managed actors",
	)

	proxyViper.SetDefault(proxySecretKey, proxy.DefaultOptions.Implementation)
	_ = proxyViper.BindEnv(proxySecretKey, proxySecretEnv)
	proxyCmd.Flags().String(
		proxySecretKey,
		viper.GetString(proxySecretKey),
		"Secret used to sign the actor trial tokens",
	)

	utils.PopulateDirectoryRegistrationOptionsFlags(
		"WEB_PROXY",
		proxyCmd,
		proxyViper,
		proxy.DefaultOptions.RegistrationOptions,
	)

	// Don't sort alphabetically, keep insertion order
	proxyCmd.Flags().SortFlags = false

	// Bind "cobra" flags defined in the CLI with viper
	_ = proxyViper.BindPFlags(proxyCmd.Flags())
}
