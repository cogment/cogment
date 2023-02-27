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

package utils

import (
	"context"
	"sync"
)

type ObservableListItem interface{}
type ObservableListObserver chan ObservableListItem

type ObservableList interface {
	Len() int
	HasEnded() bool
	Item(index int) (ObservableListItem, bool)
	Append(item ObservableListItem, last bool)
	Observe(ctx context.Context, from int, out chan<- ObservableListItem) error
}

type observer chan bool

type observableList struct {
	itemsLock     sync.RWMutex
	items         []ObservableListItem
	ended         bool
	observersLock sync.RWMutex
	observers     map[*observer]struct{}
}

func CreateObservableList() ObservableList {
	return &observableList{
		items:     make([]ObservableListItem, 0),
		ended:     false,
		observers: make(map[*observer]struct{}),
	}
}

func (l *observableList) Len() int {
	l.itemsLock.RLock()
	defer l.itemsLock.RUnlock()
	return len(l.items)
}

func (l *observableList) HasEnded() bool {
	l.itemsLock.RLock()
	defer l.itemsLock.RUnlock()
	return l.ended
}

func (l *observableList) Item(index int) (ObservableListItem, bool) {
	l.itemsLock.RLock()
	defer l.itemsLock.RUnlock()
	if index < 0 || index > l.Len() {
		return nil, false
	}
	return l.items[index], true
}

func (l *observableList) registerObserver() *observer {
	l.observersLock.Lock()
	defer l.observersLock.Unlock()
	observer := make(observer)
	l.observers[&observer] = struct{}{}
	return &observer
}

func (l *observableList) unregisterObserver(o *observer) {
	l.observersLock.Lock()
	defer l.observersLock.Unlock()
	delete(l.observers, o)
}

func (l *observableList) Append(item ObservableListItem, lastItem bool) {
	l.itemsLock.Lock()
	l.items = append(l.items, item)
	l.ended = lastItem
	l.itemsLock.Unlock()

	// This locks the iteration thus making sure the goroutine is called with an observer that exist
	l.observersLock.RLock()
	defer l.observersLock.RUnlock()
	for o := range l.observers {
		o := o
		go func() {
			*o <- lastItem
		}()
	}
}

func (l *observableList) Observe(ctx context.Context, from int, out chan<- ObservableListItem) error {
	if l.HasEnded() {
		// The list is ended, no risk from concurrent writes
		currentItems := l.items[from:]
		for _, item := range currentItems {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case out <- item:
			}
		}
	} else {
		observer := l.registerObserver()
		defer l.unregisterObserver(observer)
		for {
			// Read everything up to the current count
			l.itemsLock.RLock()
			end := len(l.items)
			ended := l.ended
			currentItems := []ObservableListItem{}
			if from <= end {
				currentItems = l.items[from:end]
				from = end
			}
			l.itemsLock.RUnlock()
			for _, item := range currentItems {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case out <- item:
				}
			}

			if ended {
				break
			}

			// Block until there's some update
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-*observer:
			}
		}
	}
	return nil
}
