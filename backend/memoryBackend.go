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

package backend

import (
	"fmt"

	grpcapi "github.com/cogment/cogment-activity-logger/grpcapi/cogment/api"
)

type trialData struct {
	status          TrialStatus
	samplesCount    int
	params          *grpcapi.TrialParams
	paramsAvailable chan bool
	samplesChannel  chan *grpcapi.DatalogSample
}

func createTrialInfo(trialID string, data *trialData) TrialInfo {
	return TrialInfo{
		TrialID:            trialID,
		Status:             data.status,
		SamplesCount:       data.samplesCount,
		StoredSamplesCount: len(data.samplesChannel),
	}
}

type memoryBackend struct {
	capacity uint
	trials   map[string]*trialData
}

// CreateMemoryBackend creates a Backend that will store at most "capacity" samples per trial.
func CreateMemoryBackend(capacity uint) (Backend, error) {
	backend := &memoryBackend{
		capacity: capacity,
		trials:   make(map[string]*trialData),
	}

	return backend, nil
}

// Destroy terminates the underlying storage
func (b *memoryBackend) Destroy() {
	// Nothing
}

func (b *memoryBackend) retrieveOrCreateTrial(trialID string, create bool) (*trialData, error) {
	if data, exists := b.trials[trialID]; exists {
		return data, nil
	}
	if !create {
		return nil, fmt.Errorf("unknown trial %q", trialID)
	}
	data := &trialData{
		status:          TrialNotStarted,
		samplesCount:    0,
		params:          nil,
		paramsAvailable: make(chan bool, 1),
		samplesChannel:  make(chan *grpcapi.DatalogSample, b.capacity),
	}
	b.trials[trialID] = data
	return data, nil
}

func (b *memoryBackend) OnTrialStart(trialID string, trialParams *grpcapi.TrialParams) (TrialInfo, error) {
	data, error := b.retrieveOrCreateTrial(trialID, true)
	if error != nil {
		return TrialInfo{}, error
	}
	if data.status != TrialNotStarted {
		return TrialInfo{}, fmt.Errorf("trial %q has already been started", trialID)
	}
	data.status = TrialRunning
	data.params = trialParams
	data.paramsAvailable <- true
	close(data.paramsAvailable)
	return createTrialInfo(trialID, data), nil
}

func (b *memoryBackend) OnTrialEnd(trialID string) (TrialInfo, error) {
	data, error := b.retrieveOrCreateTrial(trialID, false)
	if error != nil {
		return TrialInfo{}, error
	}
	if data.status != TrialRunning {
		return TrialInfo{}, fmt.Errorf("trial %q is not running", trialID)
	}
	data.status = TrialEnded
	close(data.samplesChannel)
	return createTrialInfo(trialID, data), nil
}

func (b *memoryBackend) OnTrialSample(trialID string, sample *grpcapi.DatalogSample) (TrialInfo, error) {
	data, error := b.retrieveOrCreateTrial(trialID, false)
	if error != nil {
		return TrialInfo{}, error
	}
	// Discard enough sample to have room for the new one
	for len(data.samplesChannel) >= cap(data.samplesChannel) {
		<-data.samplesChannel
	}
	data.samplesChannel <- sample
	data.samplesCount++
	return createTrialInfo(trialID, data), nil
}

func (b *memoryBackend) GetTrialInfo(trialID string) (TrialInfo, error) {
	data, error := b.retrieveOrCreateTrial(trialID, true)
	if error != nil {
		return TrialInfo{}, error
	}
	return createTrialInfo(trialID, data), nil
}

func (b *memoryBackend) GetTrialParams(trialID string) (*grpcapi.TrialParams, error) {
	result := <-b.GetTrialParamsAsync(trialID)
	return result.Ok, result.Err
}

func (b *memoryBackend) GetTrialParamsAsync(trialID string) TrialParamsPromise {
	promise := make(chan TrialParamsResult)
	go func() {
		data, error := b.retrieveOrCreateTrial(trialID, true)
		defer close(promise)
		if error != nil {
			promise <- TrialParamsResult{Err: error}
			return
		}
		for range data.paramsAvailable {
			// NOTHING
		}
		promise <- TrialParamsResult{Ok: data.params}
	}()
	return promise
}

func (b *memoryBackend) ConsumeTrialSamples(trialID string) DatalogSampleStream {
	stream := make(chan DatalogSampleResult)
	go func() {
		data, error := b.retrieveOrCreateTrial(trialID, true)
		defer close(stream)
		if error != nil {
			stream <- DatalogSampleResult{Err: error}
			return
		}
		for sample := range data.samplesChannel {
			stream <- DatalogSampleResult{Ok: sample}
		}
	}()
	return stream
}
