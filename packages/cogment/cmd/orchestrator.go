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

package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cogment/cogment/services/orchestrator"
)

// orchestratorViper represents the configuration of the orchestrator command
var orchestratorViper = viper.New()

var orchestratorLifecyclePortKey = "lifecycle_port"
var orchestratorActorPortKey = "actor_port"
var orchestratorActorHTTPPortKey = "actor_http_port"
var orchestratorParamsFileKey = "params"
var orchestratorDirectoryServicesKey = "directory_services"
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

		options := orchestrator.Options{
			LifecyclePort:              orchestratorViper.GetUint(orchestratorLifecyclePortKey),
			ActorPort:                  orchestratorViper.GetUint(orchestratorActorPortKey),
			ActorHTTPPort:              orchestratorViper.GetUint(orchestratorActorHTTPPortKey),
			ParamsFile:                 orchestratorViper.GetString(orchestratorParamsFileKey),
			DirectoryServicesEndpoints: orchestratorViper.GetStringSlice(orchestratorDirectoryServicesKey),
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
			log.Info("received interruption signal")
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
	servicesCmd.AddCommand(orchestratorCmd)

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

	orchestratorViper.SetDefault(orchestratorActorHTTPPortKey, orchestrator.DefaultOptions.ActorHTTPPort)
	_ = orchestratorViper.BindEnv(orchestratorActorHTTPPortKey, "COGMENT_WEB_PROXY_PORT")
	orchestratorCmd.Flags().Uint(
		orchestratorActorHTTPPortKey,
		orchestratorViper.GetUint(orchestratorActorHTTPPortKey),
		"The port on which grpc web proxy listens and forwards to the trial actors port (disabled if 0)",
	)

	_ = orchestratorViper.BindEnv(orchestratorParamsFileKey, "COGMENT_DEFAULT_PARAMS_FILE")
	orchestratorCmd.Flags().String(
		orchestratorParamsFileKey,
		orchestratorViper.GetString(orchestratorParamsFileKey),
		"Default trial parameters file name",
	)

	orchestratorViper.SetDefault(orchestratorDirectoryServicesKey, orchestrator.DefaultOptions.DirectoryServicesEndpoints)
	_ = orchestratorViper.BindEnv(orchestratorDirectoryServicesKey, "COGMENT_DIRECTORY_SERVICES")
	orchestratorCmd.Flags().StringSlice(
		orchestratorDirectoryServicesKey,
		orchestratorViper.GetStringSlice(orchestratorDirectoryServicesKey),
		"Directory service gRPC endpoints",
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
		"COGMENT_ORCHestrator_PROMETHEUS_PORT",
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