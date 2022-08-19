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
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cogment/cogment/services/directory"
	"github.com/cogment/cogment/version"
)

const (
	directoryPortKey           = "port"
	directoryPortEnvKey        = "COGMENT_DIRECTORY_PORT"
	directoryGrpcReflectionKey = "grpc_reflection"
	directoryGrpcReflectionEnv = "COGMENT_DIRECTORY_GRPC_REFLECTION"
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
			Port:           directoryViper.GetUint(directoryPortKey),
			GrpcReflection: directoryViper.GetBool(directoryGrpcReflectionKey),
		}

		return directory.Run(options)
	},
}

func init() {
	directoryViper.SetDefault(directoryPortKey, directory.DefaultOptions.Port)
	_ = directoryViper.BindEnv(directoryPortKey, directoryPortEnvKey)
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

	// Don't sort alphabetically, keep insertion order
	directoryCmd.Flags().SortFlags = false

	// Bind "cobra" flags defined in the CLI with viper
	_ = directoryViper.BindPFlags(directoryCmd.Flags())
}
