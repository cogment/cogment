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
	"github.com/cogment/cogment/services/registry"
	"github.com/cogment/cogment/version"
)

// registryViper represents the configuration of the model_registry command
var registryViper = viper.New()

var registryPortKey = "port"
var registryGrpcReflectionKey = "grpc_reflection"
var registryArchiveDirKey = "archive_dir"
var registryCacheMaxItemsKey = "cache_max_items"
var registrySentVersionChunkSizeKey = "sent_version_chunk_size"

// registryCmd represents the model_registry
var registryCmd = &cobra.Command{
	Use:     "model_registry",
	Aliases: []string{"registry"},
	Short:   "Run the model registry",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _args []string) error {
		err := configureLog(servicesViper)
		if err != nil {
			return err
		}

		log.WithFields(logrus.Fields{
			"version": version.Version,
			"hash":    version.Hash,
		}).Info("starting the model registry service")

		directoryOptions, err := utils.GetDirectoryRegistrationOptions(registryViper)
		if err != nil {
			return err
		}

		options := registry.Options{
			RegistrationOptions:      directoryOptions,
			Port:                     registryViper.GetUint(registryPortKey),
			GrpcReflection:           registryViper.GetBool(registryGrpcReflectionKey),
			ArchiveDir:               registryViper.GetString(registryArchiveDirKey),
			CacheMaxItems:            registryViper.GetInt(registryCacheMaxItemsKey),
			SentVersionDataChunkSize: registryViper.GetInt(registrySentVersionChunkSizeKey),
		}

		ctx := utils.ContextWithUserTermination(context.Background())

		err = registry.Run(ctx, options)
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
	registryViper.SetDefault(registryPortKey, registry.DefaultOptions.Port)
	_ = registryViper.BindEnv(registryPortKey, "COGMENT_MODEL_REGISTRY_PORT")
	registryCmd.Flags().Uint(registryPortKey, registryViper.GetUint(registryPortKey), "The port to listen on")

	registryViper.SetDefault(registryGrpcReflectionKey, registry.DefaultOptions.GrpcReflection)
	_ = registryViper.BindEnv(registryGrpcReflectionKey, "COGMENT_MODEL_REGISTRY_GRPC_REFLECTION")
	registryCmd.Flags().Bool(
		registryGrpcReflectionKey,
		registryViper.GetBool(registryGrpcReflectionKey),
		"Start the gRPC reflection server",
	)

	registryViper.SetDefault(registryArchiveDirKey, registry.DefaultOptions.ArchiveDir)
	_ = registryViper.BindEnv(registryArchiveDirKey, "COGMENT_MODEL_REGISTRY_ARCHIVE_DIR")
	registryCmd.Flags().String(
		registryArchiveDirKey,
		registryViper.GetString(registryArchiveDirKey),
		"The directory to store model archives",
	)

	registryViper.SetDefault(registryCacheMaxItemsKey, registry.DefaultOptions.CacheMaxItems)
	_ = registryViper.BindEnv(registryCacheMaxItemsKey, "COGMENT_MODEL_REGISTRY_VERSION_CACHE_MAX_ITEMS")
	registryCmd.Flags().Uint32(
		registryCacheMaxItemsKey,
		registryViper.GetUint32(registryCacheMaxItemsKey),
		"Maximum number of model versions the memory storage holds before evicting least recently used",
	)

	registryViper.SetDefault(registrySentVersionChunkSizeKey, registry.DefaultOptions.SentVersionDataChunkSize)
	_ = registryViper.BindEnv(
		registrySentVersionChunkSizeKey,
		"COGMENT_MODEL_REGISTRY_SENT_MODEL_VERSION_DATA_CHUNK_SIZE",
	)
	registryCmd.Flags().Int(
		registrySentVersionChunkSizeKey,
		registryViper.GetInt(registrySentVersionChunkSizeKey),
		"The size of the model version data chunk sent by the server (in bytes)",
	)

	utils.PopulateDirectoryRegistrationOptionsFlags(
		"MODEL_REGISTRY",
		registryCmd,
		registryViper,
		registry.DefaultOptions.RegistrationOptions,
	)

	// Don't sort alphabetically, keep insertion order
	registryCmd.Flags().SortFlags = false

	// Bind "cobra" flags defined in the CLI with viper
	_ = registryViper.BindPFlags(registryCmd.Flags())
}
