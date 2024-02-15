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
	"testing"
	"time"

	cogmentAPI "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/cogment/cogment/services/datastore/backend"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockBackend struct {
	mock.Mock
}

func (s *MockBackend) Destroy() {}

func (s *MockBackend) CreateOrUpdateTrials(_ context.Context, _ []*backend.TrialParams) error {
	return nil
}
func (s *MockBackend) RetrieveTrials(
	_ context.Context,
	_ backend.TrialFilter,
	_ int,
	_ int,
) (backend.TrialsInfoResult, error) {
	return backend.TrialsInfoResult{}, nil
}
func (s *MockBackend) ObserveTrials(_ context.Context,
	_ backend.TrialFilter,
	_ int,
	_ int,
	_ chan<- backend.TrialsInfoResult,
) error {
	return nil
}
func (s *MockBackend) DeleteTrials(_ context.Context, _ []string) error { return nil }

func (s *MockBackend) GetTrialParams(_ context.Context, _ []string) ([]*backend.TrialParams, error) {
	return nil, nil
}

func (s *MockBackend) AddSamples(ctx context.Context, samples []*cogmentAPI.StoredTrialSample) error {
	args := s.Mock.Called(ctx, samples)
	return args.Error(0)
}
func (s *MockBackend) ObserveSamples(
	_ context.Context,
	_ backend.TrialSampleFilter,
	_ chan<- *cogmentAPI.StoredTrialSample,
) error {
	return nil
}

func fakeSample() *cogmentAPI.StoredTrialSample {
	return &cogmentAPI.StoredTrialSample{}
}

func newSut(size int, timeout time.Duration) (*sampleBatcher, *MockBackend) {
	mockBackend := new(MockBackend)
	ctx := context.Background()
	sut := &sampleBatcher{
		fullBatchSize: size,
		maxBatchDelay: timeout,
		backend:       mockBackend,
		err:           make(chan<- error),
		ctx:           ctx,
	}
	return sut, mockBackend
}

func TestFlushWhenFullBatch(t *testing.T) {
	// inf timeout
	// small size
	timeout := 60 * time.Hour
	batchSize := 6
	chunks := 10
	sut, mockBackend := newSut(batchSize, timeout)

	mockBackend.On("AddSamples", mock.Anything, mock.Anything).Return(nil).Times(chunks)
	for i := 0; i < batchSize*chunks; i++ {
		assert.NoError(t, sut.AddSample(fakeSample()))
	}
	mockBackend.AssertExpectations(t)

}
func TestFlushWhenTimeout(t *testing.T) {
	// small timeout
	// inf size
	// assert flushed incomplete batch
	timeout := 0 * time.Nanosecond
	batchSize := 9001
	sut, mockBackend := newSut(batchSize, timeout)

	mockBackend.On("AddSamples", mock.Anything, mock.Anything).Return(nil).Times(1)
	assert.NoError(t, sut.AddSample(fakeSample()))
	time.Sleep(1 * time.Millisecond)
	mockBackend.AssertExpectations(t)
}

func TestFlushFullBatchesAndTimeoutOnTail(t *testing.T) {
	// n fullBatches + 1 incomplete batch
	batchSize := 3
	n := 5
	numberOfSamples := n*batchSize + 1
	sut, mockBackend := newSut(batchSize, 0*time.Nanosecond)

	mockBackend.On("AddSamples", mock.Anything, mock.Anything).Return(nil).Times(n + 1)
	for i := 0; i < numberOfSamples; i++ {
		assert.NoError(t, sut.AddSample(fakeSample()))
	}
	time.Sleep(1 * time.Millisecond)
	mockBackend.AssertExpectations(t)
}
