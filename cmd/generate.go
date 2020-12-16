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
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/spf13/cobra"
	"gitlab.com/cogment/cogment/api"
	"gitlab.com/cogment/cogment/templates"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"gopkg.in/yaml.v2"
)

const settingsFilename = "cog_settings"
const (
	python = iota
	javascript
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate settings and compile your proto files",
	PreRun: func(cmd *cobra.Command, args []string) {
		if !cmd.Flags().Changed("python_dir") && !cmd.Flags().Changed("js_dir") {
			log.Fatalln("no destination specified")
		}

		file, err := cmd.Flags().GetString("file")
		if err != nil {
			log.Fatalln(err)
		}

		if _, err := os.Stat(file); os.IsNotExist(err) {
			log.Fatalf("%s doesn't exist", file)
		}

	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := runGenerateCmd(cmd); err != nil {
			log.Fatalln(err)
		}

		fmt.Println("Files have been generated")
	},
}

func registerProtoFile(src string, filename string) error {
	log.Printf("registering %s", filename)

	// First, convert the .proto file to a file descriptor in a temp directory
	tmpDir, err := ioutil.TempDir("", "registerprotofile")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	tmpFile := path.Join(tmpDir, filename+".pb")

	os.MkdirAll(path.Dir(tmpFile), 0755)

	cmd := exec.Command("protoc",
		"--descriptor_set_out="+tmpFile,
		"-I"+src,
		path.Join(src, filename))

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		if Verbose {
			log.Println(cmd.String())
		}
		log.Printf("cmd.Run() failed with %s\n", err)
		return err
	}

	// Now load that temporary file as a descriptor protobuf
	protoFile, err := ioutil.ReadFile(tmpFile)
	if err != nil {
		return err
	}

	pbSet := new(descriptorpb.FileDescriptorSet)
	if err := proto.Unmarshal(protoFile, pbSet); err != nil {
		return err
	}

	pb := pbSet.GetFile()[0]

	// And initialized the descriptor with it
	fd, err := protodesc.NewFile(pb, protoregistry.GlobalFiles)
	if err != nil {
		return err
	}

	// and finally register it.
	return protoregistry.GlobalFiles.RegisterFile(fd)
}

func runGenerateCmd(cmd *cobra.Command) error {

	file, err := cmd.Flags().GetString("file")
	if err != nil {
		log.Fatalln(err)
	}

	//TODO: validate logic in config

	src := filepath.Dir(file)

	config := createProjectConfigFromYaml(file)
	for _, proto := range config.Import.Proto {
		err = registerProtoFile(src, proto)
		if err != nil {
			return err
		}
	}

	if cmd.Flags().Changed("python_dir") {
		dest, err := cmd.Flags().GetString("python_dir")
		if err != nil {
			return err
		}

		// We need to reload the config because it is being manipulated
		config = createProjectConfigFromYaml(file)
		if err := generatePythonSettings(config, src, dest); err != nil {
			return err
		}
	}

	if cmd.Flags().Changed("js_dir") {
		dest, err := cmd.Flags().GetString("js_dir")
		if err != nil {
			return err
		}

		// We need to reload the config because it is being manipulated
		config = createProjectConfigFromYaml(file)
		if err := generateJavascriptSettings(config, src, dest); err != nil {
			return err
		}
	}

	return nil
}

func generatePythonSettings(config *api.ProjectConfig, src, dir string) error {
	dest := path.Join(dir, settingsFilename+".py")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	var err error
	for k, proto := range config.Import.Proto {
		config.Import.Proto[k] = strings.TrimSuffix(proto, ".proto") + "_pb2"
		config.Import.Proto[k] = strings.ReplaceAll(config.Import.Proto[k], "/", ".")
		if Verbose {
			log.Println("Compiling " + proto)
		}

		if err = compileProto(src, dir, proto, python); err != nil {
			break
		}
	}

	if err != nil {
		return err
	}

	config = updateConfigWithMessage(config)

	err = templates.GenerateFromTemplate("/templates/cog_settings.py.tmpl", config, dest)

	return err
}

func generateJavascriptSettings(config *api.ProjectConfig, src, dir string) error {
	dest := path.Join(dir, settingsFilename+".js")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	var err error
	for k, proto := range config.Import.Proto {
		config.Import.Proto[k] = strings.TrimSuffix(proto, ".proto") + "_pb"
		if Verbose {
			log.Println("Compiling " + proto)
		}

		if err = compileProto(src, dir, proto, javascript); err != nil {
			break
		}
	}

	if err != nil {
		return err
	}

	config = updateConfigWithMessage(config)

	err = templates.GenerateFromTemplate("/templates/cog_settings.js.tmpl", config, dest)

	return err
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

func getProtoAlias(protoFile string) string {
	fname := strings.Split(protoFile, ".")[0]
	return strings.ReplaceAll(fname, "/", "_") + "_pb"
}

func createProjectConfigFromYaml(filename string) *api.ProjectConfig {

	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}

	config := api.ProjectConfig{}
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	for _, proto := range config.Import.Proto {
		config.Import.ProtoAlias = append(config.Import.ProtoAlias, getProtoAlias(proto))
	}

	//fmt.Println(helper.PrettyPrint(m))
	return &config
}

func lookupMessageType(name string) string {
	s := strings.Split(name, ".")
	protoFile := findFileContainingSymbol(name)

	return getProtoAlias(protoFile) + "." + s[len(s)-1]
}

func findFileContainingSymbol(name string) string {
	tmp := protoreflect.FullName(name)
	desc, err := protoregistry.GlobalFiles.FindDescriptorByName(tmp)

	if err != nil {
		log.Printf("Failed to lookup %s", name)
		log.Fatalf("%v", err)
	}

	return desc.ParentFile().Path()
}

func compileProto(src, dest, proto string, language int8) error {

	var params []string
	switch language {
	case python:
		_, err := exec.LookPath("protoc-gen-mypy")
		if err != nil {
			log.Printf("Warning: protoc-gen-mypy not found, IDE autocomplete support will be limited.")
		} else {
			params = append(params, "--mypy_out="+dest)
		}

		params = append(params, "-I"+src, "--python_out="+dest, path.Join(src, proto))
	case javascript:
		params = append(params, "-I"+src, "--js_out=import_style=commonjs,binary:"+dest, path.Join(src, proto))
	default:
		return fmt.Errorf("language %d is not supported", language)
	}

	cmd := exec.Command("protoc", params...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		if Verbose {
			log.Println(cmd.String())
		}
		log.Printf("cmd.Run() failed with %s\n", err)
		return err
	}

	return nil
}

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.Flags().StringP("file", "f", "cogment.yaml", "project configuration file")
	generateCmd.Flags().String("python_dir", "", "destination of python generated files")
	generateCmd.Flags().String("js_dir", "", "destination of javascript generated files")

}
