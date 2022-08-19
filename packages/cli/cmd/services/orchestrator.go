// Copyright 2022 AI Redefined Inc. <dev+cogment@ai-r.com>
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
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cogment/cogment/services/orchestrator"
	"github.com/cogment/cogment/version"
)

// orchestratorViper represents the configuration of the orchestrator command
var orchestratorViper = viper.New()

var orchestratorLifecyclePortKey = "lifecycle_port"
var orchestratorActorPortKey = "actor_port"
var orchestratorActorWebPortKey = "actor_web_port"
var orchestratorParamsFileKey = "params"
var orchestratorDirectoryServicesKey = "directory_endpoint"
var orchestratorDirectoryAuthTokenKey = "directory_authentication_token"
var orchestratorDirectoryAutoRegisterKey = "directory_auto_register"
var orchestratorDirectoryRegisterHostKey = "directory_registration_host"
var orchestratorDirectoryRegisterPropsKey = "directory_registration_properties"
var orchestratorPretrialHooksKey = "pre_trial_hooks"
var orchestratorPrometheusPortKey = "prometheus_port"
var orchestratorStatusFileKey = "status_file"
var orchestratorPrivateKeyFileKey = "private_key"
var orchestratorRootCertFileKey = "root_cert"
var orchestratorTrustChainFileKey = "trust_chain"
var orchestratorGcFrequencyKey = "gc_frequency"

