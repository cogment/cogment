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
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	grpcapi "github.com/cogment/cogment-trial-datastore/grpcapi/cogment/api"
)

var nextTickID uint64 // = 0

func generateTrialParams(actorCount int, maxSteps uint32) *grpcapi.TrialParams {
	params := &grpcapi.TrialParams{
		Actors:   make([]*grpcapi.ActorParams, actorCount),
		MaxSteps: maxSteps,
	}
	return params
}

func generateSample(trialID string, actorCount int, end bool) *grpcapi.StoredTrialSample {
	sample := &grpcapi.StoredTrialSample{
		TrialId:      trialID,
		TickId:       nextTickID,
		Timestamp:    uint64(time.Now().Unix()),
		State:        grpcapi.TrialState_RUNNING,
		ActorSamples: make([]*grpcapi.StoredTrialActorSample, actorCount),
	}
	if end {
		sample.State = grpcapi.TrialState_ENDED
	}
	nextTickID++
	return sample
}

func TestCreateBackend(t *testing.T) {
	b, err := CreateMemoryBackend(DefaultMaxSampleSize)
	assert.NoError(t, err)
	defer b.Destroy()

	assert.NotNil(t, b)
}

func TestCreateOrUpdateTrials(t *testing.T) {
	b, err := CreateMemoryBackend(DefaultMaxSampleSize)
	assert.NoError(t, err)
	assert.NotNil(t, b)
	defer b.Destroy()

	{
		err = b.CreateOrUpdateTrials(context.Background(), []*TrialParams{
			{
				TrialID: "trial-1",
				Params:  generateTrialParams(2, 100),
			},
			{
				TrialID: "trial-2",
				Params:  generateTrialParams(4, 150),
			},
		})
		assert.NoError(t, err)
	}

	{
		r, err := b.RetrieveTrials(context.Background(), []string{}, -1, -1)
		assert.NoError(t, err)

		assert.Len(t, r.TrialInfos, 2)

		assert.Equal(t, "trial-1", r.TrialInfos[0].TrialID)
		assert.Equal(t, grpcapi.TrialState_UNKNOWN, r.TrialInfos[0].State)
		assert.Equal(t, 0, r.TrialInfos[0].SamplesCount)
		assert.Equal(t, 0, r.TrialInfos[0].StoredSamplesCount)

		assert.Equal(t, "trial-2", r.TrialInfos[1].TrialID)
		assert.Equal(t, grpcapi.TrialState_UNKNOWN, r.TrialInfos[1].State)
		assert.Equal(t, 0, r.TrialInfos[1].SamplesCount)
		assert.Equal(t, 0, r.TrialInfos[1].StoredSamplesCount)

		assert.Equal(t, 2, r.NextTrialIdx)
	}

	{
		err = b.CreateOrUpdateTrials(context.Background(), []*TrialParams{
			{
				TrialID: "trial-2",
				Params:  generateTrialParams(8, 12),
			},
			{
				TrialID: "trial-3",
				Params:  generateTrialParams(5, 7),
			},
		})
		assert.NoError(t, err)
	}

	{
		r, err := b.RetrieveTrials(context.Background(), []string{}, 0, 5)
		assert.NoError(t, err)

		assert.Len(t, r.TrialInfos, 3)

		assert.Equal(t, "trial-1", r.TrialInfos[0].TrialID)
		assert.Equal(t, "trial-2", r.TrialInfos[1].TrialID)
		assert.Equal(t, "trial-3", r.TrialInfos[2].TrialID)

		assert.Equal(t, 3, r.NextTrialIdx)
	}
}

