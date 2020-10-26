// Copyright 2020 Artificial Intelligence Redefined <dev+cogment@ai-r.com>
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
	"github.com/spf13/cobra"
	"gitlab.com/cogment/cogment/api"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"runtime"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a command from a cogment.yaml 'commands' section.",
	Args:  cobra.MinimumNArgs(1),
	//The PreRun hook replaces the alias by the final command
	PreRun: func(cmd *cobra.Command, args []string) {
		command := args[0]

		file, err := cmd.Flags().GetString("file")
		if err != nil {
			log.Fatalln(err)
		}

		if _, err := os.Stat(file); os.IsNotExist(err) {
			log.Fatalf("%s doesn't exist", file)
		}

		yamlFile, err := ioutil.ReadFile(file)
		if err != nil {
			log.Printf("yamlFile.Get err   #%v ", err)
		}

		config := api.ProjectConfig{}
		err = yaml.Unmarshal(yamlFile, &config)
		if err != nil {
			log.Fatalf("Unmarshal: %v", err)
		}

		commands := config.Commands

		if _, ok := commands[command]; !ok {
			log.Fatalf("The command %s isn't define in the 'commands' section of %s", command, file)
		}

		args[0] = commands[command]

	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := runRunCmd(args[0]); err != nil {
			log.Fatalln(err)
		}
	},
}

func runRunCmd(command string) error {
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
		log.Fatal(err)
	}

	if _, err := commandFile.Write([]byte(command)); err != nil {
		log.Fatal(err)
	}

	err = commandFile.Close()
	if err != nil {
		log.Fatal(err)
	}

	err = os.Chmod(commandFile.Name(), 0755)
	if err != nil {
		log.Fatal(err)
	}

	defer os.Remove(commandFile.Name())

	cmd := exec.Command(shellCommand, args, commandFile.Name())

	fmt.Printf("Executing %s\n", cmd)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		if Verbose {
			log.Println(command)
			log.Printf("cmd.Run() failed with %s\n", err)
		}
		return err
	}

	return nil

}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringP("file", "f", "cogment.yaml", "project configuration file")

}
