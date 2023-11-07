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
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cogment/cogment/services/directory"
	"github.com/cogment/cogment/version"
)

const (
	directoryPortKey                = "port"
	directoryPortEnv                = "COGMENT_DIRECTORY_PORT"
	directoryGrpcReflectionKey      = "grpc_reflection"
	directoryGrpcReflectionEnv      = "COGMENT_DIRECTORY_GRPC_REFLECTION"
	directoryRegistrationLagKey     = "registration_lag"
	directoryRegistrationLagEnv     = "COGMENT_DIRECTORY_REGISTRATION_LAG"
	directoryPersistenceFilenameKey = "persistence_file"
	directoryPersistenceFilenameEnv = "COGMENT_DIRECTORY_PERSISTENCE_FILE"
	directoryLoadBalancingKey       = "load_balancing"
	directoryLoadBalancingEnv       = "COGMENT_DIRECTORY_LOAD_BALANCING"
	directoryCheckOnInquireKey      = "check_on_inquire"
	directoryCheckOnInquireEnv      = "COGMENT_DIRECTORY_CHECK_ON_INQUIRE"
	directoryForcePermanentKey      = "force_permanent"
	directoryForcePermanentEnv      = "COGMENT_DIRECTORY_FORCE_PERMANENT"
)

var directoryViper = viper.New()

var directoryCmd = &cobra.Command{
	Use:   "directory",
	Short: "Run directory",
	Args:  cobra.NoArgs,
	RunE: func(_cmd *cobra.Command, _args []string) error {
		err := configureLog(servicesViper)
		if err != nil {
			return err
		}

		log.WithFields(logrus.Fields{
			"version": version.Version,
			"hash":    version.Hash,
		}).Info("Starting the directory service")

		options := directory.Options{
			Port:                directoryViper.GetUint(directoryPortKey),
			GrpcReflection:      directoryViper.GetBool(directoryGrpcReflectionKey),
			RegistrationLag:     directoryViper.GetUint(directoryRegistrationLagKey),
			PersistenceFilename: directoryViper.GetString(directoryPersistenceFilenameKey),
			LoadBalancing:       directoryViper.GetBool(directoryLoadBalancingKey),
			CheckOnInquire:      directoryViper.GetBool(directoryCheckOnInquireKey),
			ForcePermanent:      directoryViper.GetBool(directoryForcePermanentKey),
		}

		return directory.Run(options)
	},
}

func init() {
	directoryViper.AllowEmptyEnv(true)

	directoryViper.SetDefault(directoryPortKey, directory.DefaultOptions.Port)
	_ = directoryViper.BindEnv(directoryPortKey, directoryPortEnv)
	directoryCmd.Flags().Uint(
		directoryPortKey,
		directoryViper.GetUint(directoryPortKey),
		"The port to listen on",
	)

	directoryViper.SetDefault(directoryGrpcReflectionKey, directory.DefaultOptions.GrpcReflection)
	_ = directoryViper.BindEnv(directoryGrpcReflectionKey, directoryGrpcReflectionEnv)
	directoryCmd.Flags().Bool(
		directoryGrpcReflectionKey,
		directoryViper.GetBool(directoryGrpcReflectionKey),
		"Start the gRPC reflection server",
	)

	directoryViper.SetDefault(directoryRegistrationLagKey, directory.DefaultOptions.RegistrationLag)
	_ = directoryViper.BindEnv(directoryRegistrationLagKey, directoryRegistrationLagEnv)
	directoryCmd.Flags().Uint(
		directoryRegistrationLagKey,
		directoryViper.GetUint(directoryRegistrationLagKey),
		"Acceptable lag for services registration before they are declared not found in the directory (seconds)",
	)

	directoryViper.SetDefault(directoryPersistenceFilenameKey, directory.DefaultOptions.PersistenceFilename)
	_ = directoryViper.BindEnv(directoryPersistenceFilenameKey, directoryPersistenceFilenameEnv)
	directoryCmd.Flags().String(
		directoryPersistenceFilenameKey,
		directoryViper.GetString(directoryPersistenceFilenameKey),
		"Filename where to store persistence data",
	)

	directoryViper.SetDefault(directoryLoadBalancingKey, directory.DefaultOptions.LoadBalancing)
	_ = directoryViper.BindEnv(directoryLoadBalancingKey, directoryLoadBalancingEnv)
	directoryCmd.Flags().Bool(
		directoryLoadBalancingKey,
		directoryViper.GetBool(directoryLoadBalancingKey),
		"Enable load balancing",
	)

	directoryViper.SetDefault(directoryCheckOnInquireKey, directory.DefaultOptions.CheckOnInquire)
	_ = directoryViper.BindEnv(directoryCheckOnInquireKey, directoryCheckOnInquireEnv)
	directoryCmd.Flags().Bool(
		directoryCheckOnInquireKey,
		directoryViper.GetBool(directoryCheckOnInquireKey),
		"Check health and status when an inquiry is made",
	)

	directoryViper.SetDefault(directoryForcePermanentKey, directory.DefaultOptions.ForcePermanent)
	_ = directoryViper.BindEnv(directoryForcePermanentKey, directoryForcePermanentEnv)
	directoryCmd.Flags().Bool(
		directoryForcePermanentKey,
		directoryViper.GetBool(directoryForcePermanentKey),
		"Force all entries to be 'permanent' (no health checks, no duplicates)",
	)

	// Don't sort alphabetically, keep insertion order
	directoryCmd.Flags().SortFlags = false

	// Bind "cobra" flags defined in the CLI with viper
	_ = directoryViper.BindPFlags(directoryCmd.Flags())
}
