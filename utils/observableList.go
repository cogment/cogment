// Copyright 2021 AI Redefined Inc. <dev+cogment@ai-r.com>
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
	Item(index int) (ObservableListItem, bool)
	Append(item ObservableListItem, last bool)
	Observe(ctx context.Context, from int, out chan<- ObservableListItem) error
}

type observer chan bool

type observableList struct {
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
	return len(l.items)
}

func (l *observableList) Item(index int) (ObservableListItem, bool) {
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
	l.items = append(l.items, item)
	l.ended = lastItem
	l.observersLock.RLock() // This locks the iteration thus making sure the goroutine is called with an observer that exist
	defer l.observersLock.RUnlock()
	for o := range l.observers {
		o := o
		go func() {
			*o <- lastItem
		}()
	}
}

func (l *observableList) Observe(ctx context.Context, from int, out chan<- ObservableListItem) error {
	if l.ended {
		for _, sample := range l.items[from:] {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case out <- sample:
			}
		}
	} else {
		observer := l.registerObserver()
		defer l.unregisterObserver(observer)
		i := from
		for {
			// Read everything up to the current count
			for ; i < len(l.items); i++ {
				sample := l.items[i]
				select {
				case <-ctx.Done():
					return ctx.Err()
				case out <- sample:
				}
			}

			if l.ended {
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
