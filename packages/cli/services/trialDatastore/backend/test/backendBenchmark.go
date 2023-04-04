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

package test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	grpcapi "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/cogment/cogment/services/trialDatastore/backend"
	"github.com/stretchr/testify/assert"
)

type BenchmarkCfg struct {
	trialCount           int
	samplesPerTrialCount int
	actorCount           int
	actorPayloadSize     int
	consumerCount        int
}

func runBenchmark(
	b *testing.B,
	createBackend func() backend.Backend,
	destroyBackend func(backend.Backend),
	cfg BenchmarkCfg,
) {
	b.ReportAllocs()

	bck := createBackend()
	defer destroyBackend(bck)

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		trialIDs := []string{}
		for trialIdx := 0; trialIdx < cfg.trialCount; trialIdx++ {
			trialID := fmt.Sprintf("trial-%d-%d", n, trialIdx)
			trialIDs = append(trialIDs, trialID)
		}

		wg := sync.WaitGroup{}

		// Asynchronously consume the samples from all trials
		for consumerIdx := 0; consumerIdx < cfg.consumerCount; consumerIdx++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				trialsObserver := make(backend.TrialsInfoObserver)
				go func() {
					err := bck.ObserveTrials(
						context.Background(),
						backend.NewTrialFilter(trialIDs, map[string]string{}),
						-1, -1,
						trialsObserver,
					)
					assert.NoError(b, err)
					close(trialsObserver)
				}()

				// Mock consuming the trials (we just want to wait until they are all created)
				for range trialsObserver {
				}

				samplesObserver := make(backend.TrialSampleObserver)
				go func() {
					err := bck.ObserveSamples(
						context.Background(),
						backend.TrialSampleFilter{TrialIDs: trialIDs},
						samplesObserver,
					)
					assert.NoError(b, err)
					close(samplesObserver)
				}()

				for range samplesObserver {
				}
			}()
		}

		// Asynchronously creating trials and add samples to them
		for _, trialID := range trialIDs {
			trialID := trialID
			wg.Add(1)
			go func() {
				defer wg.Done()

				trialParams := &backend.TrialParams{
					TrialID: trialID,
					UserID:  "benchmark",
					Params:  generateTrialParams(cfg.actorCount, 100),
				}
				err := bck.CreateOrUpdateTrials(context.Background(), []*backend.TrialParams{trialParams})
				assert.NoError(b, err)

				for sampleIdx := 0; sampleIdx < cfg.samplesPerTrialCount; sampleIdx++ {
					sample := generateSample(
						trialID,
						cfg.actorCount,
						cfg.actorPayloadSize,
						sampleIdx == cfg.samplesPerTrialCount-1,
					)
					err = bck.AddSamples(context.Background(), []*grpcapi.StoredTrialSample{sample})
					assert.NoError(b, err)
				}
			}()
		}

		// Wait for all the async tasks to end
		wg.Wait()

		// Remove the trials
		err := bck.DeleteTrials(context.Background(), trialIDs)
		assert.NoError(b, err)
	}
}

func RunBenchmarks(b *testing.B, createBackend func() backend.Backend, destroyBackend func(backend.Backend)) {
	b.Run("5x100x10x1KB_samples_0_consumers", func(b *testing.B) {
		runBenchmark(
			b, createBackend, destroyBackend,
			BenchmarkCfg{trialCount: 5, samplesPerTrialCount: 100, actorCount: 10, actorPayloadSize: 1024, consumerCount: 0},
		)
	})
	b.Run("5x100x10x1KB_samples_5_consumers", func(b *testing.B) {
		runBenchmark(
			b, createBackend, destroyBackend,
			BenchmarkCfg{trialCount: 5, samplesPerTrialCount: 100, actorCount: 10, actorPayloadSize: 1024, consumerCount: 5},
		)
	})
	b.Run("50x500x10x1KB_samples_5_consumers", func(b *testing.B) {
		runBenchmark(
			b, createBackend, destroyBackend,
			BenchmarkCfg{trialCount: 50, samplesPerTrialCount: 500, actorCount: 10, actorPayloadSize: 1024, consumerCount: 5},
		)
	})
	b.Run("50x500x5x1/4KB_samples_5_consumers", func(b *testing.B) {
		runBenchmark(
			b, createBackend, destroyBackend,
			BenchmarkCfg{trialCount: 50, samplesPerTrialCount: 500, actorCount: 5, actorPayloadSize: 256, consumerCount: 5},
		)
	})
}
