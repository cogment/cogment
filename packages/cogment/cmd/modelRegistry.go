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
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cogment/cogment/services/modelRegistry"
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
	Use:   "model_registry",
	Short: "Run the model registry",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _args []string) error {
		err := configureLog(servicesViper)
		if err != nil {
			return err
		}

		options := modelRegistry.Options{
			Port:                     registryViper.GetUint(registryPortKey),
			GrpcReflection:           registryViper.GetBool(registryGrpcReflectionKey),
			ArchiveDir:               registryViper.GetString(registryArchiveDirKey),
			CacheMaxItems:            registryViper.GetInt(registryCacheMaxItemsKey),
			SentVersionDataChunkSize: registryViper.GetUint32(registrySentVersionChunkSizeKey),
		}

		return modelRegistry.Run(options)
	},
}

func init() {
	servicesCmd.AddCommand(registryCmd)

	registryViper.SetDefault(registryPortKey, modelRegistry.DefaultOptions.Port)
	_ = registryViper.BindEnv(registryPortKey, "COGMENT_MODEL_REGISTRY_PORT")
	registryCmd.Flags().Uint(registryPortKey, registryViper.GetUint(registryPortKey), "The port to listen on")

	registryViper.SetDefault(registryGrpcReflectionKey, modelRegistry.DefaultOptions.GrpcReflection)
	_ = registryViper.BindEnv(registryGrpcReflectionKey, "COGMENT_MODEL_REGISTRY_GRPC_REFLECTION")
	registryCmd.Flags().Bool(
		registryGrpcReflectionKey,
		registryViper.GetBool(registryGrpcReflectionKey),
		"Start the gRPC reflection server",
	)

	registryViper.SetDefault(registryArchiveDirKey, modelRegistry.DefaultOptions.ArchiveDir)
	_ = registryViper.BindEnv(registryArchiveDirKey, "COGMENT_MODEL_REGISTRY_ARCHIVE_DIR")
	registryCmd.Flags().String(
		registryArchiveDirKey,
		registryViper.GetString(registryArchiveDirKey),
		"The directory to store model archives",
	)

	registryViper.SetDefault(registryCacheMaxItemsKey, modelRegistry.DefaultOptions.CacheMaxItems)
	_ = registryViper.BindEnv(registryCacheMaxItemsKey, "COGMENT_MODEL_REGISTRY_VERSION_CACHE_MAX_ITEMS")
	registryCmd.Flags().Uint32(
		registryCacheMaxItemsKey,
		registryViper.GetUint32(registryCacheMaxItemsKey),
		"Maximum cumulated size of samples size the memory storage "+
			"holds before evicting least recently used trials samples (in bytes)",
	)

	registryViper.SetDefault(registrySentVersionChunkSizeKey, modelRegistry.DefaultOptions.SentVersionDataChunkSize)
	_ = registryViper.BindEnv(
		registrySentVersionChunkSizeKey,
		"COGMENT_MODEL_REGISTRY_SENT_MODEL_VERSION_DATA_CHUNK_SIZE",
	)
	registryCmd.Flags().Uint32(
		registrySentVersionChunkSizeKey,
		registryViper.GetUint32(registrySentVersionChunkSizeKey),
		"The size of the model version data chunk sent by the server (in bytes)",
	)

	// Don't sort alphabetically, keep insertion order
	registryCmd.Flags().SortFlags = false

	// Bind "cobra" flags defined in the CLI with viper
	_ = registryViper.BindPFlags(registryCmd.Flags())
}
