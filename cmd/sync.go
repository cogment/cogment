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

// generateCmd represents the generate command
var syncCmd = &cobra.Command{
	Use:   "sync dir1 dir2 dir3",
	Short: "Sync settings and proto files",
	RunE: func(cmd *cobra.Command, args []string) error {

		hasCogmentYaml := false
		protoFiles := []string{}
		directories := []string{}

		files, err := ioutil.ReadDir(".")
		if err != nil {
			return err
		}

		for _, file := range files {
			if strings.HasSuffix(file.Name(), ".proto") && !file.IsDir() {
				protoFiles = append(protoFiles, file.Name())
			}
			if file.Name() == "cogment.yaml" {
				hasCogmentYaml = true
			}
			if file.IsDir() {
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

		if !hasCogmentYaml {
			return fmt.Errorf("Not a cogment project!")
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

	syncCmd.Flags().BoolP("all", "a", false, "sync all folders")
}
