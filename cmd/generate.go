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
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"github.com/golang/protobuf/proto"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/cogment/cogment-cli/api"
	"github.com/cogment/cogment-cli/templates"
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate settings and compile your proto files",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		pythonOutPaths, err := cmd.Flags().GetStringArray("python_dir")
		if err != nil {
			return err
		}
		jsOutPaths, err := cmd.Flags().GetStringArray("js_dir")
		if err != nil {
			return err
		}
		projectConfigPath, err := cmd.Flags().GetString("file")
		if err != nil {
			return err
		}

		if projectConfigPath == "" {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			projectConfigPath, err = api.GetProjectConfigPathFromProjectPath(cwd)
			if err != nil {
				return err
			}
			err = cmd.Flags().Set("file", projectConfigPath)
			if err != nil {
				return err
			}
		}

		for _, pythonOutPath := range pythonOutPaths {
			if _, err := os.Stat(pythonOutPath); os.IsNotExist(err) {
				logger.Fatalf("Python output path %s does not exist: %v", pythonOutPath, err)
			}
		}
		for _, jsOutPath := range jsOutPaths {
			if _, err := os.Stat(jsOutPath); os.IsNotExist(err) {
				logger.Fatalf("Python output path %s does not exist: %v", jsOutPath, err)
			}
		}
		if _, err := os.Stat(projectConfigPath); os.IsNotExist(err) {
			logger.Fatalf("%s doesn't exist: %v", projectConfigPath, err)
		}

		return err
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		color.Yellow("Command 'generate' is deprecated. Please use 'sync' instead.")
		err := runGenerateCmd(cmd)
		if err != nil {
			return err
		}
		logger.Info("Files have been generated")
		return nil
	},
}

func execCommand(command string, directory string, errorString string) error {
	shellCommand := "/bin/sh"

	if runtime.GOOS == "windows" {
		shellCommand = "cmd"
	}

	subProcess := exec.Command(shellCommand)
	subProcess.Dir = directory
	subProcess.Stdout = os.Stdout
	subProcess.Stderr = os.Stderr

	stdin, err := subProcess.StdinPipe()
	if err != nil {
		logger.Fatalf("Unable to open pipe to subprocess stdin: %v", err)
	}
	defer func() {
		if err := stdin.Close(); err != nil {
			logger.Fatalf("%s: %v", errorString, err)
		}
	}()

	if err := subProcess.Start(); err != nil {
		logger.Fatalf("%s: %v", errorString, err)
	}

	if _, err = fmt.Fprintln(stdin, command); err != nil {
		return err
	}

	if err := subProcess.Wait(); err != nil {
		return err
	}

	return nil
}

func runProtoc(protocBinary string, projectRootPath string, protoFiles []string, otherParams []string) error {
	params := append(otherParams, "--proto_path", projectRootPath)
	params = append(params, protoFiles...)

	logger.Debugf("Running `%s %s`", protocBinary, strings.Join(params, " "))

	subProcess := exec.Command(protocBinary, params...)
	subProcess.Dir = projectRootPath
	subProcess.Stdout = os.Stdout
	subProcess.Stderr = os.Stderr

	err := subProcess.Run()
	return err
}

func registerProtos(config *api.ProjectConfig) error {
	logger.Debugf("Registering protos for %s", config.ProjectConfigPath)
	projectRootPath := path.Dir(config.ProjectConfigPath)

	tmpDir, err := ioutil.TempDir("", "registerprotofile")
	if err != nil {
		return err
	}
	defer func() {
		os.RemoveAll(tmpDir)
	}()
	descriptorPath := path.Join(tmpDir, "data.pb")

	// Generating a descriptor file from all the proto files in the project
	err = runProtoc("protoc", projectRootPath, config.Import.Proto, []string{"--descriptor_set_out", descriptorPath})
	if err != nil {
		return err
	}

	// Loading the descriptor file
	descriptorFile, err := ioutil.ReadFile(descriptorPath)
	if err != nil {
		return err
	}

	pbSet := new(descriptorpb.FileDescriptorSet)
	err = proto.Unmarshal(descriptorFile, pbSet)
	if err != nil {
		return err
	}

	// For each of the described file
	for _, pb := range pbSet.GetFile() {
		// Initialized a descriptor of thie file
		fd, err := protodesc.NewFile(pb, protoregistry.GlobalFiles)
		if err != nil {
			return err
		}

		// And register it.
		logger.Debugf("Registering definitions from proto file at '%s'", fd.Path())
		err = protoregistry.GlobalFiles.RegisterFile(fd)
		if err != nil {
			return err
		}
	}
	return nil
}

