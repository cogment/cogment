// Copyright 2022 AI Redefined Inc. <dev+cogment@ai-r.com>
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

/*
	extern void status_listener_callback(void* ctx, int status);
*/
import "C"
import (
	"sync"
	"unsafe"

	"github.com/cogment/cogment/services/utils"
)

// Pointer to a C function able to be used as a callback for the orchestrator's status listener
//
// This function takes a "context" handle that is used to route the call to the right underlying go status listener.
// The management of the handles and the routing is done by `statusListenerRegistry`
var cStatusListenerCallback unsafe.Pointer = C.status_listener_callback

//export status_listener_callback
func status_listener_callback(ctx unsafe.Pointer, status C.int) {
	fn := statusListenerRegistrySingleton.lookup(uintptr(ctx))
	switch status {
	case 1:
		fn(utils.StatusInitializating)
	case 2:
		fn(utils.StatusReady)
	case 3:
		fn(utils.StatusTerminated)
	default:
		fn(utils.StatusUnknown)
	}
}

type statusListenerRegistry struct {
	mutex     sync.Mutex
	index     uintptr
	callbacks map[uintptr]utils.StatusListener
}

func (r *statusListenerRegistry) register(statusListener utils.StatusListener) uintptr {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.index++
	for r.callbacks[r.index] != nil {
		r.index++
	}
	r.callbacks[r.index] = statusListener
	return r.index
}

func (r *statusListenerRegistry) lookup(handle uintptr) utils.StatusListener {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.callbacks[handle]
}

func (r *statusListenerRegistry) unregister(handle uintptr) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.callbacks, handle)
}

var statusListenerRegistrySingleton = statusListenerRegistry{
	index:     0,
	callbacks: make(map[uintptr]utils.StatusListener),
}
