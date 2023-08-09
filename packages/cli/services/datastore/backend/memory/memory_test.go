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

package memory

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	grpcapi "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/cogment/cogment/services/datastore/backend"
	"github.com/cogment/cogment/services/datastore/backend/test"
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

func TestSuiteMemoryBackend(t *testing.T) {
	test.RunSuite(t, func() backend.Backend {
		b, err := CreateMemoryBackend(DefaultMaxSampleSize)
		assert.NoError(t, err)
		return b
	}, func(b backend.Backend) {
		mb := b.(*memoryBackend)
		mb.Destroy()
	})
}

func BenchmarkMemoryBackend(b *testing.B) {
	test.RunBenchmarks(b, func() backend.Backend {
		bck, err := CreateMemoryBackend(DefaultMaxSampleSize)
		assert.NoError(b, err)
		return bck
	}, func(bck backend.Backend) {
		mb := bck.(*memoryBackend)
		mb.Destroy()
	})
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
		err = b.CreateOrUpdateTrials(context.Background(), []*backend.TrialParams{{
			TrialID: trialID,
			Params:  generateTrialParams(12, 100),
		}})
		assert.NoError(t, err)
	}

	// Add samples to trial-1
	trial1SamplesCount := 1000
	{
		for i := 0; i < trial1SamplesCount; i++ {
			err = b.AddSamples(
				context.Background(),
				[]*grpcapi.StoredTrialSample{generateSample("trial-1", 12, i == trial1SamplesCount-1)},
			)
			assert.NoError(t, err)
		}

		time.Sleep(100 * time.Millisecond) // Give time to the trial eviction worker

		r, err := b.RetrieveTrials(
			context.Background(),
			backend.NewTrialFilter([]string{"trial-1", "trial-2", "trial-3"}, map[string]string{}),
			0, -1,
		)
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
			err = b.AddSamples(
				context.Background(),
				[]*grpcapi.StoredTrialSample{generateSample("trial-2", 12, i == trial2SamplesCount-1)},
			)
			assert.NoError(t, err)
		}

		time.Sleep(100 * time.Millisecond) // Give time to the trial eviction worker

		r, err := b.RetrieveTrials(
			context.Background(),
			backend.NewTrialFilter([]string{"trial-1", "trial-2", "trial-3"}, map[string]string{}),
			0, -1,
		)
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
			err = b.AddSamples(
				context.Background(),
				[]*grpcapi.StoredTrialSample{generateSample("trial-3", 12, i == trial3SamplesCount-1)},
			)
			assert.NoError(t, err)
		}

		time.Sleep(100 * time.Millisecond) // Give time to the trial eviction worker

		r, err := b.RetrieveTrials(
			context.Background(),
			backend.NewTrialFilter([]string{"trial-1", "trial-2", "trial-3"}, map[string]string{}),
			0, -1,
		)
		assert.NoError(t, err)
		assert.Equal(t, trial1SamplesCount, r.TrialInfos[0].SamplesCount)
		assert.Equal(t, 0, r.TrialInfos[0].StoredSamplesCount)
		assert.Equal(t, trial2SamplesCount, r.TrialInfos[1].SamplesCount)
		assert.Equal(t, trial2SamplesCount, r.TrialInfos[1].StoredSamplesCount)
		assert.Equal(t, trial3SamplesCount, r.TrialInfos[2].SamplesCount)
		assert.Equal(t, trial3SamplesCount, r.TrialInfos[2].StoredSamplesCount)
	}

}
