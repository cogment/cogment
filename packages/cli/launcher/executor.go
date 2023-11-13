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
	"os/exec"
	"regexp"
	"sync"

	"github.com/cogment/cogment/utils"
)

var errCtxDone = fmt.Errorf("stopped by user")

type executor struct {
	Ctx           context.Context
	Folder        string
	Environment   []string
	OutputEnabled bool
	OutputRegex   *regexp.Regexp
	OutputMatched *utils.SingleEvent
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

func (exe *executor) streamOut(out func(args ...interface{}), src *io.PipeReader, wg *sync.WaitGroup) {
	defer wg.Done()

	scanner := bufio.NewScanner(src)
	for scanner.Scan() {
		text := scanner.Text()

		if exe.OutputEnabled {
			out(text)
		}

		if !exe.OutputMatched.IsSet() && exe.OutputRegex.MatchString(text) {
			exe.OutputMatched.Set()
		}
	}
}

func (exe *executor) execute(cmdDesc string, cmdArgs []string) error {
	logger := log.WithField("cmd", cmdDesc)

	if exe.OutputRegex == nil {
		exe.OutputMatched.Set()
	}

	if len(cmdArgs) < 1 || len(cmdArgs[0]) == 0 {
		logger.Trace("Empty command ignored")
		return nil
	}

	cmd := cmdArgs[0]
	args := trimTrail(cmdArgs[1:])

	cmdCtx := exec.CommandContext(exe.Ctx, cmd, args...)
	cmdCtx.Dir = exe.Folder
	cmdCtx.Env = exe.Environment

	if exe.OutputEnabled || !exe.OutputMatched.IsSet() {
		errReader, errWriter := io.Pipe()
		outReader, outWriter := io.Pipe()
		cmdCtx.Stderr = errWriter
		cmdCtx.Stdout = outWriter

		logWg := new(sync.WaitGroup)
		logWg.Add(2)

		go exe.streamOut(logger.Info, outReader, logWg)
		go exe.streamOut(logger.Warn, errReader, logWg)
		defer func() {
			errWriter.Close()
			outWriter.Close()
			logWg.Wait()
		}()
	}

	cmdLine := cmd
	for _, arg := range args {
		cmdLine += " " + arg
	}
	if exe.OutputEnabled {
		logger.Trace("Launch [", cmdLine, "]")
	} else {
		logger.Trace("Launch quiet [", cmdLine, "]")
	}

	err := cmdCtx.Start()
	if err != nil {
		select {
		case <-exe.Ctx.Done():
			logger.Trace("Stopped")
			return errCtxDone
		default:
			logger.Debug("Failed [", err, "]")
			return err
		}
	}

	err = cmdCtx.Wait()
	logger.Trace("Time: User [", cmdCtx.ProcessState.UserTime(), "] System [", cmdCtx.ProcessState.SystemTime(), "]")
	if err != nil {
		select {
		case <-exe.Ctx.Done():
			logger.Trace("Stopped")
			return errCtxDone
		default:
			logger.Debug("Failed [", err, "]")
			return err
		}
	}
	logger.Trace("Completed. Exit code [", cmdCtx.ProcessState.ExitCode(), "]")

	return nil
}
