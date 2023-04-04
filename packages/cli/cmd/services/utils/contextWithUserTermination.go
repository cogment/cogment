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

package utils

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func ContextWithUserTermination(ctx context.Context) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	// using a buffered channel cf. https://link.medium.com/M8dPZv9Wuob
	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-interruptChan
		log.Debug("SIGTERM received")
		signal.Stop(interruptChan) // Stopping registration to this channel
		cancel()
	}()

	return ctx
}
