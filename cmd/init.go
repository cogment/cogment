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
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"gitlab.com/cogment/cogment/api"
	"gitlab.com/cogment/cogment/helper"
	"gitlab.com/cogment/cogment/templates"
	"gitlab.com/cogment/cogment/version"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init DESTINATION",
	Short: "Bootstrap a new project locally",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("requires a destination to create")
		}

		dest := args[0]
		if _, err := os.Stat(dest); !os.IsNotExist(err) {
			return fmt.Errorf("destination %s already exists", dest)
		}

		return nil
	},

	Run: func(cmd *cobra.Command, args []string) {

		dst := "."
		if len(args) > 0 {
			dst = args[0]
		}

		config, err := createProjectConfigFromReader(os.Stdin)
		if err != nil {
			log.Fatalln(err)
		}

		projectname := strings.Split(dst, "/")
		config.ProjectName = projectname[len(projectname)-1]
		config.CliVersion = version.CliVersion

		err = createProjectFiles(dst, config)
		if err != nil {
			log.Fatalln(err)
		}
	},
}

func createProjectFiles(dst string, config *api.ProjectConfig) error {

	err := templates.RecursivelyGenerateFromTemplates(
		"/templates",
		[]string{
			"agents/AGENT_NAME",
			"cog_settings*",
		},
		config,
		dst)
	if err != nil {
		return err
	}

	for k, actor := range config.ActorClasses {
		countAi, _ := config.CountActorsByActorClass(config.ActorClasses[k].Id)
		if countAi < 1 {
			continue
		}

		err := templates.RecursivelyGenerateFromTemplates("/templates/agents/AGENT_NAME", []string{}, actor, path.Join(dst, "agents", helper.Snakeify(actor.Id)))
		if err != nil {
			return err
		}
	}

	cmd := generateCmd
	//cmd := cobra.Command{}
	args := []string{"--file", dst + "/cogment.yaml", "--python_dir", dst}

	//cmd.SetArgs([]string{"--file", dst + "/cogment.yaml", "--python-dir", dst})1
	//cmd.ParseFlags()
	cmd.ParseFlags(args)
	err = runGenerateCmd(cmd)

	//err := cmd.Execute()

	return err
}

func createProjectConfigFromReader(stdin io.Reader) (*api.ProjectConfig, error) {
	reader := bufio.NewReader(stdin)

	config := api.ProjectConfig{TrialParams: &api.TrialParams{}}

	name, err := getClientNameFromReader(reader)
	if err != nil {
		return nil, err
	}

	actorClass := api.ActorClass{Id: name}

	config.ActorClasses = append(config.ActorClasses, &actorClass)

	actor := api.Actor{
		ActorClass: name,
		Endpoint:   "human",
	}

	config.TrialParams.Actors = append(config.TrialParams.Actors, &actor)

	nbAgentActors, err := getTotalAgentsFromReader(reader)
	if err != nil {
		return nil, err
	}

	for i := 0; i < nbAgentActors; i++ {

		name, err := getAgentClassNameFromReader(reader, i)
		for _, val := range config.ActorClasses {
			if name == val.Id {
				err = fmt.Errorf("this name is already taken")
			}
		}
		if err != nil {
			return nil, err
		}

		actorClass := api.ActorClass{Id: name}

		config.ActorClasses = append(config.ActorClasses, &actorClass)

		nbAgentInstances, err := getAgentInstantsFromReader(reader, name)
		if err != nil {
			return nil, err
		}

		for j := 0; j < nbAgentInstances; j++ {
			actor := api.Actor{
				ActorClass: name,
				Endpoint:   "grpc://" + helper.Kebabify(name) + ":9000",
			}

			config.TrialParams.Actors = append(config.TrialParams.Actors, &actor)

		}

	}

	return &config, nil
}

func getActorClassFromReader(reader *bufio.Reader, name string) (totalAi, totalHuman int) {

	for {
		fmt.Printf("Number of AI \"%s\": ", name)
		result, err := getIntegerFromReader(reader)
		if err == nil && result >= 0 {
			totalAi = result
			break
		}

		fmt.Println("This value must be an integer >= 0")
	}

	for {
		fmt.Printf("Number of Human \"%s\": ", name)
		result, err := getIntegerFromReader(reader)
		if err == nil && result >= 0 {
			totalHuman = result
			break
		}

		fmt.Println("This value must be an integer >= 0")
	}

	return totalAi, totalHuman
}

func getIntegerFromReader(reader *bufio.Reader) (int, error) {
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	result, err := strconv.Atoi(input)
	if err != nil {
		return 0, err
	}
	return result, nil
}

func getTotalActorsFromReader(reader *bufio.Reader) (int, error) {

	for {
		fmt.Print("Number of actor types: ")

		result, err := getIntegerFromReader(reader)

		if err == nil && result > 0 {
			return result, nil
		}

		fmt.Println("this value must be an integer greater than 0")
	}
}

func getActorClassNameFromReader(reader *bufio.Reader) (string, error) {
	fmt.Printf("Actor name: ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return input, fmt.Errorf("name can't be empty")
	}

	return input, nil
}

func getClientNameFromReader(reader *bufio.Reader) (string, error) {
	fmt.Printf("Master client actor name: ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return input, fmt.Errorf("name can't be empty")
	}
	if input != strings.ToLower(input) {
		return input, fmt.Errorf("no upper case letters")
	}
	if strings.Contains(input, "-") {
		return input, fmt.Errorf("no dashes/hyphens")
	}
	if strings.Contains("0123456789", input[0:1]) {
		return input, fmt.Errorf("name can't start with numeric")
	}

	return input, nil
}

func getAgentClassNameFromReader(reader *bufio.Reader, agentNumber int) (string, error) {
	fmt.Printf("Agent actor type " + strconv.Itoa(agentNumber+1) + " name: ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return input, fmt.Errorf("name can't be empty")
	}
	if input != strings.ToLower(input) {
		return input, fmt.Errorf("no upper case letters")
	}
	if strings.Contains(input, "-") {
		return input, fmt.Errorf("no dashed/hyphens")
	}
	if strings.Contains("0123456789", input[0:1]) {
		return input, fmt.Errorf("name can't start with numeric")
	}

	return input, nil
}

func getTotalAgentsFromReader(reader *bufio.Reader) (int, error) {

	for {
		fmt.Print("Number of agent actor types: ")

		result, err := getIntegerFromReader(reader)

		if err == nil && result > 0 {
			return result, nil
		}

		fmt.Println("this value must be an integer greater than 0")
	}
}

func getAgentInstantsFromReader(reader *bufio.Reader, agentName string) (int, error) {

	for {
		fmt.Print("Number of agent '" + agentName + "' instances: ")

		result, err := getIntegerFromReader(reader)

		if err == nil && result > 0 {
			return result, nil
		}

		fmt.Println("this value must be an integer greater than 0")
	}
}

func init() {
	rootCmd.AddCommand(initCmd)
}
