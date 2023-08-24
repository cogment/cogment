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
	"github.com/cogment/cogment/services/datastore"
	"github.com/cogment/cogment/version"
)

// datastoreViper represents the configuration of the trial_datastore command
var datastoreViper = viper.New()

var datastorePortKey = "port"
var datastoreGrpcReflectionKey = "grpc_reflection"
var datastoreMemoryStorageMaxSamplesSizeKey = "memory_storage_max_samples_size"
var datastoreFileStoragePathKey = "file_storage"

// datastoreCmd represents the trial_datastore command
var datastoreCmd = &cobra.Command{
	Use:     "trial_datastore",
	Aliases: []string{"datastore"},
	Short:   "Run trial datastore",
	Args:    cobra.NoArgs,
	RunE: func(_cmd *cobra.Command, _args []string) error {
		err := configureLog(servicesViper)
		if err != nil {
			return err
		}

		log.WithFields(logrus.Fields{
			"version": version.Version,
			"hash":    version.Hash,
		}).Info("starting the trial datastore service")

		directoryOptions, err := utils.GetDirectoryRegistrationOptions(datastoreViper)
		if err != nil {
			return err
		}

		options := datastore.Options{
			RegistrationOptions:         directoryOptions,
			Storage:                     datastore.Memory,
			Port:                        datastoreViper.GetUint(datastorePortKey),
			GrpcReflection:              datastoreViper.GetBool(datastoreGrpcReflectionKey),
			MemoryStorageMaxSamplesSize: datastoreViper.GetUint32(datastoreMemoryStorageMaxSamplesSizeKey),
			FileStoragePath:             datastoreViper.GetString(datastoreFileStoragePathKey),
		}

		if datastoreViper.IsSet(datastoreFileStoragePathKey) {
			options.Storage = datastore.File
		}

		ctx := utils.ContextWithUserTermination(context.Background())

		err = datastore.Run(ctx, options)
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
	datastoreViper.SetDefault(datastorePortKey, datastore.DefaultOptions.Port)
	_ = datastoreViper.BindEnv(datastorePortKey, "COGMENT_TRIAL_DATASTORE_PORT")
	datastoreCmd.Flags().Uint(
		datastorePortKey,
		datastoreViper.GetUint(datastorePortKey),
		"The port to listen on",
	)

	datastoreViper.SetDefault(datastoreGrpcReflectionKey, datastore.DefaultOptions.GrpcReflection)
	_ = datastoreViper.BindEnv(datastoreGrpcReflectionKey, "COGMENT_TRIAL_DATASTORE_GRPC_REFLECTION")
	datastoreCmd.Flags().Bool(
		datastoreGrpcReflectionKey,
		datastoreViper.GetBool(datastoreGrpcReflectionKey),
		"Start the gRPC reflection server",
	)

	datastoreViper.SetDefault(
		datastoreMemoryStorageMaxSamplesSizeKey,
		datastore.DefaultOptions.MemoryStorageMaxSamplesSize,
	)
	_ = datastoreViper.BindEnv(
		datastoreMemoryStorageMaxSamplesSizeKey,
		"COGMENT_TRIAL_DATASTORE_MEMORY_STORAGE_MAX_SAMPLE_SIZE",
	)
	datastoreCmd.Flags().Uint32(
		datastoreMemoryStorageMaxSamplesSizeKey,
		datastoreViper.GetUint32(datastoreMemoryStorageMaxSamplesSizeKey),
		"Maximum cumulated size of samples size the memory storage holds "+
			"before evicting least recently used trials samples (in bytes)",
	)

	_ = datastoreViper.BindEnv(datastoreFileStoragePathKey, "COGMENT_TRIAL_DATASTORE_FILE_STORAGE_PATH")
	datastoreCmd.Flags().String(
		datastoreFileStoragePathKey,
		datastoreViper.GetString(datastoreFileStoragePathKey),
		"If provided, the datastore uses a file-based storage instead of "+
			"the default in-memory one with the provided file path as its location",
	)
	if !datastoreViper.IsSet(datastoreFileStoragePathKey) {
		datastoreCmd.Flags().Lookup(datastoreFileStoragePathKey).NoOptDefVal = datastore.DefaultOptions.FileStoragePath
	}

	utils.PopulateDirectoryRegistrationOptionsFlags(
		"TRIAL_DATASTORE",
		datastoreCmd,
		datastoreViper,
		datastore.DefaultOptions.RegistrationOptions,
	)

	// Don't sort alphabetically, keep insertion order
	datastoreCmd.Flags().SortFlags = false

	// Bind "cobra" flags defined in the CLI with viper
	_ = datastoreViper.BindPFlags(datastoreCmd.Flags())
}
