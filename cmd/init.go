/*
Copyright © 2019 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"gitlab.com/cogment/cogment/api"
	"gitlab.com/cogment/cogment/helper"
	"gitlab.com/cogment/cogment/templates"
	"gitlab.com/cogment/cogment/templates/agents"
	"gitlab.com/cogment/cogment/templates/clients"
	"gitlab.com/cogment/cogment/templates/envs"
	"gitlab.com/cogment/cogment/templates/orchestrator"
	"html/template"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
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

		err = createProjectFiles(dst, config)
		if err != nil {
			log.Fatalln(err)
		}
	},
}

func createProjectFiles(dst string, config *api.ProjectConfig) error {

	if err := generateFromTemplate(templates.ROOT_README, dst+"/README.md", config); err != nil {
		return err
	}

	if err := generateFromTemplate(templates.ROOT_DOCKER_COMPOSE, dst+"/docker-compose.yaml", config); err != nil {
		return err
	}

	if err := generateFromTemplate(templates.ROOT_DATA_PROTO, dst+"/data.proto", config); err != nil {
		return err
	}

	if err := generateFromTemplate(templates.ROOT_COGMENT_YAML, dst+"/cogment.yaml", config); err != nil {
		return err
	}

	if err := generateFromTemplate(templates.ROOT_GITIGNORE, dst+"/.gitignore", config); err != nil {
		return err
	}

	if err := generateFromTemplate(orchestrator.DOCKERFILE, dst+"/orchestrator/Dockerfile", config); err != nil {
		return err
	}

	if err := generateFromTemplate(envs.DOCKERFILE, dst+"/envs/Dockerfile", config); err != nil {
		return err
	}

	if err := generateFromTemplate(envs.MAIN_PY, dst+"/envs/main.py", config); err != nil {
		return err
	}

	if err := generateFromTemplate(clients.DOCKERFILE, dst+"/clients/Dockerfile", config); err != nil {
		return err
	}

	if err := generateFromTemplate(clients.MAIN_PY, dst+"/clients/main.py", config); err != nil {
		return err
	}

	for k, actor := range config.ActorClasses {
		countAi, _ := config.CountActorsByActorClass(config.ActorClasses[k].Id)
		if countAi < 1 {
			continue
		}

		name := helper.Snakeify(actor.Id)
		if err := generateFromTemplate(agents.DOCKERFILE, path.Join(dst, "agents", name, "Dockerfile"), name); err != nil {
			return err
		}

		if err := generateFromTemplate(agents.MAIN_PY, path.Join(dst, "agents", name, "main.py"), name); err != nil {
			return err
		}
	}

	cmd := generateCmd
	//cmd := cobra.Command{}
	args := []string{"--file", dst + "/cogment.yaml", "--python_dir", dst}

	//cmd.SetArgs([]string{"--file", dst + "/cogment.yaml", "--python-dir", dst})1
	//cmd.ParseFlags()
	cmd.ParseFlags(args)
	err := runGenerateCmd(cmd)

	//err := cmd.Execute()

	return err
}

func generateFromTemplate(tplStr, dst string, config interface{}) error {
	t := template.New(dst).Funcs(template.FuncMap{
		"snakeify":  helper.Snakeify,
		"kebabify":  helper.Kebabify,
		"pascalify": helper.Pascalify,
	})

	t = template.Must(t.Parse(tplStr))

	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}

	f, err := os.Create(dst)
	if err != nil {
		return err
	}

	if err = t.Execute(f, config); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}

	return err
}

func createProjectConfigFromReader(stdin io.Reader) (*api.ProjectConfig, error) {
	reader := bufio.NewReader(stdin)

	config := api.ProjectConfig{TrialParams: &api.TrialParams{}}

	nbActors, err := getTotalActorsFromReader(reader)
	if err != nil {
		return nil, err
	}

	totalHuman := 0
	for len(config.ActorClasses) < nbActors {

		name, err := getActorClassNameFromReader(reader)
		for _, val := range config.ActorClasses {
			if name == val.Id {
				err = fmt.Errorf("this name is already taken")
			}
		}
		if err != nil {
			fmt.Println(err)
			continue
		}

		nbAi, nbHuman := getActorClassFromReader(reader, name)
		actorClass := api.ActorClass{Id: name}

		for i := 0; i < nbAi; i++ {
			actor := api.Actor{
				ActorClass: name,
				Endpoint:   "grpc://" + helper.Kebabify(name) + ":9000",
			}

			config.TrialParams.Actors = append(config.TrialParams.Actors, &actor)
		}

		for i := 0; i < nbHuman; i++ {
			actor := api.Actor{
				ActorClass: name,
				Endpoint:   "human",
			}

			config.TrialParams.Actors = append(config.TrialParams.Actors, &actor)
		}

		if totalHuman+nbHuman > 1 {
			fmt.Println("cogment doesn't support more than 1 human now")
			continue
		}
		totalHuman += nbHuman

		config.ActorClasses = append(config.ActorClasses, &actorClass)
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

func init() {
	rootCmd.AddCommand(initCmd)
}
