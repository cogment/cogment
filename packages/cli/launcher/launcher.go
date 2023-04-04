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

// Represents a single script to launch
type Script struct {
	Commands    [][]string
	Environment map[string]string
	Dir         string
	Quiet       bool
}

// The expected structure of top of the yaml file
type scriptFile struct {
	Scripts map[string]Script
}

// Loads a series of scripts from a yaml file
func LoadScriptsFromYaml(fileName string) (map[string]Script, error) {
	var result scriptFile

	yamlContent, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(yamlContent, &result)

	if err != nil {
		return nil, err
	}
	return result.Scripts, nil
}

// Generates the correct script(id/total) string for a given command within a script
func formatCommandName(scriptName string, script Script, cmdID int) string {
	return fmt.Sprintf("%s:(%d/%d)", scriptName, cmdID+1, len(script.Commands))
}

// Utility function used to annotate and stream output from a child process
func streamLog(src *io.PipeReader, dst *os.File, cmdName string, index int, wg *sync.WaitGroup) {
	scanner := bufio.NewScanner(src)
	for scanner.Scan() {
		log.WithFields(logrus.Fields{
			"cmd": cmdName,
		}).Info(scanner.Text())
	}
	wg.Done()
}

func stringsToMap(src []string) map[string]string {
	result := make(map[string]string)
	for _, str := range src {
		splits := strings.Split(str, "=")
		key := splits[0]
		val := strings.Join(splits[1:], "=")
		result[key] = val
	}
	return result
}

func mapToStrings(src map[string]string) []string {
	result := []string{}
	for k, v := range src {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}
	return result
}

func substitute(text string, data map[string]string) (string, error) {
	tmpl, err := template.New("tmp").Parse(text)
	if err != nil {
		return "", err
	}

	var resultBytes bytes.Buffer
	if err := tmpl.Execute(&resultBytes, data); err != nil {
		return "", err
	}

	return resultBytes.String(), nil
}

// To be called from launchScript() exclusively
func executeSingleCommand(ctx context.Context, name string,
	env map[string]string, cmdArgs []string, script Script, index int) error {

	// Run the arguments through the template parser
	var realArgs = []string{}
	for _, arg := range cmdArgs {
		str, err := substitute(arg, env)
		if err != nil {
			return err
		}
		realArgs = append(realArgs, str)
	}

	log.WithFields(logrus.Fields{
		"cmd":  name,
		"what": fmt.Sprintf("%v", realArgs),
	}).Info("Launch")

	cmd := exec.CommandContext(ctx, realArgs[0], realArgs[1:]...)
	cmd.Env = mapToStrings(env)
	cmd.Dir = script.Dir

	logWg := new(sync.WaitGroup)

	errReader, errWriter := io.Pipe()
	outReader, outWriter := io.Pipe()

	if !script.Quiet {
		// Stream stderr/stdout
		cmd.Stderr = errWriter
		cmd.Stdout = outWriter

		logWg.Add(2)
		go streamLog(errReader, os.Stderr, name, index, logWg)
		go streamLog(outReader, os.Stdout, name, index, logWg)
	}

	err := cmd.Start()

	if err != nil {
		errWriter.Close()
		outWriter.Close()
		return err
	}

	err = cmd.Wait()

	// Wait for the logs to be done streaming. Otherwise, the Failure/Completion
	// log entry can be printed before the last logs of the command
	errWriter.Close()
	outWriter.Close()
	logWg.Wait()

	if err != nil {
		log.WithFields(logrus.Fields{
			"cmd":    name,
			"status": err,
		}).Info("Script command failed")
	} else {
		log.WithFields(logrus.Fields{
			"cmd": name,
		}).Info("Script command completed")
	}
	return err
}

// To be called from LaunchScripts() exclusively
func launchScript(ctx context.Context, g *errgroup.Group, scriptName string, script Script, index int) {

	// The call to g.Go() is here (as opposed to the callsite) because of the
	// closing by reference behavior of inline functions
	g.Go(func() error {
		// Set up environment variables
		env := stringsToMap(os.Environ())
		for k, v := range script.Environment {
			str, err := substitute(v, env)
			if err != nil {
				return err
			}
			env[k] = str
		}

		for i, cmdArgs := range script.Commands {
			err := executeSingleCommand(ctx, formatCommandName(scriptName, script, i), env, cmdArgs, script, index)
			if err != nil {
				log.WithFields(logrus.Fields{
					"script": scriptName,
					"error":  fmt.Sprintf("%v", err),
				}).Info("Script failed")
				return err
			}
		}

		log.WithFields(logrus.Fields{
			"script": scriptName,
		}).Info("Script completed")

		return ErrScriptCompleted
	})
}

// Launches a concurrent set of named scripts.
//   - The function will only return once all child processes have terminated
//   - On a OS signal, it will terminate all child processes, and block until all
//     of them have completed.
func LaunchScripts(scripts map[string]Script, rootPath string) error {
	rootCtx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(rootCtx)

	// Setup signal handling
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs

		log.WithFields(logrus.Fields{
			"signal": fmt.Sprintf("%v", sig),
		}).Info("Stopping due to signal")

		cancel()
	}()

	// Launch all child processes
	i := 0
	for name, cfg := range scripts {
		if !filepath.IsAbs(cfg.Dir) {
			cfg.Dir = filepath.Join(rootPath, cfg.Dir)
		}
		launchScript(ctx, g, name, cfg, i)
		i++
	}

	return g.Wait()
}

// Launches a concurrent set of named scripts as sepcified in a yaml file.
// See LaunchScripts() for details
func LaunchFromFile(fileName string) error {
	if err := configureLog(); err != nil {
		return err
	}
	rootPath := filepath.Dir(fileName)

	scriptsToLaunch, err := LoadScriptsFromYaml(fileName)
	if err != nil {
		return err
	}

	return LaunchScripts(scriptsToLaunch, rootPath)
}
