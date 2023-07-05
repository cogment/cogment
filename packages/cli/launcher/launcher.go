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
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"text/template"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v2"
)

var ErrScriptCompleted = fmt.Errorf("script completed")

type ScriptDef struct {
	Commands    [][]string
	Environment yaml.MapSlice
	Dir         string // Deprecated
	Folder      string
	Quiet       bool
}

type GlobalDef struct {
	Environment yaml.MapSlice
	Folder      string
}

// The expected top level structure of the yaml file
type scriptFile struct {
	Global  GlobalDef
	Scripts map[string]ScriptDef
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

// Remove trailing empty values
func trimTrail(src []string) []string {
	lastIndex := len(src) - 1
	for lastIndex >= 0 {
		if len(src[lastIndex]) == 0 {
			lastIndex--
		} else {
			break
		}
	}

	return src[:lastIndex+1]
}

// Async utility function used to annotate and stream output from a child process
func streamStdOut(src *io.PipeReader, cmdName string, wg *sync.WaitGroup) {
	scanner := bufio.NewScanner(src)
	for scanner.Scan() {
		log.WithFields(logrus.Fields{
			"cmd": cmdName,
		}).Info(scanner.Text())
	}
	wg.Done()
}
func streamStdErr(src *io.PipeReader, cmdName string, wg *sync.WaitGroup) {
	scanner := bufio.NewScanner(src)
	for scanner.Scan() {
		log.WithFields(logrus.Fields{
			"cmd": cmdName,
		}).Warn(scanner.Text())
	}
	wg.Done()
}

func loadYaml(fileName string) (*scriptFile, error) {
	var result scriptFile

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

func parse(text string, dictionary map[string]string) (string, error) {
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

func executeCommand(ctx context.Context, cmdDesc string,
	folder string, env []string, cmdArgs []string, quiet bool) error {

	if len(cmdArgs) < 1 || len(cmdArgs[0]) == 0 {
		log.WithFields(logrus.Fields{"cmd": cmdDesc}).Trace("Empty command ignored")
		return nil
	}
	cmd := cmdArgs[0]
	args := trimTrail(cmdArgs[1:])

	action := "Launch"
	if quiet {
		action += " quiet"
	}
	log.WithFields(logrus.Fields{
		"cmd":  cmdDesc,
		"what": fmt.Sprintf("%v", cmdArgs[:len(args)+1]),
	}).Trace(action)

	cmdCtx := exec.CommandContext(ctx, cmd, args...)
	cmdCtx.Dir = folder
	cmdCtx.Env = env

	logWg := new(sync.WaitGroup)

	errReader, errWriter := io.Pipe()
	outReader, outWriter := io.Pipe()

	if !quiet {
		// Stream stderr/stdout
		cmdCtx.Stderr = errWriter
		cmdCtx.Stdout = outWriter

		logWg.Add(2)
		go streamStdErr(errReader, cmdDesc, logWg)
		go streamStdOut(outReader, cmdDesc, logWg)
	}

	err := cmdCtx.Start()

	if err != nil {
		errWriter.Close()
		outWriter.Close()
		return err
	}

	err = cmdCtx.Wait()

	// Wait for the logs to be done streaming. Otherwise, the Failure/Completion
	// log entry can be printed before the last logs of the command
	errWriter.Close()
	outWriter.Close()
	logWg.Wait()

	return err
}

// Launch asynchronously
func launchScript(ctx context.Context, eGroup *errgroup.Group, scriptName string,
	script ScriptDef, baseDict map[string]string, baseEnv []string) {

	// The call to eGroup.Go() is here (as opposed to the call site) because of the
	// closing by reference behavior of inline functions
	eGroup.Go(func() error {
		scriptDict := copyMap(baseDict)
		scriptEnv := copySlice(baseEnv)

		for _, item := range script.Environment {
			name := fmt.Sprintf("%v", item.Key)
			value := fmt.Sprintf("%v", item.Value)

			parsedValue, err := parse(value, scriptDict)
			if err != nil {
				log.WithFields(logrus.Fields{
					"cmd":   scriptName,
					"error": err,
				}).Debug("Environment variable substitution failed")
				return err
			}
			scriptDict[name] = parsedValue
			scriptEnv = append(scriptEnv, fmt.Sprintf("%v=%v", name, parsedValue))
		}

		for cmdIndex, cmdArgs := range script.Commands {

			var parsedCmdArgs = make([]string, 0, len(cmdArgs))
			for _, arg := range cmdArgs {
				parsedArg, err := parse(arg, scriptDict)
				if err != nil {
					log.WithFields(logrus.Fields{
						"cmd":   scriptName,
						"error": err,
					}).Debug("Command variable substitution failed")
					return err
				}
				parsedCmdArgs = append(parsedCmdArgs, parsedArg)
			}

			cmdDesc := fmt.Sprintf("%s:(%d/%d)", scriptName, cmdIndex+1, len(script.Commands))

			err := executeCommand(ctx, cmdDesc, script.Folder, scriptEnv, parsedCmdArgs, script.Quiet)
			if err != nil {
				// TODO: Find a better way than this
				if err.Error() == "signal: killed" {
					log.WithFields(logrus.Fields{
						"cmd": cmdDesc,
					}).Debug("Script command killed")
				} else {
					log.WithFields(logrus.Fields{
						"cmd":   cmdDesc,
						"error": err,
					}).Debug("Script command failed")
				}
				return err
			}

			log.WithFields(logrus.Fields{
				"cmd": cmdDesc,
			}).Trace("Script command completed")
		}

		log.WithFields(logrus.Fields{
			"cmd": scriptName,
		}).Trace("Script completed")

		return ErrScriptCompleted
	})
}

func LaunchAllScripts(ctx context.Context, eGroup *errgroup.Group,
	yamlDef *scriptFile, rootPath string, launchArgs []string) error {

	// Make base parsing dictionary and global environment
	dict := make(map[string]string)
	for _, str := range os.Environ() {
		index := strings.IndexRune(str, '=')
		name := str[:index]
		value := str[index+1:]
		dict[name] = value
	}

	argIndex := 1
	for ; argIndex < len(launchArgs); argIndex++ {
		argName := fmt.Sprintf("__%v", argIndex)
		dict[argName] = launchArgs[argIndex]
	}
	// There will always be 1-9, even if no arguments are passed
	for ; argIndex < 10; argIndex++ {
		argName := fmt.Sprintf("__%v", argIndex)
		dict[argName] = ""
	}

	env := os.Environ()
	for _, item := range yamlDef.Global.Environment {
		name := fmt.Sprintf("%v", item.Key)
		value := fmt.Sprintf("%v", item.Value)

		parsedValue, err := parse(value, dict)
		if err != nil {
			log.WithFields(logrus.Fields{
				"error": err,
			}).Debug("Global environment variable substitution failed")
			return err
		}
		dict[name] = parsedValue
		env = append(env, fmt.Sprintf("%v=%v", name, parsedValue))
	}

	var basePath string
	if filepath.IsAbs(yamlDef.Global.Folder) {
		basePath = yamlDef.Global.Folder
	} else {
		basePath = filepath.Join(rootPath, yamlDef.Global.Folder)
	}

	// Launch all child processes/scripts
	for name, script := range yamlDef.Scripts {
		if len(script.Folder) != 0 {
			if !filepath.IsAbs(script.Folder) {
				script.Folder = filepath.Join(basePath, script.Folder)
			}
		} else if len(script.Dir) != 0 {
			// "Dir" is deprecated
			if !filepath.IsAbs(script.Dir) {
				script.Folder = filepath.Join(basePath, script.Dir)
			} else {
				script.Folder = script.Dir
			}
		}
		launchScript(ctx, eGroup, name, script, dict, env)
	}

	return nil
}

// The function will only return once all child processes have terminated
// On a OS signal, it will terminate all child processes, and block until all of them have completed.
func LaunchFromFile(args []string, quietLevel int) error {
	minimalOutput := (quietLevel > 2)
	err := configureLog(minimalOutput)
	if err != nil {
		return err
	}

	// Most simplistic way to add a "quiet" mode
	if quietLevel <= 0 {
		log.SetLevel(logrus.TraceLevel)
	} else if quietLevel == 1 {
		log.SetLevel(logrus.DebugLevel)
	} else if quietLevel >= 2 {
		log.SetLevel(logrus.InfoLevel)
	}

	if len(args) < 1 {
		return fmt.Errorf("launch file (yaml) not provided")
	}

	fileName := args[0]
	rootPath := filepath.Dir(fileName)

	yamlDef, err := loadYaml(fileName)
	if err != nil {
		return err
	}

	rootCtx, cancelCtx := context.WithCancel(context.Background())
	eGroup, ctx := errgroup.WithContext(rootCtx)

	// Setup signal handling
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs

		log.WithFields(logrus.Fields{
			"signal": fmt.Sprintf("%v", sig),
		}).Trace("Stopping due to signal")

		cancelCtx()
	}()

	err = LaunchAllScripts(ctx, eGroup, yamlDef, rootPath, args)
	if err != nil {
		return err
	}

	return eGroup.Wait()
}
