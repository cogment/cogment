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
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/cogment/cogment-cli/api"
	"github.com/cogment/cogment-cli/helper"
	"github.com/cogment/cogment-cli/templates"
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate settings and compile your proto files",
	PreRun: func(cmd *cobra.Command, args []string) {
		pythonOutPaths, err := cmd.Flags().GetStringArray("python_dir")
		helper.CheckError(err)
		jsOutPaths, err := cmd.Flags().GetStringArray("js_dir")
		helper.CheckError(err)
		projectConfigPath, err := cmd.Flags().GetString("file")
		helper.CheckError(err)

		if projectConfigPath == "" {
			cwd, err := os.Getwd()
			helper.CheckError(err)
			projectConfigPath, err = api.GetProjectConfigPathFromProjectPath(cwd)
			helper.CheckError(err)
			err = cmd.Flags().Set("file", projectConfigPath)
			helper.CheckError(err)
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

		// TODO: check for existence of npx, npm if web-client enabled
	},
	Run: func(cmd *cobra.Command, args []string) {
		err := runGenerateCmd(cmd)
		helper.CheckError(err)
		logger.Info("Files have been generated")
	},
}

func registerProtos(config *api.ProjectConfig) error {
	logger.Debugf("Registering protos for %s", config.ProjectConfigPath)
	projectRootPath := path.Dir(config.ProjectConfigPath)
	tmpDir, err := ioutil.TempDir("", "registerprotofile")
	helper.CheckError(err)
	defer func() {
		err := os.RemoveAll(tmpDir)
		helper.CheckError(err)
	}()

	for _, protoPath := range config.Import.Proto {
		params := append([]string{
			"--descriptor_set_out",
			path.Join(tmpDir, "data.pb"),
			"-I",
			path.Dir(protoPath),
		}, protoPath)

		logger.Debugf("protoc %s", strings.Join(params, " "))

		subProcess := exec.Command("protoc", params...)
		subProcess.Dir = projectRootPath
		subProcess.Stdout = os.Stdout
		subProcess.Stderr = os.Stderr

		err = subProcess.Run()
		helper.CheckErrorf(err, "%s has failed", subProcess.String())

		protoFile, err := ioutil.ReadFile(path.Join(tmpDir, "data.pb"))
		helper.CheckError(err)

		pbSet := new(descriptorpb.FileDescriptorSet)
		err = proto.Unmarshal(protoFile, pbSet)
		helper.CheckError(err)

		pb := pbSet.GetFile()[0]

		// And initialized the descriptor with it
		fd, err := protodesc.NewFile(pb, protoregistry.GlobalFiles)
		helper.CheckError(err)
		// and finally register it.
		err = protoregistry.GlobalFiles.RegisterFile(fd)
		helper.CheckError(err)
	}
	return nil
}

func clearProtoRegistry() {
	protoregistry.GlobalFiles = &protoregistry.Files{}
}

func runGenerateCmd(cmd *cobra.Command) error {
	projectConfigPath, err := cmd.Flags().GetString("file")
	helper.CheckError(err)
	typescript, err := cmd.Flags().GetBool("typescript")
	helper.CheckError(err)
	pythonOutPaths, err := cmd.Flags().GetStringArray("python_dir")
	helper.CheckError(err)
	jsOutPaths, err := cmd.Flags().GetStringArray("js_dir")
	helper.CheckError(err)

	config, err := api.CreateProjectConfigFromYaml(projectConfigPath)
	helper.CheckError(err)

	config.Typescript = typescript

	return generate(config, pythonOutPaths, jsOutPaths)
}

func generate(config *api.ProjectConfig, pythonOutPaths []string, jsOutPaths []string) error {
	logger.Infof("Generating project configuration for %s", config.ProjectConfigPath)
	typescript := config.Typescript

	config, err := api.CreateProjectConfigFromYaml(config.ProjectConfigPath)
	helper.CheckError(err)

	config.Typescript = typescript

	if err := registerProtos(config); err != nil {
		return err
	}
	defer clearProtoRegistry()

	config = updateConfigWithMessage(config)

	err = generatePython(config, pythonOutPaths)
	helper.CheckError(err)

	if len(jsOutPaths) > 0 {
		err := generateWebClient(config, jsOutPaths)
		helper.CheckError(err)
	}
	return nil
}

