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
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func runProcess(ctx context.Context, def launchDefinition, index int) {
	proc := def.processes[index]
	logger := log.WithField("cmd", proc.Name)
	proc.Ready.SetCallback(func() {
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
		return
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
			if err == errCtxDone {
				logger.Trace("Stopped")
			} else {
				logger.Debug("Failed")
			}
			return
		}
	}

	logger.Trace("Completed")
}

// This function will only return once all child processes have terminated.
func Launch(args []string, launcherQuietLevel int) error {
	configureLog(launcherQuietLevel)

	if len(args) == 0 {
		return fmt.Errorf("required launch definition file missing")
	}
	filename := args[0]
	launchArgs := args[1:]

	definitionFile, err := parseFile(filename, launchArgs)
	if err != nil {
		return err
	}

	ctx, cancelCtx := context.WithCancel(context.Background())
	processReturns := make(chan int, len(definitionFile.processes))
	returnCount := new(int)

	// Setup signal handling
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-processReturns:
			*returnCount++

		case sig := <-sigs:
			log.Debug("User signal [", sig, "]. Stopping...")
		}
		cancelCtx()
	}()

	for index := range definitionFile.processes {
		procIndex := index
		go func() {
			runProcess(ctx, definitionFile, procIndex)
			processReturns <- procIndex
		}()
	}

	<-ctx.Done()

	// Give a chance to all processes to report their exit
	for {
		<-processReturns
		*returnCount++
		if *returnCount >= len(definitionFile.processes) {
			break
		}
	}

	return nil
}
