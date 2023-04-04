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

//go:build cgo
// +build cgo

package wrapper

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStartAndShutdownOrchestrator(t *testing.T) {
	w, err := NewWrapper()
	assert.NoError(t, err)
	assert.NotNil(t, w)

	// Test some options
	err = w.AddPretrialHooksEndpoint("grpc://my_endpoint:8000")
	assert.NoError(t, err)

	err = w.AddDirectoryServicesEndpoint("grpc://my_other_endpoint:8000")
	assert.NoError(t, err)

	err = w.SetLifecyclePort(9500)
	assert.NoError(t, err)

	err = w.Start()
	assert.NoError(t, err)

	go func() {
		time.Sleep(5 * time.Second)
		err := w.Shutdown()
		assert.NoError(t, err)
	}()
	err = w.Wait()
	assert.NoError(t, err)
}
