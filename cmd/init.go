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
	"github.com/spf13/cobra"
	"gitlab.com/cogment/cogment/api"
	"gitlab.com/cogment/cogment/helper"
	"gitlab.com/cogment/cogment/templates"
	"gitlab.com/cogment/cogment/templates/agents"
	"gitlab.com/cogment/cogment/templates/clients"
	"gitlab.com/cogment/cogment/templates/configs"
	"gitlab.com/cogment/cogment/templates/configurator"
	"gitlab.com/cogment/cogment/templates/envoy"
	"gitlab.com/cogment/cogment/templates/envs"
	"gitlab.com/cogment/cogment/templates/js_clients"
	"gitlab.com/cogment/cogment/templates/orchestrator"
	"gitlab.com/cogment/cogment/templates/replaybuffer"
	"gitlab.com/cogment/cogment/templates/revisioning"
	"gitlab.com/cogment/cogment/version"
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

	if err := generateFromTemplate(templates.ROOT_README, dst+"/README.md", config); err != nil {
		return err
	}

	if err := generateFromTemplate(templates.ROOT_DOCKER_COMPOSE, dst+"/docker-compose.yaml", config); err != nil {
		return err
	}

	if err := generateFromTemplate(templates.ROOT_DOCKER_COMPOSE_OVERRIDE, dst+"/docker-compose.override.template.yaml", config); err != nil {
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

	if err := generateFromTemplate(envoy.ENVOY_YAML, dst+"/envoy/envoy.yaml", config); err != nil {
		return err
	}

	if err := generateFromTemplate(js_clients.MAIN_JS, dst+"/clients/js/main.js", config); err != nil {
		return err
	}

	if err := generateFromTemplate(js_clients.INDEX_HTML, dst+"/clients/js/index.html", config); err != nil {
		return err
	}

	if err := generateFromTemplate(js_clients.PACKAGE_JSON, dst+"/clients/js/package.json", config); err != nil {
		return err
	}

	if err := generateFromTemplate(js_clients.WEBPACK_CONFIG_JS, dst+"/clients/js/webpack.config.js", config); err != nil {
		return err
	}

	if err := generateFromTemplate(js_clients.DOCKERFILE, dst+"/clients/js/Dockerfile", config); err != nil {
		return err
	}

	if err := generateFromTemplate(configs.PROMETHEUS_YAML, dst+"/configs/prometheus/prometheus.yaml", config); err != nil {
		return err
	}

	if err := generateFromTemplate(configs.GRAFANA_DASHBOARD_JSON, dst+"/configs/grafana/dashboards/dashboard.json", config); err != nil {
		return err
	}

	if err := generateFromTemplate(configs.GRAFANA_DASHBOARD_YAML, dst+"/configs/grafana/dashboards/dashboard.yaml", config); err != nil {
		return err
	}

	if err := generateFromTemplate(configs.GRAFANA_DATASOURCE_YAML, dst+"/configs/grafana/datasources/datasource.yaml", config); err != nil {
		return err
	}

	if err := generateFromTemplate(replaybuffer.DOCKERFILE, dst+"/replaybuffer/Dockerfile", config); err != nil {
		return err
	}

	if err := generateFromTemplate(replaybuffer.REPLAYBUFFER_PY, dst+"/replaybuffer/replaybuffer.py", config); err != nil {
		return err
	}

	if err := generateFromTemplate(revisioning.REQUIREMENTS_TXT, dst+"/requirements.txt", config); err != nil {
		return err
	}

	if err := generateFromTemplate(revisioning.COGMENT_SH, dst+"/cogment.sh", config); err != nil {
		return err
	}

	if err := generateFromTemplate(configurator.DOCKERFILE, dst+"/configurator/Dockerfile", config); err != nil {
		return err
	}

	if err := generateFromTemplate(configurator.CONFIGURATOR_PY, dst+"/configurator/main.py", config); err != nil {
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
		"tocaps":    helper.Tocaps,
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

func getAgentClassNameFromReader(reader *bufio.Reader, agent_number int) (string, error) {
	fmt.Printf("Agent actor type " + strconv.Itoa(agent_number+1) + " name: ")
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

func getAgentInstantsFromReader(reader *bufio.Reader, agent_name string) (int, error) {

	for {
		fmt.Print("Number of agent '" + agent_name + "' instances: ")

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
