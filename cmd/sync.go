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
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func copy(src, _dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	dst := filepath.Join(_dst, filepath.Base(src))

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

func isDir(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fileInfo.IsDir()
}

// generateCmd represents the generate command
var syncCmd = &cobra.Command{
	Use:   "sync file1 file2 directory1 dicrectory2",
	Short: "Sync a list of files to a list of directories",
	Long:  "Sync a list of files to a list of directories, order doesn't matter, supports glob format, as in `cogment sync *.proto client`",
	RunE: func(cmd *cobra.Command, args []string) error {

		files := []string{}
		directories := []string{}

		for _, arg := range args {
			fileOrDirs, err := filepath.Glob(arg)
			if err != nil {
				return err
			}
			for _, fileOrDir := range fileOrDirs {
				isDir := isDir(fileOrDir)
				if isDir {
					directories = append(directories, fileOrDir)
				} else {
					files = append(files, fileOrDir)
				}
			}
		}

		if len(files) == 0 {
			return fmt.Errorf("no files to sync")
		}
		if len(directories) == 0 {
			return fmt.Errorf("no directories to sync")
		}

		for _, outputDirectory := range directories {
			for _, inFile := range files {
				message := fmt.Sprintf("Copying %s to %s", inFile, outputDirectory)
				fmt.Println(message)
				_, err := copy(inFile, outputDirectory)
				if err != nil {
					return fmt.Errorf("Copy fail! %v", err)
				}
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
