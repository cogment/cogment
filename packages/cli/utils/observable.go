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
	"sync"
)

type Observer struct {
	channel chan struct{}
}

func newObserver() *Observer {
	return &Observer{
		channel: make(chan struct{}),
	}
}

func (observer *Observer) Receive() <-chan struct{} {
	return observer.channel
}

func (observer *Observer) emit() {
	go func() {
		observer.channel <- struct{}{}
	}()
}

// Observable represents a structure that can be observed by multiple observers (in multiple goroutines)
type Observable struct {
	lock      sync.RWMutex
	observers map[*Observer]struct{}
}

func NewObservable() *Observable {
	return &Observable{
		lock:      sync.RWMutex{},
		observers: make(map[*Observer]struct{}),
	}
}

func (observable *Observable) Subscribe() *Observer {
	observable.lock.Lock()
	defer observable.lock.Unlock()
	observer := newObserver()
	observable.observers[observer] = struct{}{}
	return observer
}

func (observable *Observable) Unsubscribe(observer *Observer) {
	observable.lock.Lock()
	defer observable.lock.Unlock()
	delete(observable.observers, observer)
}

func (observable *Observable) Emit() {
	observable.lock.RLock()
	defer observable.lock.RUnlock()
	for observer := range observable.observers {
		observer.emit()
	}
}
