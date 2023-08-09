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
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestObservableOneObserverWait(t *testing.T) {
	observable := NewObservable()

	observer := observable.Subscribe()
	defer observable.Unsubscribe(observer)

	go func() {
		time.After(100 * time.Millisecond)
		observable.Emit()
	}()

	select {
	case <-observer.Receive():
		return
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout")
	}
}

func TestObservableMultipleObserversWait(t *testing.T) {
	observable := NewObservable()

	subscribedGroup := sync.WaitGroup{}
	endedGroup := sync.WaitGroup{}
	atomicCounter := atomic.Int32{}
	atomicCounter.Store(0)
	const nbObservers = 10
	const nbExpectedNotifications = 10

	for i := 0; i < nbObservers; i++ {
		i := i
		subscribedGroup.Add(1)
		endedGroup.Add(1)
		go func() {
			defer endedGroup.Done()

			observer := observable.Subscribe()
			defer observable.Unsubscribe(observer)
			subscribedGroup.Done()

			nbNotifications := 0
			for {
				select {
				case <-observer.Receive():
					atomicCounter.Add(1)
					nbNotifications++
					if nbNotifications >= nbExpectedNotifications {
						return
					}
				case <-time.After(1000 * time.Millisecond):
					t.Errorf("observer #%d timed out", i)
				}
			}
		}()
	}

	subscribedGroup.Wait()

	for i := 0; i < nbExpectedNotifications; i++ {
		observable.Emit()
		time.Sleep(50 * time.Millisecond)
	}

	endedGroup.Wait()

	assert.Equal(t, nbObservers*nbExpectedNotifications, int(atomicCounter.Load()))
}
