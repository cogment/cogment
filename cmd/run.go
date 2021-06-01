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
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"runtime"

	"github.com/cogment/cogment-cli/api"
	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:          "run",
	Short:        "Run a command from a cogment.yaml 'commands' section.",
	Args:         cobra.MinimumNArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		commandName := args[0]

		yamlFile, err := cmd.Flags().GetString("file")
		if err != nil {
			return fmt.Errorf("Unable to retrieve the project config file: %w", err)
		}

		workingDir := path.Dir(yamlFile)

		config, err := api.CreateProjectConfigFromYaml(yamlFile)
		if err != nil {
			return fmt.Errorf("Unable to deserialize the project config from '%s': %w", yamlFile, err)
		}

		command, ok := config.Commands[commandName]

		if !ok {
			return fmt.Errorf("Unknown command '%s': it is not define in the 'commands' section of '%s'", commandName, yamlFile)
		}

		log.Printf("Executing %s in %s\n", command, workingDir)
		err = runCommand(command, workingDir)
		if err != nil {
			return fmt.Errorf("Error while running command '%s': %w", commandName, err)
		}
		return nil
	},
}

func runCommand(command string, workingDir string) error {
	shellCommand := "/bin/sh"
	args := "-c"
	fileExtension := "sh" // File extension is required on windows otherwise "cmd /C" won't work.

	if runtime.GOOS == "windows" {
		shellCommand = "cmd"
		args = "/C"
		fileExtension = "cmd"
	}

	commandFile, err := ioutil.TempFile("", "cogment-cli-*."+fileExtension)
	if err != nil {
		return err
	}

	if _, err := commandFile.Write([]byte(command)); err != nil {
		return err
	}

	err = commandFile.Close()
	if err != nil {
		return err
	}

	err = os.Chmod(commandFile.Name(), 0755)
	if err != nil {
		return err
	}

	defer os.Remove(commandFile.Name())

	cmd := exec.Command(shellCommand, args, commandFile.Name())
	cmd.Dir = workingDir

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("'%s' failed (%w)", command, err)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringP("file", "f", "cogment.yaml", "project configuration file")

}
