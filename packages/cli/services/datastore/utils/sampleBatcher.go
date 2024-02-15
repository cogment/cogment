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
	"sync"
	"time"

	cogmentAPI "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/cogment/cogment/services/datastore/backend"
)

const (
	DefaultMaxBatchDelay = 10 * time.Millisecond
	DefaultMaxBatchSize  = 1000
)

type SampleBatcher interface {
	AddSample(sample *cogmentAPI.StoredTrialSample) error
}

type sampleBatcher struct {
	// protects batch and timer
	mu sync.Mutex
	// Number of samples to trigger a batch flush
	fullBatchSize int
	// Delay to trigger a batch flush
	maxBatchDelay time.Duration
	// backend to flush the samples
	backend backend.Backend
	// Samples to be flushed to the backend
	batch []*cogmentAPI.StoredTrialSample //TODO: maybe use interface{}
	timer *time.Timer
	err   chan<- error
	ctx   context.Context
}

func NewSampleBatcher(ctx context.Context, backend backend.Backend, errors chan<- error) SampleBatcher {
	return &sampleBatcher{
		fullBatchSize: DefaultMaxBatchSize,
		maxBatchDelay: DefaultMaxBatchDelay,
		backend:       backend,
		err:           errors,
		ctx:           ctx,
	}
}

// Starts a new timer
func (sb *sampleBatcher) resetTimer() {
	if sb.timer == nil {
		sb.timer = time.AfterFunc(sb.maxBatchDelay, sb.flush)
	} else {
		sb.timer.Reset(sb.maxBatchDelay)
	}
}

// Flushes batch to backend
func (sb *sampleBatcher) flush() {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	sb.timer.Stop()
	if len(sb.batch) > 0 {
		err := sb.backend.AddSamples(sb.ctx, sb.batch)
		if err != nil {
			sb.err <- err
		}
		//reset buffer
		sb.batch = sb.batch[:0]
	}
}

// Adds sample to buffer. Will eventually be flushed to backend.
func (sb *sampleBatcher) AddSample(sample *cogmentAPI.StoredTrialSample) error {
	sb.mu.Lock()
	size := len(sb.batch)
	if size == 0 {
		sb.resetTimer()
	}
	sb.batch = append(sb.batch, sample)
	sb.mu.Unlock()
	if size+1 >= sb.fullBatchSize {
		sb.flush()
	}
	return nil
}
