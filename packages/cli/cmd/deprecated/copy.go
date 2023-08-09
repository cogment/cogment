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

package deprecated

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

//nolint:all
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

// CopyCmd represents the copy command
var CopyCmd = &cobra.Command{
	Use:   "copy file_or_folder	...",
	Short: "Copy all files listed to all existing folders listed",
	Long: "Copy all files listed to all existing folders listed, " +
		"order doesn't matter, supports glob format, as in `cogment copy *.proto client`",
	Deprecated: "this command will be removed in a future version, consider using \"cp\"",
	RunE: func(cmd *cobra.Command, args []string) error {

		files := []string{}
		folders := []string{}

		for _, arg := range args {
			fileOrDirs, err := filepath.Glob(arg)
			if err != nil {
				return err
			}
			for _, fileOrDir := range fileOrDirs {
				isDir := isDir(fileOrDir)
				if isDir {
					folders = append(folders, fileOrDir)
				} else {
					files = append(files, fileOrDir)
				}
			}
		}

		if len(files) == 0 {
			return fmt.Errorf("no files to copy")
		}
		if len(folders) == 0 {
			return fmt.Errorf("no folders to copy")
		}

		for _, outputDirectory := range folders {
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