func clearProtoRegistry() {
	protoregistry.GlobalFiles = &protoregistry.Files{}
}

func runGenerateCmd(cmd *cobra.Command) error {
	projectConfigPath, err := cmd.Flags().GetString("file")
	if err != nil {
		return err
	}
	typescript, err := cmd.Flags().GetBool("typescript")
	if err != nil {
		return err
	}
	pythonOutPaths, err := cmd.Flags().GetStringArray("python_dir")
	if err != nil {
		return err
	}
	jsOutPaths, err := cmd.Flags().GetStringArray("js_dir")
	if err != nil {
		return err
	}

	config, err := api.CreateProjectConfigFromYaml(projectConfigPath)
	if err != nil {
		return err
	}

	config.Typescript = typescript

	return generate(config, pythonOutPaths, jsOutPaths)
}

func generate(config *api.ProjectConfig, pythonOutPaths []string, jsOutPaths []string) error {
	logger.Infof("Generating project configuration for %s", config.ProjectConfigPath)
	typescript := config.Typescript

	config, err := api.CreateProjectConfigFromYaml(config.ProjectConfigPath)
	if err != nil {
		return err
	}

	config.Typescript = typescript

	if err := registerProtos(config); err != nil {
		return err
	}
	defer clearProtoRegistry()

	config, err = updateConfigWithMessage(config)
	if err != nil {
		return err
	}

	if len(pythonOutPaths) > 0 {
		err = generatePython(config, pythonOutPaths)
	}
	if err != nil {
		return err
	}

	if len(jsOutPaths) > 0 {
		err = generateWebClient(config, jsOutPaths)
	}
	return err
}

func generatePython(config *api.ProjectConfig, outputPaths []string) error {
	logger.Infof("Generating python configuration for %s", config.ProjectConfigPath)

	params := []string{}
	for _, outputPath := range outputPaths {
		params = append(params, "--python_out", outputPath)
	}

	err := runProtoc("protoc", path.Dir(config.ProjectConfigPath), config.Import.Proto, params)
	if err != nil {
		return err
	}

	for _, outputPath := range outputPaths {
		settingsFilenameTmpl := fmt.Sprintf("/templates/%s.tmpl", api.SettingsFilenamePy)
		pythonSettingsPath := path.Join(outputPath, api.SettingsFilenamePy)
		logger.Infof("Generating %s/cog_settings.py", pythonSettingsPath)
		if err := os.MkdirAll(outputPath, 0755); err != nil {
			return err
		}

		if err := templates.GenerateFromTemplate(
			settingsFilenameTmpl,
			config,
			pythonSettingsPath,
		); err != nil {
			return fmt.Errorf("error generating python: %v", err)
		}
	}

	return nil
}

func generateWebClient(config *api.ProjectConfig, jsOutPaths []string) error {

	projectRootPath := path.Dir(config.ProjectConfigPath)
	for _, jsOutPath := range jsOutPaths {
		logger.Infof("Generating %s/%s", projectRootPath, jsOutPath)

		if _, err := os.Stat(path.Join(jsOutPath, "node_modules")); os.IsNotExist(err) {
			if _, err := os.Stat(path.Join(jsOutPath, "package.json")); os.IsNotExist(err) {
				return fmt.Errorf("ERROR: no package.json in web-client")
			}
			if err := execCommand(`
				set -xe
				npm i
				exit
			`, jsOutPath, "Error when installing node dependencies"); err != nil {
				return err
			}
		}

		if err := compileProtosJs(config, jsOutPath); err != nil {
			return fmt.Errorf("error generating web-client: %v", err)
		}

		if err := generateCogSettings(config, jsOutPath); err != nil {
			return fmt.Errorf("error generating web client: %v", err)
		}
	}

	return nil
}

