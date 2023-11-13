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

import "sync"

// Broadcasting single event
type SingleEvent struct {
	eventCond sync.Cond
	eventVar  bool
	disabled  bool
	callback  func()
}

func MakeSingleEvent() *SingleEvent {
	lock := sync.Mutex{}
	event := SingleEvent{eventCond: *sync.NewCond(&lock)}
	return &event
}

func (se *SingleEvent) IsSet() bool {
	// Ideally this variable would be atomic, but it is overkill here
	return se.eventVar
}

func (se *SingleEvent) Disable() {
	se.eventCond.L.Lock()
	se.disabled = true
	se.eventCond.L.Unlock()
	se.eventCond.Broadcast()
}

func (se *SingleEvent) Set() {
	se.eventCond.L.Lock()
	se.eventVar = true

	callOnSet := func() {}
	if se.callback != nil {
		callOnSet = se.callback
		se.callback = nil
	}
	se.eventCond.L.Unlock()

	callOnSet()
	se.eventCond.Broadcast()
}

func (se *SingleEvent) Wait() bool {
	se.eventCond.L.Lock()
	defer se.eventCond.L.Unlock()

	if !se.eventVar && !se.disabled {
		se.eventCond.Wait()
	}

	return se.eventVar
}

func (se *SingleEvent) SetCallback(callback func()) {
	se.eventCond.L.Lock()

	if se.eventVar {
		se.eventCond.L.Unlock()
		callback()
	} else {
		se.callback = callback
		se.eventCond.L.Unlock()
	}
}
