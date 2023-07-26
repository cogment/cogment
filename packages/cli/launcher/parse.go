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

package launcher

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/hashicorp/terraform/dag"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"

	"github.com/cogment/cogment/utils"
)

// The empty string is not a legal script name, so we use as an internal node
const dagRootNode = ""
const dagRootIndex = -1

// Internal representation of definition file (pre-processed)
type launchDefinition struct {
	processes []launchProcess
}

type launchProcess struct {
	Name        string
	Folder      string
	Environment []string
	Quiet       bool
	ReadyRegex  *regexp.Regexp
	Dependency  []int

	Commands [][]string
	Ready    *utils.SingleEvent
}

// YAML definition file descriptor structures
type yamlFile struct {
	Global  yamlGlobal
	Scripts map[string]yamlScript
}

type yamlScript struct {
	Folder      string
	Environment yaml.MapSlice
	Dir         string // Deprecated
	Quiet       bool
	ReadyOutput string   `yaml:"ready_output"`
	DependsOn   []string `yaml:"depends_on"`
	Commands    [][]string
}

type yamlGlobal struct {
	Environment yaml.MapSlice
	Folder      string
}

func copyMap(src map[string]string) map[string]string {
	result := make(map[string]string)
	for name, value := range src {
		result[name] = value
	}

	return result
}

// Copy slice in a new array
func copySlice(src []string) []string {
	result := make([]string, len(src))
	copy(result, src)
	return result
}

func loadYaml(fileName string) (*yamlFile, error) {
	var result yamlFile

	yamlContent, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(yamlContent, &result)

	if err != nil {
		return nil, err
	}
	return &result, nil
}

func parseString(text string, dictionary map[string]string) (string, error) {
	parseTemplate, err := template.New("tmp").Parse(text)
	if err != nil {
		return "", err
	}

	var resultBytes bytes.Buffer
	err = parseTemplate.Execute(&resultBytes, dictionary)
	if err != nil {
		return "", err
	}

	return resultBytes.String(), nil
}

func parseScript(script *yamlScript, scriptName string, baseDict map[string]string, globalEnv []string, basePath string,
) (launchProcess, error) {
	proc := launchProcess{
		Name:  scriptName,
		Quiet: script.Quiet,
		Ready: utils.MakeSingleEvent(),
	}

	if len(script.Folder) != 0 {
		if !filepath.IsAbs(script.Folder) {
			proc.Folder = filepath.Join(basePath, script.Folder)
		} else {
			proc.Folder = script.Folder
		}
	} else if len(script.Dir) != 0 {
		log.Debug("The 'dir' script node is deprecated. Use 'folder' instead.")
		if !filepath.IsAbs(script.Dir) {
			proc.Folder = filepath.Join(basePath, script.Dir)
		} else {
			proc.Folder = script.Dir
		}
	} else {
		proc.Folder = basePath
	}

	scriptDict := copyMap(baseDict)
	proc.Environment = copySlice(globalEnv)

	for _, item := range script.Environment {
		name := fmt.Sprintf("%v", item.Key)
		var value string
		if item.Value != nil {
			value = fmt.Sprintf("%v", item.Value)
		}

		parsedValue, err := parseString(value, scriptDict)
		if err != nil {
			log.WithFields(logrus.Fields{
				"cmd":   scriptName,
				"error": err,
			}).Debug("Environment variable substitution failed")
			return launchProcess{}, err
		}
		scriptDict[name] = parsedValue
		proc.Environment = append(proc.Environment, fmt.Sprintf("%v=%v", name, parsedValue))
	}

	if len(script.ReadyOutput) > 0 {
		parsedRegex, err := parseString(script.ReadyOutput, scriptDict)
		if err != nil {
			log.WithFields(logrus.Fields{
				"cmd":   scriptName,
				"error": err,
			}).Debug("Regex string variable substitution failed")
			return launchProcess{}, err
		}
		compiledRegex, err := regexp.Compile(parsedRegex)
		if err != nil {
			log.WithFields(logrus.Fields{
				"cmd":   scriptName,
				"regex": parsedRegex,
				"error": err,
			}).Debug("Regex invalid")
			return launchProcess{}, err
		}
		proc.ReadyRegex = compiledRegex
	}

	proc.Commands = make([][]string, 0, len(script.Commands))
	for _, cmd := range script.Commands {
		var parsedCmd = make([]string, 0, len(cmd))
		for _, arg := range cmd {
			parsedArg, err := parseString(arg, scriptDict)
			if err != nil {
				log.WithFields(logrus.Fields{
					"cmd":   scriptName,
					"error": err,
				}).Debug("Command variable substitution failed")
				return launchProcess{}, err
			}
			parsedCmd = append(parsedCmd, parsedArg)
		}

		proc.Commands = append(proc.Commands, parsedCmd)
	}

	return proc, nil
}

