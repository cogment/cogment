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

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cogment/cogment/cmd/services/utils"
	"github.com/cogment/cogment/services/orchestrator"
	"github.com/cogment/cogment/version"
)

// orchestratorViper represents the configuration of the orchestrator command
var orchestratorViper = viper.New()

var orchestratorLifecyclePortKey = "lifecycle_port"
var orchestratorActorPortKey = "actor_port"
var orchestratorActorWebPortKey = "actor_web_port"
var orchestratorParamsFileKey = "params"
var orchestratorDirectoryAutoRegisterKey = "directory_auto_register"
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
			DirectoryRegistrationOptions: utils.GetDirectoryRegistrationOptions(orchestratorViper),
			LifecyclePort:                orchestratorViper.GetUint(orchestratorLifecyclePortKey),
			ActorPort:                    orchestratorViper.GetUint(orchestratorActorPortKey),
			ActorWebPort:                 actorWebPort,
			ParamsFile:                   orchestratorViper.GetString(orchestratorParamsFileKey),
			DirectoryAutoRegister:        orchestratorViper.GetBool(orchestratorDirectoryAutoRegisterKey),
			PretrialHooksEndpoints:       orchestratorViper.GetStringSlice(orchestratorPretrialHooksKey),
			PrometheusPort:               orchestratorViper.GetUint(orchestratorPrometheusPortKey),
			StatusFile:                   orchestratorViper.GetString(orchestratorStatusFileKey),
			PrivateKeyFile:               orchestratorViper.GetString(orchestratorPrivateKeyFileKey),
			RootCertificateFile:          orchestratorViper.GetString(orchestratorRootCertFileKey),
			TrustChainFile:               orchestratorViper.GetString(orchestratorTrustChainFileKey),
			GarbageCollectorFrequency:    orchestratorViper.GetUint(orchestratorGcFrequencyKey),
		}

		ctx := utils.ContextWithUserTermination(context.Background())

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

	utils.PopulateDirectoryRegistrationOptionsFlags(
		"ORCHESTRATOR", orchestratorCmd, orchestratorViper,
		orchestrator.DefaultOptions.DirectoryRegistrationOptions,
	)

	orchestratorViper.SetDefault(orchestratorDirectoryAutoRegisterKey, orchestrator.DefaultOptions.DirectoryAutoRegister)
	_ = orchestratorViper.BindEnv(orchestratorDirectoryAutoRegisterKey, "COGMENT_ORCHESTRATOR_DIRECTORY_AUTO_REGISTER")
	orchestratorCmd.Flags().Bool(
		orchestratorDirectoryAutoRegisterKey,
		viper.GetBool(orchestratorDirectoryAutoRegisterKey),
		"Whether to register the Orchestrator automatically to the given directory (disabled if false or 0)",
	)

	orchestratorViper.SetDefault(orchestratorPretrialHooksKey, orchestrator.DefaultOptions.PretrialHooksEndpoints)
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