func TestObserveTrials(t *testing.T) {
	t.Parallel() // This test involves goroutines and `time.Sleep`

	b, err := CreateMemoryBackend(DefaultMaxSampleSize)
	assert.NoError(t, err)
	assert.NotNil(t, b)
	defer b.Destroy()

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		// Waiting before actually adding the trials
		defer wg.Done()
		time.Sleep(100 * time.Millisecond)
		err = b.CreateOrUpdateTrials(context.Background(), []*TrialParams{
			{
				TrialID: "trial-1",
				Params:  generateTrialParams(2, 100),
			},
			{
				TrialID: "trial-2",
				Params:  generateTrialParams(4, 150),
			},
		})
		assert.NoError(t, err)
	}()

	wg.Add(1)
	go func() {
		// Can I retrieve just trial 2
		defer wg.Done()
		observer := make(TrialsInfoObserver)
		go func() {
			err := b.ObserveTrials(context.Background(), []string{"trial-2"}, 0, -1, observer)
			assert.NoError(t, err)
			close(observer)
		}()
		results := []*TrialInfo{}
		nextTrialIdx := 0
		for r := range observer {
			results = append(results, r.TrialInfos...)
			nextTrialIdx = r.NextTrialIdx
		}

		assert.Len(t, results, 1)

		assert.Equal(t, "trial-2", results[0].TrialID)

		assert.Equal(t, 2, nextTrialIdx)
	}()

	wg.Add(1)
	go func() {
		// Can I retrieve trial 2 and trial 1
		defer wg.Done()
		observer := make(TrialsInfoObserver)
		go func() {
			err := b.ObserveTrials(context.Background(), []string{"trial-2", "trial-1"}, 0, -1, observer)
			assert.NoError(t, err)
			close(observer)
		}()
		results := []*TrialInfo{}
		nextTrialIdx := 0
		for r := range observer {
			results = append(results, r.TrialInfos...)
			nextTrialIdx = r.NextTrialIdx
		}

		assert.Len(t, results, 2)

		assert.Equal(t, "trial-1", results[0].TrialID)
		assert.Equal(t, "trial-2", results[1].TrialID)

		assert.Equal(t, 2, nextTrialIdx)
	}()

	wg.Add(1)
	go func() {
		// Can I retrieve the 2 first trials
		defer wg.Done()
		observer := make(TrialsInfoObserver)
		go func() {
			err := b.ObserveTrials(context.Background(), []string{}, 0, 2, observer)
			assert.NoError(t, err)
			close(observer)
		}()
		results := []*TrialInfo{}
		nextTrialIdx := 0
		for r := range observer {
			results = append(results, r.TrialInfos...)
			nextTrialIdx = r.NextTrialIdx
		}

		assert.Len(t, results, 2)

		assert.Equal(t, "trial-1", results[0].TrialID)
		assert.Equal(t, "trial-2", results[1].TrialID)

		assert.Equal(t, 2, nextTrialIdx)
	}()

	wg.Wait()
}

func TestDeleteTrials(t *testing.T) {
	b, err := CreateMemoryBackend(DefaultMaxSampleSize)
	assert.NoError(t, err)
	assert.NotNil(t, b)
	defer b.Destroy()

	{
		err = b.CreateOrUpdateTrials(context.Background(), []*TrialParams{
			{
				TrialID: "A",
				Params:  generateTrialParams(1, 2),
			},
			{
				TrialID: "B",
				Params:  generateTrialParams(3, 4),
			},
			{
				TrialID: "C",
				Params:  generateTrialParams(5, 6),
			},
		})
		assert.NoError(t, err)
	}

	{
		err = b.DeleteTrials(context.Background(), []string{"A", "C"})
		assert.NoError(t, err)
	}

	{
		r, err := b.RetrieveTrials(context.Background(), []string{}, -1, -1)
		assert.NoError(t, err)

		assert.Len(t, r.TrialInfos, 1)

		assert.Equal(t, "B", r.TrialInfos[0].TrialID)

		assert.Equal(t, 2, r.NextTrialIdx)
	}

	{
		err = b.DeleteTrials(context.Background(), []string{"B", "C", "D"})
		assert.NoError(t, err)
	}

	{
		r, err := b.RetrieveTrials(context.Background(), []string{}, -1, -1)
		assert.NoError(t, err)

		assert.Len(t, r.TrialInfos, 0)
	}
}

func TestGetTrialParams(t *testing.T) {
	b, err := CreateMemoryBackend(DefaultMaxSampleSize)
	assert.NoError(t, err)
	assert.NotNil(t, b)
	defer b.Destroy()

	{
		err = b.CreateOrUpdateTrials(context.Background(), []*TrialParams{
			{
				TrialID: "A",
				Params:  generateTrialParams(2, 100),
			},
			{
				TrialID: "B",
				Params:  generateTrialParams(4, 150),
			},
		})
		assert.NoError(t, err)
	}

	{
		r, err := b.RetrieveTrials(context.Background(), []string{}, -1, -1)
		assert.NoError(t, err)

		assert.Len(t, r.TrialInfos, 2)

		assert.Equal(t, "A", r.TrialInfos[0].TrialID)
		assert.Equal(t, "B", r.TrialInfos[1].TrialID)
	}

	{
		trialsParams, err := b.GetTrialParams(context.Background(), []string{"A", "B"})
		assert.NoError(t, err)

		assert.Len(t, trialsParams, 2)

		assert.Equal(t, "A", trialsParams[0].TrialID)
		assert.Len(t, trialsParams[0].Params.Actors, 2)
		assert.Equal(t, uint32(100), trialsParams[0].Params.MaxSteps)
		assert.Equal(t, "B", trialsParams[1].TrialID)
		assert.Len(t, trialsParams[1].Params.Actors, 4)
		assert.Equal(t, uint32(150), trialsParams[1].Params.MaxSteps)
	}
}