func dependsOn(ag *dag.AcyclicGraph, dependent dag.Vertex, independent dag.Vertex) {
	// Not very efficient to duplicate vertex addition, but low numbers are expected
	ag.Add(dependent)
	ag.Add(independent)
	ag.Connect(dag.BasicEdge(independent, dependent))
}

func parseDependencies(def launchDefinition, file *yamlFile) error {
	nameIndex := make(map[string]int)
	for index, proc := range def.processes {
		nameIndex[proc.Name] = index
	}

	testDag := dag.AcyclicGraph{}
	for _, proc := range def.processes {
		dependsOn(&testDag, proc.Name, dagRootNode)

		script := file.Scripts[proc.Name]
		for _, name := range script.DependsOn {
			_, ok := nameIndex[name]
			if !ok {
				return fmt.Errorf("unknown dependency [%s] in script [%s]", name, proc.Name)
			}
			dependsOn(&testDag, proc.Name, name)
		}
	}

	err := testDag.Validate()
	if err != nil {
		return fmt.Errorf("invalid dependency - %w", err)
	}
	testDag.TransitiveReduction()

	nameIndex[dagRootNode] = dagRootIndex
	for index, proc := range def.processes {
		depEdges := testDag.EdgesTo(proc.Name)
		for _, edge := range depEdges {
			var depName string = edge.Source().(string)
			depIndex, ok := nameIndex[depName]
			if !ok {
				return fmt.Errorf("dependency DAG unexpected entry[%v]", depName)
			}
			if depIndex == dagRootIndex {
				continue
			}
			if index == depIndex {
				return fmt.Errorf("dependency DAG inconsistency [%v]", proc.Name)
			}
			// TODO: There has got to be a better way! E.g. using proc
			def.processes[index].Dependency = append(def.processes[index].Dependency, depIndex)
		}
	}

	return nil
}

func parseFile(filename string, cliArgs []string) (launchDefinition, error) {

	yamlDef, err := loadYaml(filename)
	if err != nil {
		return launchDefinition{}, err
	}
	nbScripts := len(yamlDef.Scripts)
	if nbScripts == 0 {
		return launchDefinition{}, fmt.Errorf("no script defined")
	}
	result := launchDefinition{
		processes: make([]launchProcess, 0, nbScripts),
	}

	baseDict := make(map[string]string)
	for _, str := range os.Environ() {
		index := strings.IndexRune(str, '=')
		name := str[:index]
		value := str[index+1:]
		baseDict[name] = value
	}

	argIndex := 0
	for ; argIndex < len(cliArgs); argIndex++ {
		argName := fmt.Sprintf("__%v", argIndex+1)
		baseDict[argName] = cliArgs[argIndex]
	}
	for ; argIndex < 9; argIndex++ { // 1 to 9 are always defined
		argName := fmt.Sprintf("__%v", argIndex+1)
		baseDict[argName] = ""
	}

	globalEnv := os.Environ()
	for _, item := range yamlDef.Global.Environment {
		name := fmt.Sprintf("%v", item.Key)
		var value string
		if item.Value != nil {
			value = fmt.Sprintf("%v", item.Value)
		}

		parsedValue, err := parseString(value, baseDict)
		if err != nil {
			log.WithFields(logrus.Fields{
				"error": err,
			}).Debug("Global environment variable substitution failed")
			return launchDefinition{}, err
		}
		baseDict[name] = parsedValue
		globalEnv = append(globalEnv, fmt.Sprintf("%v=%v", name, parsedValue))
	}

	var basePath string
	if filepath.IsAbs(yamlDef.Global.Folder) {
		basePath = yamlDef.Global.Folder
	} else {
		rootPath := filepath.Dir(filename)
		basePath = filepath.Join(rootPath, yamlDef.Global.Folder)
	}

	for scriptName, script := range yamlDef.Scripts {
		if len(scriptName) == 0 {
			return launchDefinition{}, fmt.Errorf("empty script name")
		}

		proc, err := parseScript(&script, scriptName, baseDict, globalEnv, basePath)
		if err != nil {
			return launchDefinition{}, err
		}

		result.processes = append(result.processes, proc)
	}

	err = parseDependencies(result, yamlDef)
	if err != nil {
		return launchDefinition{}, err
	}

	return result, nil
}