func compileProtosJs(config *api.ProjectConfig, jsOutPath string) error {
	params := []string{
		fmt.Sprintf("--js_out=import_style=commonjs,binary:%s/src", jsOutPath),
	}

	if config.Typescript {
		params = append(
			params,
			fmt.Sprintf("--plugin=protoc-gen-ts=%s/node_modules/.bin/protoc-gen-ts", jsOutPath),
			fmt.Sprintf("--ts_out=service=grpc-web:%s/src", jsOutPath),
		)
	}

	jsProtocBinary := path.Join(jsOutPath, "node_modules/.bin/grpc_tools_node_protoc")

	err := runProtoc(jsProtocBinary, path.Dir(config.ProjectConfigPath), config.Import.Proto, params)
	if err != nil {
		return err
	}

	return nil
}

func generateCogSettings(config *api.ProjectConfig, jsOutPath string) error {
	settingsFilenameTmpl := fmt.Sprintf("%s.tmpl", api.SettingsFilenameJs)
	tsSettingsTemplatePath := path.Join("/templates", settingsFilenameTmpl)
	webClientSrcPath := path.Join(jsOutPath, "src")
	cogSettingsPath := path.Join(webClientSrcPath, api.SettingsFilenameJs)

	err := templates.GenerateFromTemplate(tsSettingsTemplatePath, config, cogSettingsPath)

	if err != nil {
		return err
	}

	deleteCommand := fmt.Sprintf("rm src/%s", api.SettingsFilenameJs)

	tscCompileCmd := fmt.Sprintf(
		`
		set -xe
		npx tsc --declaration --declarationMap --outDir src src/%s
		%s
		exit
	`,
		api.SettingsFilenameJs,
		deleteCommand,
	)

	return execCommand(tscCompileCmd, jsOutPath, "Error when creating react app")
}

func updateConfigWithMessage(config *api.ProjectConfig) (*api.ProjectConfig, error) {
	var err error = nil

	for k, actorClass := range config.ActorClasses {
		if actorClass.ConfigType != "" {
			config.ActorClasses[k].ConfigType, err = lookupMessageType(actorClass.ConfigType)
		}

		config.ActorClasses[k].Action.Space, err = lookupMessageType(actorClass.Action.Space)
		config.ActorClasses[k].Observation.Space, err = lookupMessageType(actorClass.Observation.Space)
	}

	if config.Environment != nil && config.Environment.ConfigType != "" {
		config.Environment.ConfigType, err = lookupMessageType(config.Environment.ConfigType)
	}

	if config.Trial != nil && config.Trial.ConfigType != "" {
		config.Trial.ConfigType, err = lookupMessageType(config.Trial.ConfigType)
	}

	return config, err
}

func lookupMessageType(name string) (string, error) {
	s := strings.Split(name, ".")
	protoFile, err := findFileContainingSymbol(name)
	if err != nil {
		return "", err
	}

	return api.ProtoAliasFromProtoPath(protoFile) + "." + s[len(s)-1], nil
}

func findFileContainingSymbol(name string) (string, error) {
	tmp := protoreflect.FullName(name)
	logger.Debugf("Finding proto file containing %s", tmp)

	desc, err := protoregistry.GlobalFiles.FindDescriptorByName(tmp)

	if err != nil {
		return "", fmt.Errorf("Failed to lookup %s", name)
	}

	return desc.ParentFile().Path(), nil
}

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.Flags().StringP("file", "f", "", "path to project config cogment.yaml")
	generateCmd.Flags().StringArrayP("js_dir", "j", []string{}, "javascript output directories (all directories must be valid npm projects, requires a node.js distribution on $PATH)")
	generateCmd.Flags().BoolP("typescript", "t", false, "project uses typescript")
	generateCmd.Flags().StringArrayP("python_dir", "p", []string{}, "python output directories")
}
