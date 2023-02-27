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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type item struct {
	value int
}

func TestObservableListSequential(t *testing.T) {
	l := CreateObservableList()

	one := &item{value: 1}
	two := &item{value: 2}
	three := &item{value: 3}

	l.Append(one, false)
	assert.Equal(t, 1, l.Len())
	l.Append(two, false)
	assert.Equal(t, 2, l.Len())
	l.Append(three, true)
	assert.Equal(t, 3, l.Len())

	{
		// First retrieval
		retrievedItems := make([]*item, 0)
		observer := make(ObservableListObserver)
		go func() {
			err := l.Observe(context.Background(), 0, observer)
			assert.NoError(t, err)
			close(observer)
		}()
		for retrievedItem := range observer {
			retrievedItems = append(retrievedItems, retrievedItem.(*item))
		}

		assert.Len(t, retrievedItems, 3)
		assert.Equal(t, one, retrievedItems[0])
		assert.Equal(t, two, retrievedItems[1])
		assert.Equal(t, three, retrievedItems[2])
	}

	{
		// Second retrieval
		observer := make(ObservableListObserver)
		go func() {
			err := l.Observe(context.Background(), 0, observer)
			assert.NoError(t, err)
			close(observer)
		}()

		retrievedOne, ok := <-observer
		assert.True(t, ok)
		assert.Equal(t, one, retrievedOne.(*item))

		retrievedTwo, ok := <-observer
		assert.True(t, ok)
		assert.Equal(t, two, retrievedTwo.(*item))

		retrievedThree, ok := <-observer
		assert.True(t, ok)
		assert.Equal(t, three, retrievedThree.(*item))

		retrievedFour, ok := <-observer
		assert.False(t, ok)
		assert.Nil(t, retrievedFour)
	}
}

func TestObservableObserverNew(t *testing.T) {
	t.Parallel() // This test involves goroutines and `time.Sleep`
	l := CreateObservableList()

	firstItemGroup := []*item{
		{value: 1},
		{value: 2},
		{value: 3},
		{value: 4},
	}

	secondItemGroup := []*item{
		{value: 5},
		{value: 6},
		{value: 7},
		{value: 8},
	}

	for _, item := range firstItemGroup {
		l.Append(item, false)
	}

	wg := sync.WaitGroup{}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			observer := make(ObservableListObserver)
			go func() {
				err := l.Observe(context.Background(), len(firstItemGroup), observer)
				assert.NoError(t, err)
				close(observer)
			}()

			for _, expectedItem := range secondItemGroup {
				retrievedItem, ok := <-observer
				assert.True(t, ok)
				assert.NotNil(t, retrievedItem)
				assert.Equal(t, expectedItem, retrievedItem.(*item))
			}

			retrievedEnd, ok := <-observer
			assert.False(t, ok)
			assert.Nil(t, retrievedEnd)
		}()
	}

	time.Sleep(100 * time.Millisecond)

	for i, item := range secondItemGroup {
		l.Append(item, i == len(secondItemGroup)-1)
	}

	wg.Wait()
}

func TestObservableListConcurrent(t *testing.T) {
	t.Parallel() // This test involves goroutines and `time.Sleep`

	l := CreateObservableList()

	items := []*item{
		{value: 1},
		{value: 2},
		{value: 3},
		{value: 4},
	}

	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		time.Sleep(10 * time.Millisecond)
		go func() {
			defer wg.Done()
			observer := make(ObservableListObserver)
			go func() {
				err := l.Observe(context.Background(), 0, observer)
				assert.NoError(t, err)
				close(observer)
			}()

			for _, expectedItem := range items {
				retrievedItem, ok := <-observer
				assert.True(t, ok)
				assert.NotNil(t, retrievedItem)
				assert.Equal(t, expectedItem, retrievedItem.(*item))
			}

			retrievedEnd, ok := <-observer
			assert.False(t, ok)
			assert.Nil(t, retrievedEnd)
		}()
	}

	l.Append(items[0], false)
	l.Append(items[1], false)

	time.Sleep(50 * time.Millisecond)

	l.Append(items[2], false)
	l.Append(items[3], true)

	wg.Wait()
}