func TestAddSamples(t *testing.T) {
	b, err := CreateMemoryBackend(DefaultMaxSampleSize)
	assert.NoError(t, err)
	assert.NotNil(t, b)
	defer b.Destroy()

	err = b.CreateOrUpdateTrials(context.Background(), []*TrialParams{{
		TrialID: "my-trial",
		Params:  generateTrialParams(12, 100),
	}})
	assert.NoError(t, err)

	firstSample := generateSample("my-trial", 12, false)
	secondSample := generateSample("my-trial", 12, false)
	err = b.AddSamples(context.Background(), []*grpcapi.StoredTrialSample{firstSample, secondSample, generateSample("my-trial", 12, false)})
	assert.NoError(t, err)

	r, err := b.RetrieveTrials(context.Background(), []string{"my-trial"}, -1, -1)
	assert.NoError(t, err)

	assert.Len(t, r.TrialInfos, 1)

	assert.Equal(t, "my-trial", r.TrialInfos[0].TrialID)
	assert.Equal(t, grpcapi.TrialState_RUNNING, r.TrialInfos[0].State)
	assert.Equal(t, 3, r.TrialInfos[0].SamplesCount)
	assert.Equal(t, 3, r.TrialInfos[0].StoredSamplesCount)

	err = b.AddSamples(context.Background(), []*grpcapi.StoredTrialSample{
		generateSample("my-trial", 12, false),
		generateSample("my-trial", 12, false),
	})
	assert.NoError(t, err)

	r, err = b.RetrieveTrials(context.Background(), []string{"my-trial"}, -1, -1)
	assert.NoError(t, err)

	assert.Len(t, r.TrialInfos, 1)

	assert.Equal(t, "my-trial", r.TrialInfos[0].TrialID)
	assert.Equal(t, grpcapi.TrialState_RUNNING, r.TrialInfos[0].State)
	assert.Equal(t, 5, r.TrialInfos[0].SamplesCount)
	assert.Equal(t, 5, r.TrialInfos[0].StoredSamplesCount)

	observer := make(TrialSampleObserver)
	go func() {
		err := b.ObserveSamples(context.Background(), TrialSampleFilter{TrialIDs: []string{"my-trial"}}, observer)
		assert.NoError(t, err)
		close(observer)
	}()
	sampleResult := <-observer
	assert.Equal(t, "my-trial", sampleResult.TrialId)
	assert.Equal(t, firstSample.TickId, sampleResult.TickId)

	sampleResult = <-observer
	assert.Equal(t, "my-trial", sampleResult.TrialId)
	assert.Equal(t, secondSample.TickId, sampleResult.TickId)
}

func TestConcurrentAddAndObserveSamples(t *testing.T) {
	t.Parallel() // This test involves goroutines and `time.Sleep`

	b, err := CreateMemoryBackend(DefaultMaxSampleSize)
	assert.NoError(t, err)
	assert.NotNil(t, b)
	defer b.Destroy()

	err = b.CreateOrUpdateTrials(context.Background(), []*TrialParams{{
		TrialID: "my-trial",
		Params:  generateTrialParams(12, 100),
	}})
	assert.NoError(t, err)

	samples := make([]*grpcapi.StoredTrialSample, 20)
	for sampleIdx := range samples {
		samples[sampleIdx] = generateSample("my-trial", 12, sampleIdx == len(samples)-1)
	}

	wg := sync.WaitGroup{}
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sampleIdx := 0
			observer := make(TrialSampleObserver)
			go func() {
				err := b.ObserveSamples(context.Background(), TrialSampleFilter{TrialIDs: []string{"my-trial"}}, observer)
				assert.NoError(t, err)
				close(observer)
			}()
			for sampleResult := range observer {
				assert.Equal(t, "my-trial", sampleResult.TrialId)
				assert.Equal(t, samples[sampleIdx].TickId, sampleResult.TickId)
				assert.Equal(t, samples[sampleIdx].Timestamp, sampleResult.Timestamp)
				sampleIdx++

				// Simulating different consumption speed
				time.Sleep(time.Duration(i) * time.Millisecond)
			}
			assert.Equal(t, len(samples), sampleIdx)
		}(i)
	}

	for _, sample := range samples {
		time.Sleep(50 * time.Millisecond)
		err = b.AddSamples(context.Background(), []*grpcapi.StoredTrialSample{sample})
		assert.NoError(t, err)
	}
	wg.Wait()
}

