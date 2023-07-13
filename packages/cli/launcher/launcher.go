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
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

var ErrScriptCompleted = fmt.Errorf("script completed")

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

func launchAsyncProcess(ctx context.Context, eGroup *errgroup.Group, proc launchProcess) {

	// The call to eGroup.Go() is here (as opposed to the call site) because of the
	// closing by reference behavior of inline functions
	eGroup.Go(func() error {
		for cmdIndex, cmdArgs := range proc.Commands {
			cmdDesc := fmt.Sprintf("%s:(%d/%d)", proc.Name, cmdIndex+1, len(proc.Commands))

			err := executeCommand(ctx, cmdDesc, proc.Folder, proc.Environment, cmdArgs, proc.Quiet)
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
			"cmd": proc.Name,
		}).Trace("Script completed")

		return ErrScriptCompleted
	})
}

// The function will only return once all child processes have terminated
// On a OS signal, it will terminate all child processes, and block until all of them have completed.
func LaunchFromFile(args []string, launcherQuietLevel int) error {
	minimalOutput := (launcherQuietLevel > 2)
	err := configureLog(minimalOutput)
	if err != nil {
		return err
	}

	// Most simplistic way to add a "quiet" mode
	if launcherQuietLevel <= 0 {
		log.SetLevel(logrus.TraceLevel)
	} else if launcherQuietLevel == 1 {
		log.SetLevel(logrus.DebugLevel)
	} else if launcherQuietLevel >= 2 {
		log.SetLevel(logrus.InfoLevel)
	}

	if len(args) < 1 {
		return fmt.Errorf("launch file (yaml) not provided")
	}

	filename := args[0]
	rootPath := filepath.Dir(filename)

	processes, err := parseFile(filename, rootPath, args[1:])
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

	for _, proc := range processes {
		launchAsyncProcess(ctx, eGroup, proc)
	}

	return eGroup.Wait()
}