func generatePython(config *api.ProjectConfig, outputPaths []string) error {
	logger.Infof("Generating python configuration for %s", config.ProjectConfigPath)

	err := compileProtosPy(config, outputPaths)
	helper.CheckErrorf(err, "Failed compiling protobufs")

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

func compileProtosPy(config *api.ProjectConfig, outputPyPaths []string) error {
	projectRootPath := path.Dir(config.ProjectConfigPath)

	var params []string

	for _, protoPathProjectRelative := range config.Import.Proto {
		protoDirProjectRelative := path.Dir(protoPathProjectRelative)
		params = append(params, "-I", protoDirProjectRelative)
	}

	for _, outputPyPath := range outputPyPaths {
		params = append(params, "--python_out", outputPyPath)
	}

	for _, protoPathProjectRelative := range config.Import.Proto {
		params = append(params, protoPathProjectRelative)
	}

	logger.Debugf("protoc %s", strings.Join(params, " "))

	subProcess := exec.Command("protoc", params...)
	subProcess.Dir = projectRootPath
	subProcess.Stdout = os.Stderr
	subProcess.Stderr = os.Stderr

	if err := subProcess.Run(); err != nil {
		return fmt.Errorf("failed compiling protobuf %v", err)
	}

	return nil
}

func generateWebClient(config *api.ProjectConfig, jsOutPaths []string) error {
	projectRootPath := path.Dir(config.ProjectConfigPath)
	for _, jsOutPath := range jsOutPaths {
		logger.Infof("Generating %s/%s", projectRootPath, jsOutPath)

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
	projectRootPath := path.Dir(config.ProjectConfigPath)
	params := append([]string{
		"-I",
		".",
		fmt.Sprintf("--js_out=import_style=commonjs,binary:%s/src", jsOutPath),
	}, config.Import.Proto...)

	if config.Typescript {
		params = append(
			params,
			fmt.Sprintf("--plugin=protoc-gen-ts=%s/node_modules/.bin/protoc-gen-ts", jsOutPath),
			fmt.Sprintf("--ts_out=service=grpc-web:%s/src", jsOutPath),
		)
	}

	subProcess := exec.Command("protoc", params...)
	subProcess.Dir = projectRootPath
	subProcess.Stdout = os.Stderr
	subProcess.Stderr = os.Stderr

	if err := subProcess.Run(); err != nil {
		return fmt.Errorf("failed compiling protobufs %v", err)
	}

	return nil
}

func generateCogSettings(config *api.ProjectConfig, jsOutPath string) error {
	settingsFilenameTmpl := fmt.Sprintf("%s.tmpl", api.SettingsFilenameJs)
	tsSettingsTemplatePath := path.Join("/templates", settingsFilenameTmpl)
	webClientSrcPath := path.Join(jsOutPath, "src")
	cogSettingsPath := path.Join(webClientSrcPath, api.SettingsFilenameJs)
	shellCommand := "/bin/sh"

	if runtime.GOOS == "windows" {
		shellCommand = "cmd"
	}

	err := templates.GenerateFromTemplate(tsSettingsTemplatePath, config, cogSettingsPath)
	helper.CheckErrorf(err, "error generating %s", api.SettingsFilenameJs)

	tscCompileCmd := fmt.Sprintf(
		`
		set -xe
		npx tsc --declaration --declarationMap --outDir src src/%s
		exit
	`,
		api.SettingsFilenameJs,
	)

	subProcess := exec.Command(shellCommand)
	subProcess.Dir = jsOutPath
	subProcess.Stdout = os.Stdout
	subProcess.Stderr = os.Stderr

	stdin, err := subProcess.StdinPipe()
	if err != nil {
		logger.Fatalf("Unable to open pipe to subprocess stdin: %v", err)
	}
	defer func() {
		if err := stdin.Close(); err != nil {
			logger.Fatalf("Error when creating react app: %v", err)
		}
	}()

	if err := subProcess.Start(); err != nil {
		logger.Fatalf("Error in init while creating react app %v", err)
	}

	if _, err = fmt.Fprintln(stdin, tscCompileCmd); err != nil {
		return err
	}

	if err := subProcess.Wait(); err != nil {
		return err
	}

	//Clean up typescript cogsettings file
	typescriptFileToRemove := fmt.Sprintf("%s/src/%s", jsOutPath, api.SettingsFilenameJs)
	err = os.Remove(typescriptFileToRemove)
	if err != nil {
		helper.CheckErrorf(err, "Could not clean up %s/src/%s", jsOutPath, api.SettingsFilenameJs)
	}

	return nil
}

func updateConfigWithMessage(config *api.ProjectConfig) *api.ProjectConfig {
	for k, actorClass := range config.ActorClasses {
		if actorClass.ConfigType != "" {
			config.ActorClasses[k].ConfigType = lookupMessageType(actorClass.ConfigType)
		}

		config.ActorClasses[k].Action.Space = lookupMessageType(actorClass.Action.Space)
		config.ActorClasses[k].Observation.Space = lookupMessageType(actorClass.Observation.Space)
		if actorClass.Observation.Delta != "" {
			config.ActorClasses[k].Observation.Delta = lookupMessageType(actorClass.Observation.Delta)
		}
	}

	if config.Environment != nil && config.Environment.ConfigType != "" {
		config.Environment.ConfigType = lookupMessageType(config.Environment.ConfigType)
	}

	if config.Trial != nil && config.Trial.ConfigType != "" {
		config.Trial.ConfigType = lookupMessageType(config.Trial.ConfigType)
	}

	return config
}

func lookupMessageType(name string) string {
	s := strings.Split(name, ".")
	protoFile := findFileContainingSymbol(name)

	return api.ProtoAliasFromProtoPath(protoFile) + "." + s[len(s)-1]
}

func findFileContainingSymbol(name string) string {
	tmp := protoreflect.FullName(name)
	logger.Debugf("Finding proto file containing %s", tmp)

	desc, err := protoregistry.GlobalFiles.FindDescriptorByName(tmp)
	helper.CheckErrorf(err, "Failed to lookup %s", name)

	return desc.ParentFile().Path()
}

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.Flags().StringP("file", "f", "", "path to project config cogment.yaml")
	generateCmd.Flags().StringArrayP("js_dir", "j", []string{}, "python output directories (all directories must be valid npm projects, requires a node.js distribution on $PATH)")
	generateCmd.Flags().BoolP("typescript", "t", false, "project uses typescript")
	generateCmd.Flags().StringArrayP("python_dir", "p", []string{}, "python output directories")
}