func TestObservableListConcurrentFromOffset(t *testing.T) {
	t.Parallel() // This test involves goroutines and `time.Sleep`

	l := CreateObservableList()

	items := []*item{
		{value: 1},
		{value: 2},
		{value: 3},
		{value: 4},
	}

	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		time.Sleep(10 * time.Millisecond)
		go func() {
			defer wg.Done()
			observer := make(ObservableListObserver)
			go func() {
				err := l.Observe(context.Background(), 2, observer)
				assert.NoError(t, err)
				close(observer)
			}()

			for _, expectedItem := range items[2:] {
				retrievedItem, ok := <-observer
				assert.True(t, ok)
				assert.NotNil(t, retrievedItem)
				assert.Equal(t, expectedItem, retrievedItem.(*item))
			}

			retrievedEnd, ok := <-observer
			assert.False(t, ok)
			assert.Nil(t, retrievedEnd)
		}()
	}

	l.Append(items[0], false)
	l.Append(items[1], false)

	time.Sleep(50 * time.Millisecond)

	l.Append(items[2], false)
	l.Append(items[3], true)

	wg.Wait()
}

func TestObservableListConcurrentOneBlocker(t *testing.T) {
	t.Parallel() // This test involves goroutines and `time.Sleep`

	l := CreateObservableList()

	items := []*item{
		{value: 1},
		{value: 2},
		{value: 3},
	}

	wg := sync.WaitGroup{}
	// Some nice observers
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(delay time.Duration) {
			defer wg.Done()
			itemIdx := 0
			observer := make(ObservableListObserver)
			go func() {
				err := l.Observe(context.Background(), 0, observer)
				assert.NoError(t, err)
				close(observer)
			}()
			for retrievedItem := range observer {
				assert.Equal(t, items[itemIdx], retrievedItem.(*item))
				itemIdx++
				time.Sleep(delay)
			}
			assert.Equal(t, len(items), itemIdx)
		}(10 * time.Duration(i) * time.Millisecond)
	}

	// One blocking observer
	wg.Add(1)
	go func() {
		defer wg.Done()
		observer := make(ObservableListObserver)
		go func() {
			err := l.Observe(context.Background(), 0, observer)
			assert.NoError(t, err)
			close(observer)
		}()
		retrievedItem, ok := <-observer
		assert.True(t, ok)
		assert.Equal(t, items[0], retrievedItem.(*item))
	}()

	for itemIdx, item := range items {
		l.Append(item, itemIdx == len(items)-1)
		time.Sleep(50 * time.Millisecond)
	}
	wg.Wait()
}

func TestObservableListConcurrentOneCancelled(t *testing.T) {
	t.Parallel() // This test involves goroutines and `time.Sleep`

	l := CreateObservableList()

	items := []*item{
		{value: 1},
		{value: 2},
		{value: 3},
	}

	wg := sync.WaitGroup{}
	// Some nice observers
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(delay time.Duration) {
			defer wg.Done()
			itemIdx := 0
			observer := make(ObservableListObserver)
			go func() {
				err := l.Observe(context.Background(), 0, observer)
				assert.NoError(t, err)
				close(observer)
			}()
			for retrievedItem := range observer {
				assert.Equal(t, items[itemIdx], retrievedItem.(*item))
				itemIdx++
				time.Sleep(delay)
			}
			assert.Equal(t, len(items), itemIdx)
		}(10 * time.Duration(i) * time.Millisecond)
	}

	// One observer getting canceled
	wg.Add(1)
	go func() {
		defer wg.Done()
		observer := make(ObservableListObserver)
		cancellableCtx, cancel := context.WithCancel(context.Background())
		go func() {
			err := l.Observe(cancellableCtx, 0, observer)
			assert.ErrorIs(t, context.Canceled, err)
			close(observer)
		}()
		retrievedItem, ok := <-observer
		assert.True(t, ok)
		assert.Equal(t, items[0], retrievedItem.(*item))
		cancel()
	}()

	for itemIdx, item := range items {
		l.Append(item, itemIdx == len(items)-1)
		time.Sleep(50 * time.Millisecond)
	}
	wg.Wait()
}
