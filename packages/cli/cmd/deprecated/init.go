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
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cogment/cogment/api"
	"github.com/cogment/cogment/templates"
	"github.com/cogment/cogment/utils"
	"github.com/cogment/cogment/version"
)

var JavascriptDependencies = []string{
	"google-protobuf",
	"grpc-tools",
	"grpc_tools_node_protoc_ts",
	"ts-protoc-gen@0.14.0",
}

var JavascriptDevDependencies = []string{
	"tmp",
	"jsdoc",
	"eslint",
	"nps",
	"typescript",
}

var TypescriptDependencies = []string{}

var TypescriptDevDependencies = []string{
	"@types/google-protobuf",
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// InitCmd represents the init command
var InitCmd = &cobra.Command{
	Use:        "init DESTINATION",
	Short:      "Bootstrap a new project locally",
	Deprecated: "this command will be removed in a future version",
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

	RunE: func(cmd *cobra.Command, args []string) error {

		dst := "."
		if len(args) > 0 {
			dst = args[0]
		}

		config, err := createProjectConfigFromReader(os.Stdin)

		if err != nil {
			return err
		}
		// TODO: check for existence of npx, npm

		projectname := strings.Split(dst, "/")
		config.ProjectName = projectname[len(projectname)-1]
		config.Version = version.Version
		config.ProjectConfigPath = path.Join(dst, "cogment.yaml")

		err = createProjectFiles(config)

		return err
	},
}

func createProjectFiles(config *api.ProjectConfig) error {
	log.Info("Creating project")

	err := createApp(config)
	if err != nil {
		return err
	}

	if config.WebClient {
		log.Info("Creating web-client")
		err = createWebClient(config)
		if err != nil {
			return err
		}
	}

	pythonOutPaths := []string{api.EnvironmentServiceName, api.ClientServiceName}
	for _, service := range config.ListServiceActorServices() {
		pythonOutPaths = append(pythonOutPaths, utils.Snakeify(service.Name))
	}
	return nil
}

func createApp(config *api.ProjectConfig) error {
	log.Info("Scaffolding application")
	projectRootPath := path.Dir(config.ProjectConfigPath)
	log.Infof("Creating project structure at %s", projectRootPath)
	err := templates.RecursivelyGenerateFromTemplates(
		"/templates",
		[]string{
			"ACTOR_SERVICE_NAME",
			"cog_settings*",
			"CogSettings*",
			"web-client",
		},
		config,
		projectRootPath,
	)
	if err != nil {
		return err
	}

	for _, service := range config.ListServiceActorServices() {
		actorServiceTemplateConfig := map[string]interface{}{
			"Project": config,
			"Service": service,
		}

		if err := templates.RecursivelyGenerateFromTemplates(
			"/templates/ACTOR_SERVICE_NAME",
			[]string{},
			actorServiceTemplateConfig,
			path.Join(projectRootPath, utils.Snakeify(service.Name)),
		); err != nil {
			return err
		}
	}
	log.Info("Scaffolding application complete")
	return nil
}

func createWebClient(config *api.ProjectConfig) error {
	log.Info("Creating web-client from npm")
	projectRootPath := path.Dir(config.ProjectConfigPath)

	shellCommand := "/bin/sh"

	if runtime.GOOS == "windows" {
		shellCommand = "cmd"
	}

	createReactAppCmd := "npm init react-app web-client"
	npmDependencies := JavascriptDependencies[:]
	npmDependencies = append(
		npmDependencies,
		fmt.Sprintf("%s@%s", config.Components.Javascript.Package, config.Components.Javascript.Version),
	)
	npmDevDependencies := JavascriptDevDependencies[:]

	if !commandExists("npm") {
		return fmt.Errorf("you must have a working installation of both node and npm in order to create a web client")
	}

	if config.Typescript {
		createReactAppCmd = fmt.Sprintf("%s --template typescript", createReactAppCmd)
		npmDependencies = append(npmDependencies, TypescriptDependencies...)
		npmDevDependencies = append(npmDevDependencies, TypescriptDevDependencies...)
	}

	log.Debug(createReactAppCmd)

	installJavascriptCmd := fmt.Sprintf(
		`
		set -xe
		%s
		cd web-client
		npm install --save --no-audit %s
		npm install --save-dev --no-audit %s
		npx nps init
		rm -rf .git
		exit
		`,
		createReactAppCmd,
		strings.Join(npmDependencies, " "),
		strings.Join(npmDevDependencies, " "),
	)

	subProcess := exec.Command(shellCommand)
	subProcess.Dir = projectRootPath
	subProcess.Stdout = os.Stdout
	subProcess.Stderr = os.Stderr

	stdin, err := subProcess.StdinPipe()
	if err != nil {
		return err
	}
	defer func() {
		if err := stdin.Close(); err != nil {
			log.Fatalf("Error when creating react app: %v", err)
		}
	}()

	if err := subProcess.Start(); err != nil {
		log.Fatalf("Error in init while creating react app %v", err)
	}

	if _, err = fmt.Fprint(stdin, installJavascriptCmd); err != nil {
		return err
	}

	if err := subProcess.Wait(); err != nil {
		return err
	}

	path.Join(projectRootPath, "web-client", ".git")

	webClientRootPath := path.Join(projectRootPath, "web-client")

	log.Info("Create web-client structure")

	// Ignore ts files when generating javascript
	ignore := []string{"**/useActions.ts*"}
	if config.Typescript {
		// Ignore js files when generating typescript
		ignore = []string{"**/useActions.js*"}
	}

	if err := templates.RecursivelyGenerateFromTemplates(
		"/templates/web-client",
		ignore,
		config,
		webClientRootPath,
	); err != nil {
		log.Fatalf("Error generating web-client: %v", err)
	}

	return nil
}

func integerFromReader(reader *bufio.Reader, prompt string, defaultValue *int, validate func(int) (
	int,
	error,
), retryCount int) (int, error) {
	for i := 0; i < retryCount; i++ {
		fmt.Print(prompt)
		inputStr, _ := reader.ReadString('\n')
		inputStr = strings.TrimSpace(inputStr)
		if inputStr == "" && defaultValue != nil {
			return *defaultValue, nil
		}
		inputInt, err := strconv.Atoi(inputStr)
		if err != nil {
			fmt.Printf("\t'%s' is invalid, expecting an integer.\n", inputStr)
		} else if validatedInputInt, err := validate(inputInt); err != nil {
			fmt.Printf("\t%s\n", err)
		} else {
			return validatedInputInt, nil
		}
	}
	return 0, fmt.Errorf("invalid user input")
}

func stringFromReader(reader *bufio.Reader, prompt string, defaultValue *string, validate func(string) (
	string,
	error,
), retryCount int) (string, error) {
	for i := 0; i < retryCount; i++ {
		fmt.Print(prompt)
		inputStr, _ := reader.ReadString('\n')
		inputStr = strings.TrimSpace(inputStr)
		if inputStr == "" && defaultValue != nil {
			return *defaultValue, nil
		}
		if validatedInputStr, err := validate(inputStr); err != nil {
			fmt.Printf("\t%s\n", err)
		} else {
			return validatedInputStr, nil
		}
	}
	return "", fmt.Errorf("invalid user input")
}

func validatePositiveNumber(input int) (int, error) {
	if input < 0 {
		return 0, fmt.Errorf("'%d' is invalid, expecting a positive value", input)
	}
	return input, nil
}

var yesNoAnswers = map[string]string{
	"y":   "Y",
	"yes": "Y",
	"n":   "N",
	"no":  "N",
}

func validateYesNoAnswer(input string) (string, error) {
	validatedInput, inputIsValid := yesNoAnswers[strings.ToLower(input)]
	if !inputIsValid {
		return "", fmt.Errorf("'%s' is invalid, expecting Y or N", input)
	}
	return validatedInput, nil
}

func createValidateName(existingNames []string) func(string) (string, error) {
	return func(input string) (string, error) {
		if input == "" {
			return "", fmt.Errorf("'%s' is invalid, expecting an non-empty name", input)
		}
		if strings.Contains("0123456789", input[0:1]) {
			return "", fmt.Errorf("'%s' is invalid, expecting to not start with numeric character", input)
		}
		validatedInput := utils.Snakeify(input)
		for _, existingName := range existingNames {
			if validatedInput == existingName {
				return "", fmt.Errorf("'%s' is invalid, expecting a unique name", input)
			}
		}
		return validatedInput, nil
	}
}

func createProjectConfigFromReader(stdin io.Reader) (*api.ProjectConfig, error) {
	reader := bufio.NewReader(stdin)

	config, err := api.ExtendDefaultProjectConfig(&api.ProjectConfig{TrialParams: &api.TrialParams{}})
	if err != nil {
		return nil, err
	}

	actorClassesCount, err := integerFromReader(
		reader,
		"Enter how many actor classes should be created: ",
		nil,
		validatePositiveNumber,
		3,
	)
	if err != nil {
		return nil, err
	}

	var actorClassNames []string
	var serviceImplNames []string
	connectedImplCreated := false
	for classIdx := 0; classIdx < actorClassesCount; classIdx++ {
		className, err := stringFromReader(
			reader,
			fmt.Sprintf("[class %d] Enter the name of the class: ", classIdx+1),
			nil,
			createValidateName(actorClassNames),
			3,
		)
		if err != nil {
			return nil, err
		}
		class := api.ActorClass{Name: className}
		config.ActorClasses = append(config.ActorClasses, &class)
		actorClassNames = append(actorClassNames, className)

		defaultServiceImplCount := 1
		serviceImplCount, err := integerFromReader(
			reader,
			fmt.Sprintf(
				"[class #%d '%s'] Enter the number of service implementations that should be created (empty for 1): ",
				classIdx+1,
				className,
			),
			&defaultServiceImplCount,
			validatePositiveNumber,
			3,
		)
		if err != nil {
			return nil, err
		}

		for implIdx := 0; implIdx < serviceImplCount; implIdx++ {
			implName, err := stringFromReader(
				reader,
				fmt.Sprintf(
					"[class #%d '%s' > service impl. #%d] Enter the name of the implementation: ",
					classIdx+1,
					className,
					implIdx+1,
				),
				nil,
				createValidateName(serviceImplNames),
				3,
			)
			if err != nil {
				return nil, err
			}
			serviceImplNames = append(serviceImplNames, implName)

			defaultActorsCount := 1
			actorsCount, err := integerFromReader(
				reader,
				fmt.Sprintf(
					"[class #%d '%s' > service impl. #%d '%s'] "+
						"Enter the number of actor instances using this implementation (empty for 1): ",
					classIdx+1,
					className,
					implIdx+1,
					implName,
				),
				&defaultActorsCount,
				validatePositiveNumber,
				3,
			)
			if err != nil {
				return nil, err
			}

			for actorIdx := 0; actorIdx < actorsCount; actorIdx++ {
				actorName := fmt.Sprintf(
					"%s_%s_%d",
					className,
					implName,
					actorIdx+1,
				)
				actor := api.TrialActor{
					Name:           actorName,
					ActorClass:     className,
					Implementation: implName,
					Endpoint:       "grpc://" + utils.Kebabify(implName) + ":9000",
				}

				config.TrialParams.Actors = append(config.TrialParams.Actors, &actor)
			}
		}

		if !connectedImplCreated {
			defaultCreateClientImpl := "Y"
			createClientImpl, err := stringFromReader(
				reader,
				fmt.Sprintf(
					"[class #%d '%s'] Should a client implementation be created (Y or N, empty for Y): ",
					classIdx+1,
					className,
				),
				&defaultCreateClientImpl,
				validateYesNoAnswer,
				3,
			)
			if err != nil {
				return nil, err
			}

			if createClientImpl == "Y" {
				// cogment init only creates a single client actor, using default values
				actorName := "client_actor"
				implName := "client_actor_impl"

				actor := api.TrialActor{
					Name:           actorName,
					ActorClass:     className,
					Implementation: implName,
					Endpoint:       api.ClientActorServiceEndpoint,
				}

				config.TrialParams.Actors = append(config.TrialParams.Actors, &actor)

				connectedImplCreated = true
			}
		}
	}

	defaultCreateWebClient := "Y"
	createWebClient, err := stringFromReader(
		reader,

		"Should a web-client be created (Y or N, empty for Y): ",
		&defaultCreateWebClient,
		validateYesNoAnswer,
		3,
	)
	if err != nil {
		return nil, err
	}

	config.WebClient = createWebClient == "Y"

	if createWebClient == "Y" {
		defaultTypescriptWebClient := "Y"
		typescriptWebClient, err := stringFromReader(
			reader,
			"Should the web-client use Typescript (Y or N, empty for Y): ",
			&defaultTypescriptWebClient,
			validateYesNoAnswer,
			3,
		)
		if err != nil {
			return nil, err
		}

		config.Typescript = typescriptWebClient == "Y"
	}

	return config, nil
}
