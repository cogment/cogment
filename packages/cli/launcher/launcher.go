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
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"
)

var errScriptCompleted = fmt.Errorf("script completed")
var errScriptCancelled = fmt.Errorf("script cancelled")

func runProcess(ctx context.Context, def launchDefinition, index int) error {
	proc := def.processes[index]
	logger := log.WithField("cmd", proc.Name)
	proc.Ready.ReadyFunc(func() {
		logger.Trace("Ready")
	})

	// Wait for dependencies
	for _, dependenceIndex := range proc.Dependency {
		def.processes[dependenceIndex].Ready.Wait()
	}
	defer proc.Ready.Disable()

	select {
	case <-ctx.Done():
		logger.Debug("Cancelled")
		return errScriptCancelled
	default:
	}

	exe := executor{
		Ctx:           ctx,
		Folder:        proc.Folder,
		Environment:   proc.Environment,
		OutputEnabled: !proc.Quiet,
		OutputRegex:   proc.ReadyRegex,
		OutputMatched: proc.Ready,
	}

	for cmdIndex, cmdArgs := range proc.Commands {
		cmdDesc := fmt.Sprintf("%s:(%d/%d)", proc.Name, cmdIndex+1, len(proc.Commands))

		err := exe.execute(cmdDesc, cmdArgs)
		if err != nil {
			return err
		}
	}

	logger.Trace("Completed")

	return errScriptCompleted
}

// On a OS signal, it will terminate all child processes, and block until all of them have completed.
func launchFile(args []string) (*errgroup.Group, error) {
	rootCtx, cancelCtx := context.WithCancel(context.Background())
	eGroup, ctx := errgroup.WithContext(rootCtx)

	// Setup signal handling
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.WithField("signal", fmt.Sprintf("%v", sig)).Debug("Stopping")
		cancelCtx()
	}()

	if len(args) == 0 {
		return nil, fmt.Errorf("required launch definition file missing")
	}
	filename := args[0]
	launchArgs := args[1:]

	definitionFile, err := parseFile(filename, launchArgs)
	if err != nil {
		return nil, err
	}

	for index := range definitionFile.processes {
		procIndex := index
		eGroup.Go(func() error {
			return runProcess(ctx, definitionFile, procIndex)
		})
	}

	return eGroup, nil
}

// This function will only return once all child processes have terminated.
func Launch(args []string, launcherQuietLevel int) error {
	configureLog(launcherQuietLevel)

	eGroup, err := launchFile(args)
	if err != nil {
		return err
	}

	err = eGroup.Wait()
	if !errors.Is(err, errScriptCompleted) &&
		!errors.Is(err, errScriptCancelled) {
		return err
	}

	return nil
}
