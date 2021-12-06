// Copyright 2021 AI Redefined Inc. <dev+cogment@ai-r.com>
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
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/cogment/cogment-cli/api"
	"github.com/spf13/cobra"
)

func copy(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// generateCmd represents the generate command
var syncCmd = &cobra.Command{
	Use:   "sync --config cogment.yaml [--all | directories...]",
	Args:  cobra.MinimumNArgs(1),
	Short: "Sync the cogment project settings and proto files",
	Long:  "Sync the cogment project settings and proto files to the target component directories, or to all subdirectories",
	RunE: func(cmd *cobra.Command, args []string) error {

		configFile, err := cmd.Flags().GetString("config")
		if err != nil {
			return fmt.Errorf("Could not get config param! %v", err)
		}

		if configFile == "" {
			return fmt.Errorf("No config file specified!")
		}

		config, err := api.CreateProjectConfigFromYaml(configFile)
		if err != nil {
			return fmt.Errorf("Unable to read config file! %v", err)
		}

		files, err := ioutil.ReadDir(".")
		if err != nil {
			return err
		}

		protoFiles := config.Import.Proto
		directories := []string{}

		for _, file := range files {
			if file.IsDir() && !strings.HasPrefix(file.Name(), ".") {
				directories = append(directories, file.Name())
			}
		}

		outputDirectories := args

		all, err := cmd.Flags().GetBool("all")
		if err != nil {
			return err
		}

		if all {
			outputDirectories = directories
		}

		if len(outputDirectories) == 0 {
			return fmt.Errorf("You must provide at least one directory to sync")
		}

		for _, directory := range outputDirectories {
			for _, protoFile := range protoFiles {
				copy(protoFile, path.Join(directory, protoFile))
			}
			copy("cogment.yaml", path.Join(directory, "cogment.yaml"))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)

	syncCmd.Flags().BoolP("all", "a", false, "apply to all subdirectories")
	syncCmd.Flags().StringP("config", "c", "", "config file to sync")
}