// orchestratorCmd represents the orchestrator command
var orchestratorCmd = &cobra.Command{
	Use:   "orchestrator",
	Short: "Run the orchestrator",
	Args:  cobra.NoArgs,
	RunE: func(_cmd *cobra.Command, _args []string) error {
		err := configureLog(servicesViper)
		if err != nil {
			return err
		}

		log.WithFields(logrus.Fields{
			"version": version.Version,
			"hash":    version.Hash,
		}).Info("starting the orchestrator service")

		actorWebPort := orchestratorViper.GetUint(orchestratorActorWebPortKey)
		if actorWebPort == orchestrator.DefaultOptions.ActorWebPort {
			// Fallback to the deprecated flag if needed
			actorWebPort = orchestratorViper.GetUint("actor_http_port")
		}

		options := orchestrator.Options{
			LifecyclePort:              orchestratorViper.GetUint(orchestratorLifecyclePortKey),
			ActorPort:                  orchestratorViper.GetUint(orchestratorActorPortKey),
			ActorWebPort:               actorWebPort,
			ParamsFile:                 orchestratorViper.GetString(orchestratorParamsFileKey),
			DirectoryServicesEndpoints: orchestratorViper.GetStringSlice(orchestratorDirectoryServicesKey),
			DirectoryAuthToken:         orchestratorViper.GetString(orchestratorDirectoryAuthTokenKey),
			DirectoryAutoRegister:      orchestratorViper.GetUint(orchestratorDirectoryAutoRegisterKey),
			DirectoryRegisterHost:      orchestratorViper.GetString(orchestratorDirectoryRegisterHostKey),
			DirectoryRegisterProps:     orchestratorViper.GetString(orchestratorDirectoryRegisterPropsKey),
			PretrialHooksEndpoits:      orchestratorViper.GetStringSlice(orchestratorPretrialHooksKey),
			PrometheusPort:             orchestratorViper.GetUint(orchestratorPrometheusPortKey),
			StatusFile:                 orchestratorViper.GetString(orchestratorStatusFileKey),
			PrivateKeyFile:             orchestratorViper.GetString(orchestratorPrivateKeyFileKey),
			RootCertificateFile:        orchestratorViper.GetString(orchestratorRootCertFileKey),
			TrustChainFile:             orchestratorViper.GetString(orchestratorTrustChainFileKey),
			GarbageCollectorFrequency:  orchestratorViper.GetUint(orchestratorGcFrequencyKey),
		}

		ctx, cancel := context.WithCancel(context.Background())
		// using a buffered channel cf. https://link.medium.com/M8dPZv9Wuob
		interruptChan := make(chan os.Signal, 1)
		signal.Notify(interruptChan, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-interruptChan
			log.Debug("received interruption signal")
			cancel()
		}()

		err = orchestrator.Run(ctx, options)
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
	orchestratorViper.SetDefault(orchestratorLifecyclePortKey, orchestrator.DefaultOptions.LifecyclePort)
	_ = orchestratorViper.BindEnv(orchestratorLifecyclePortKey, "COGMENT_LIFECYCLE_PORT", "TRIAL_LIFECYCLE_PORT")
	orchestratorCmd.Flags().Uint(
		orchestratorLifecyclePortKey,
		orchestratorViper.GetUint(orchestratorLifecyclePortKey),
		"The port to listen for trial lifecycle on",
	)

	orchestratorViper.SetDefault(orchestratorActorPortKey, orchestrator.DefaultOptions.ActorPort)
	_ = orchestratorViper.BindEnv(orchestratorActorPortKey, "COGMENT_ACTOR_PORT", "TRIAL_ACTOR_PORT")
	orchestratorCmd.Flags().Uint(
		orchestratorActorPortKey,
		orchestratorViper.GetUint(orchestratorActorPortKey),
		"The port to listen for trial actors on",
	)

	orchestratorViper.SetDefault(orchestratorActorWebPortKey, orchestrator.DefaultOptions.ActorWebPort)
	_ = orchestratorViper.BindEnv(orchestratorActorWebPortKey, "COGMENT_WEB_PROXY_PORT")
	orchestratorCmd.Flags().Uint(
		orchestratorActorWebPortKey,
		orchestratorViper.GetUint(orchestratorActorWebPortKey),
		"The port on which grpc web proxy listens and forwards to the trial actors port (disabled if 0)",
	)
	orchestratorCmd.Flags().Uint(
		"actor_http_port",
		orchestratorViper.GetUint(orchestratorActorWebPortKey),
		"",
	)
	_ = orchestratorCmd.Flags().MarkDeprecated(
		"actor_http_port",
		fmt.Sprintf("please use --%s instead", orchestratorActorWebPortKey),
	)

	_ = orchestratorViper.BindEnv(orchestratorParamsFileKey, "COGMENT_DEFAULT_PARAMS_FILE")
	orchestratorCmd.Flags().String(
		orchestratorParamsFileKey,
		orchestratorViper.GetString(orchestratorParamsFileKey),
		"Default trial parameters file name",
	)

	orchestratorViper.SetDefault(orchestratorDirectoryServicesKey, orchestrator.DefaultOptions.DirectoryServicesEndpoints)
	_ = orchestratorViper.BindEnv(orchestratorDirectoryServicesKey, "COGMENT_DIRECTORY_ENDPOINT")
	orchestratorCmd.Flags().StringSlice(
		orchestratorDirectoryServicesKey,
		orchestratorViper.GetStringSlice(orchestratorDirectoryServicesKey),
		"Directory service gRPC endpoints",
	)

	orchestratorViper.SetDefault(orchestratorDirectoryAuthTokenKey, orchestrator.DefaultOptions.DirectoryAuthToken)
	_ = orchestratorViper.BindEnv(orchestratorDirectoryAuthTokenKey, "COGMENT_DIRECTORY_AUTHENTICATION_TOKEN")
	orchestratorCmd.Flags().String(
		orchestratorDirectoryAuthTokenKey,
		orchestratorViper.GetString(orchestratorDirectoryAuthTokenKey),
		"Authentication token for directory services",
	)

	orchestratorViper.SetDefault(orchestratorDirectoryAutoRegisterKey, orchestrator.DefaultOptions.DirectoryAutoRegister)
	_ = orchestratorViper.BindEnv(orchestratorDirectoryAutoRegisterKey, "COGMENT_ORCHESTRATOR_DIRECTORY_AUTO_REGISTER")
	orchestratorCmd.Flags().Uint(
		orchestratorDirectoryAutoRegisterKey,
		orchestratorViper.GetUint(orchestratorDirectoryAutoRegisterKey),
		"Whether to register the Orchestrator automatically to the given directory (disabled if 0)",
	)

	orchestratorViper.SetDefault(orchestratorDirectoryRegisterHostKey, orchestrator.DefaultOptions.DirectoryRegisterHost)
	_ = orchestratorViper.BindEnv(orchestratorDirectoryRegisterHostKey,
		"COGMENT_ORCHESTRATOR_DIRECTORY_REGISTRATION_HOST")
	orchestratorCmd.Flags().String(
		orchestratorDirectoryRegisterHostKey,
		orchestratorViper.GetString(orchestratorDirectoryRegisterHostKey),
		"Host to register as the Orchestrator in the Directory (empty for self discovery of host)",
	)

	orchestratorViper.SetDefault(orchestratorDirectoryRegisterPropsKey,
		orchestrator.DefaultOptions.DirectoryRegisterProps)
	_ = orchestratorViper.BindEnv(orchestratorDirectoryRegisterPropsKey,
		"COGMENT_ORCHESTRATOR_DIRECTORY_REGISTRATION_PROPERTIES")
	orchestratorCmd.Flags().String(
		orchestratorDirectoryRegisterPropsKey,
		orchestratorViper.GetString(orchestratorDirectoryRegisterPropsKey),
		"Properties to register to the Directory for the Orchestrator",
	)

	orchestratorViper.SetDefault(orchestratorPretrialHooksKey, orchestrator.DefaultOptions.PretrialHooksEndpoits)
	_ = orchestratorViper.BindEnv(orchestratorPretrialHooksKey, "COGMENT_PRE_TRIAL_HOOKS")
	orchestratorCmd.Flags().StringSlice(
		orchestratorPretrialHooksKey,
		orchestratorViper.GetStringSlice(orchestratorPretrialHooksKey),
		"Pre-trial hook gRPC endpoints",
	)

	orchestratorViper.SetDefault(orchestratorPrometheusPortKey, orchestrator.DefaultOptions.PrometheusPort)
	_ = orchestratorViper.BindEnv(
		orchestratorPrometheusPortKey,
		"COGMENT_ORCHESTRATOR_PROMETHEUS_PORT",
		"PROMETHEUS_PORT",
	)
	orchestratorCmd.Flags().Uint(
		orchestratorPrometheusPortKey,
		orchestratorViper.GetUint(orchestratorPrometheusPortKey),
		"The port to broadcast prometheus metrics on (disabled if 0)",
	)

	orchestratorViper.SetDefault(orchestratorStatusFileKey, orchestrator.DefaultOptions.StatusFile)
	_ = orchestratorViper.BindEnv(orchestratorStatusFileKey, "COGMENT_STATUS_FILE")
	orchestratorCmd.Flags().String(
		orchestratorStatusFileKey,
		orchestratorViper.GetString(orchestratorStatusFileKey),
		"File to store the Orchestrator status to",
	)

	orchestratorViper.SetDefault(orchestratorPrivateKeyFileKey, orchestrator.DefaultOptions.PrivateKeyFile)
	_ = orchestratorViper.BindEnv(orchestratorPrivateKeyFileKey, "COGMENT_PRIVATE_KEY_FILE")
	orchestratorCmd.Flags().String(
		orchestratorPrivateKeyFileKey,
		orchestratorViper.GetString(orchestratorPrivateKeyFileKey),
		"File containing PEM encoded private key",
	)

	orchestratorViper.SetDefault(orchestratorRootCertFileKey, orchestrator.DefaultOptions.RootCertificateFile)
	_ = orchestratorViper.BindEnv(orchestratorRootCertFileKey, "COGMENT_ROOT_CERT_FILE")
	orchestratorCmd.Flags().String(
		orchestratorRootCertFileKey,
		orchestratorViper.GetString(orchestratorRootCertFileKey),
		"File containing a PEM encoded trusted root certificate",
	)

	orchestratorViper.SetDefault(orchestratorTrustChainFileKey, orchestrator.DefaultOptions.TrustChainFile)
	_ = orchestratorViper.BindEnv(orchestratorTrustChainFileKey, "COGMENT_TRUST_CHAIN_FILE")
	orchestratorCmd.Flags().String(
		orchestratorTrustChainFileKey,
		orchestratorViper.GetString(orchestratorTrustChainFileKey),
		"File containing a PEM encoded trust chain",
	)

	orchestratorViper.SetDefault(orchestratorGcFrequencyKey, orchestrator.DefaultOptions.GarbageCollectorFrequency)
	_ = orchestratorViper.BindEnv(orchestratorGcFrequencyKey, "COGMENT_GC_FREQUENCY")
	orchestratorCmd.Flags().Uint(
		orchestratorGcFrequencyKey,
		orchestratorViper.GetUint(orchestratorGcFrequencyKey),
		"Number of trials between garbage collection runs",
	)

	// Don't sort alphabetically, keep insertion order
	orchestratorCmd.Flags().SortFlags = false

	// Bind "cobra" flags defined in the CLI with viper
	_ = orchestratorViper.BindPFlags(orchestratorCmd.Flags())
}