func TestTriaEviction(t *testing.T) {
	// Uncomment to see the log from the trial eviction worker
	// log.SetLevel(log.DebugLevel)
	b, err := CreateMemoryBackend(100000) // Should be enough for 2 trials worth of sample data.
	assert.NoError(t, err)
	assert.NotNil(t, b)
	defer b.Destroy()

	// Create trials
	for _, trialID := range []string{"trial-1", "trial-2", "trial-3"} {
		err = b.CreateOrUpdateTrials(context.Background(), []*TrialParams{{
			TrialID: trialID,
			Params:  generateTrialParams(12, 100),
		}})
		assert.NoError(t, err)
	}

	// Add samples to trial-1
	trial1SamplesCount := 1000
	{
		for i := 0; i < trial1SamplesCount; i++ {
			err = b.AddSamples(context.Background(), []*grpcapi.StoredTrialSample{generateSample("trial-1", 12, i == trial1SamplesCount-1)})
			assert.NoError(t, err)
		}

		time.Sleep(100 * time.Millisecond) // Give time to the trial eviction worker

		r, err := b.RetrieveTrials(context.Background(), []string{"trial-1", "trial-2", "trial-3"}, 0, -1)
		assert.NoError(t, err)
		assert.Equal(t, trial1SamplesCount, r.TrialInfos[0].SamplesCount)
		assert.Equal(t, trial1SamplesCount, r.TrialInfos[0].StoredSamplesCount)
		assert.Equal(t, 0, r.TrialInfos[1].SamplesCount)
		assert.Equal(t, 0, r.TrialInfos[1].StoredSamplesCount)
		assert.Equal(t, 0, r.TrialInfos[2].SamplesCount)
		assert.Equal(t, 0, r.TrialInfos[2].StoredSamplesCount)
	}

	// Add samples to trial-2
	trial2SamplesCount := 1000
	{
		for i := 0; i < trial2SamplesCount; i++ {
			err = b.AddSamples(context.Background(), []*grpcapi.StoredTrialSample{generateSample("trial-2", 12, i == trial2SamplesCount-1)})
			assert.NoError(t, err)
		}

		time.Sleep(100 * time.Millisecond) // Give time to the trial eviction worker

		r, err := b.RetrieveTrials(context.Background(), []string{"trial-1", "trial-2", "trial-3"}, 0, -1)
		assert.NoError(t, err)
		assert.Equal(t, trial1SamplesCount, r.TrialInfos[0].SamplesCount)
		assert.Equal(t, trial1SamplesCount, r.TrialInfos[0].StoredSamplesCount)
		assert.Equal(t, trial2SamplesCount, r.TrialInfos[1].SamplesCount)
		assert.Equal(t, trial2SamplesCount, r.TrialInfos[1].StoredSamplesCount)
		assert.Equal(t, 0, r.TrialInfos[2].SamplesCount)
		assert.Equal(t, 0, r.TrialInfos[2].StoredSamplesCount)
	}

	// Add samples to trial-3
	trial3SamplesCount := 1000
	{
		for i := 0; i < trial3SamplesCount; i++ {
			err = b.AddSamples(context.Background(), []*grpcapi.StoredTrialSample{generateSample("trial-3", 12, i == trial3SamplesCount-1)})
			assert.NoError(t, err)
		}

		time.Sleep(100 * time.Millisecond) // Give time to the trial eviction worker

		r, err := b.RetrieveTrials(context.Background(), []string{"trial-1", "trial-2", "trial-3"}, 0, -1)
		assert.NoError(t, err)
		assert.Equal(t, trial1SamplesCount, r.TrialInfos[0].SamplesCount)
		assert.Equal(t, 0, r.TrialInfos[0].StoredSamplesCount)
		assert.Equal(t, trial2SamplesCount, r.TrialInfos[1].SamplesCount)
		assert.Equal(t, trial2SamplesCount, r.TrialInfos[1].StoredSamplesCount)
		assert.Equal(t, trial3SamplesCount, r.TrialInfos[2].SamplesCount)
		assert.Equal(t, trial3SamplesCount, r.TrialInfos[2].StoredSamplesCount)
	}

}
