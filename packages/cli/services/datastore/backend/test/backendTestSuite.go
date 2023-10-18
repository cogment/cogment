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
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/openlyinc/pointy"
	"github.com/stretchr/testify/assert"

	cogmentAPI "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/cogment/cogment/services/datastore/backend"
)

var nextTickID uint64 // = 0

func generateTrialParams(actorCount int, maxSteps uint32) *cogmentAPI.TrialParams {
	params := &cogmentAPI.TrialParams{
		Actors:   make([]*cogmentAPI.ActorParams, actorCount),
		MaxSteps: maxSteps,
	}
	for actorIdx := range params.Actors {
		actorParams := &cogmentAPI.ActorParams{}
		actorParams.Name = fmt.Sprintf("actor-%d", actorIdx)
		params.Actors[actorIdx] = actorParams
	}
	return params
}

func generateTrialParamsWithProperties(
	actorCount int,
	maxSteps uint32,
	properties map[string]string,
) *cogmentAPI.TrialParams {
	params := generateTrialParams(actorCount, maxSteps)
	params.Properties = properties
	return params
}

const bytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func makeRandomBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = bytes[rand.Intn(len(bytes))]
	}
	return b
}

func makePayloads(payloadSize int, payloadCount int) [][]byte {
	payloads := make([][]byte, payloadCount)
	for i := range payloads {
		payloads[i] = makeRandomBytes(payloadSize)
	}
	return payloads
}

func makeActorSamples(actorCount int, actorPayloadSize int) []*cogmentAPI.StoredTrialActorSample {
	actorSamples := make([]*cogmentAPI.StoredTrialActorSample, actorCount)
	for i := range actorSamples {
		actorSample := &cogmentAPI.StoredTrialActorSample{}
		actorSample.Actor = uint32(i)
		actorSample.Observation = pointy.Uint32(uint32(i % actorPayloadSize))
		actorSample.Action = pointy.Uint32(uint32(i % actorPayloadSize))
		actorSamples[i] = actorSample
	}
	return actorSamples
}

func generateSample(trialID string, actorCount int, actorPayloadSize int, end bool) *cogmentAPI.StoredTrialSample {
	sample := &cogmentAPI.StoredTrialSample{
		TrialId:      trialID,
		TickId:       nextTickID,
		Timestamp:    uint64(time.Now().Unix()),
		State:        cogmentAPI.TrialState_RUNNING,
		ActorSamples: makeActorSamples(actorCount, actorPayloadSize),
		Payloads:     makePayloads(actorCount, actorPayloadSize),
	}
	if end {
		sample.State = cogmentAPI.TrialState_ENDED
	}
	nextTickID++
	return sample
}

func extractTrialIDs(trialInfos []*backend.TrialInfo) []string {
	trialIDs := []string{}
	for _, trialInfo := range trialInfos {
		trialIDs = append(trialIDs, trialInfo.TrialID)
	}
	return trialIDs
}

// RunSuite runs the full backend test suite
func RunSuite(t *testing.T, createBackend func() backend.Backend, destroyBackend func(backend.Backend)) {
	t.Run("TestCreateBackend", func(t *testing.T) {
		b := createBackend()
		defer destroyBackend(b)

		assert.NotNil(t, b)
	})
	t.Run("TestCreateOrUpdateTrials", func(t *testing.T) {
		b := createBackend()
		defer destroyBackend(b)

		{
			err := b.CreateOrUpdateTrials(context.Background(), []*backend.TrialParams{
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
			r, err := b.RetrieveTrials(context.Background(), backend.NewTrialFilter([]string{}, map[string]string{}), -1, -1)
			assert.NoError(t, err)

			assert.Len(t, r.TrialInfos, 2)

			assert.ElementsMatch(t, r.TrialInfos, []*backend.TrialInfo{
				{
					TrialID:            "trial-1",
					State:              cogmentAPI.TrialState_UNKNOWN,
					SamplesCount:       0,
					StoredSamplesCount: 0,
				},
				{
					TrialID:            "trial-2",
					State:              cogmentAPI.TrialState_UNKNOWN,
					SamplesCount:       0,
					StoredSamplesCount: 0,
				},
			})
		}

		{
			err := b.CreateOrUpdateTrials(context.Background(), []*backend.TrialParams{
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
			r, err := b.RetrieveTrials(context.Background(), backend.NewTrialFilter([]string{}, map[string]string{}), 0, 5)
			assert.NoError(t, err)

			assert.Len(t, r.TrialInfos, 3)

			assert.ElementsMatch(t, r.TrialInfos, []*backend.TrialInfo{
				{
					TrialID:            "trial-1",
					State:              cogmentAPI.TrialState_UNKNOWN,
					SamplesCount:       0,
					StoredSamplesCount: 0,
				},
				{
					TrialID:            "trial-2",
					State:              cogmentAPI.TrialState_UNKNOWN,
					SamplesCount:       0,
					StoredSamplesCount: 0,
				},
				{
					TrialID:            "trial-3",
					State:              cogmentAPI.TrialState_UNKNOWN,
					SamplesCount:       0,
					StoredSamplesCount: 0,
				},
			})
		}

		{
			r1, err := b.RetrieveTrials(context.Background(), backend.NewTrialFilter([]string{}, map[string]string{}), 0, 2)
			assert.NoError(t, err)

			assert.Len(t, r1.TrialInfos, 2)

			assert.ElementsMatch(t, r1.TrialInfos, []*backend.TrialInfo{
				{
					TrialID:            "trial-1",
					State:              cogmentAPI.TrialState_UNKNOWN,
					SamplesCount:       0,
					StoredSamplesCount: 0,
				},
				{
					TrialID:            "trial-2",
					State:              cogmentAPI.TrialState_UNKNOWN,
					SamplesCount:       0,
					StoredSamplesCount: 0,
				},
			})

			assert.Equal(t, 2, r1.NextTrialIdx)

			r2, err := b.RetrieveTrials(
				context.Background(),
				backend.NewTrialFilter([]string{}, map[string]string{}),
				r1.NextTrialIdx, 2,
			)
			assert.NoError(t, err)

			assert.Len(t, r2.TrialInfos, 1)

			assert.ElementsMatch(t, r2.TrialInfos, []*backend.TrialInfo{
				{
					TrialID:            "trial-3",
					State:              cogmentAPI.TrialState_UNKNOWN,
					SamplesCount:       0,
					StoredSamplesCount: 0,
				},
			})
		}
	})
	t.Run("TestRetrieveFilteredTrials", func(t *testing.T) {
		b := createBackend()
		defer destroyBackend(b)

		{
			err := b.CreateOrUpdateTrials(context.Background(), []*backend.TrialParams{
				{
					TrialID: "trial-1",
					Params:  generateTrialParamsWithProperties(2, 100, map[string]string{"foo": "bar", "baz": ""}),
				},
				{
					TrialID: "trial-2",
					Params:  generateTrialParamsWithProperties(4, 150, map[string]string{"baz": ""}),
				},
				{
					TrialID: "trial-3",
					Params:  generateTrialParamsWithProperties(4, 150, map[string]string{"foo": "bar2"}),
				},
			})
			assert.NoError(t, err)
		}

		{
			r, err := b.RetrieveTrials(
				context.Background(),
				backend.NewTrialFilter([]string{}, map[string]string{"baz": ""}),
				-1, -1,
			)
			assert.NoError(t, err)

			assert.Len(t, r.TrialInfos, 2)

			assert.ElementsMatch(t, r.TrialInfos, []*backend.TrialInfo{
				{
					TrialID:            "trial-1",
					State:              cogmentAPI.TrialState_UNKNOWN,
					SamplesCount:       0,
					StoredSamplesCount: 0,
				},
				{
					TrialID:            "trial-2",
					State:              cogmentAPI.TrialState_UNKNOWN,
					SamplesCount:       0,
					StoredSamplesCount: 0,
				},
			})
		}

		{
			r, err := b.RetrieveTrials(
				context.Background(),
				backend.NewTrialFilter([]string{}, map[string]string{"foo": "bar"}),
				-1, -1,
			)
			assert.NoError(t, err)

			assert.Len(t, r.TrialInfos, 1)

			assert.ElementsMatch(t, r.TrialInfos, []*backend.TrialInfo{
				{
					TrialID:            "trial-1",
					State:              cogmentAPI.TrialState_UNKNOWN,
					SamplesCount:       0,
					StoredSamplesCount: 0,
				},
			})
		}

		{
			r, err := b.RetrieveTrials(
				context.Background(),
				backend.NewTrialFilter([]string{"trial-2", "trial-3"}, map[string]string{"baz": ""}),
				-1, -1,
			)
			assert.NoError(t, err)

			assert.Len(t, r.TrialInfos, 1)

			assert.ElementsMatch(t, r.TrialInfos, []*backend.TrialInfo{
				{
					TrialID:            "trial-2",
					State:              cogmentAPI.TrialState_UNKNOWN,
					SamplesCount:       0,
					StoredSamplesCount: 0,
				},
			})
		}
	})
	t.Run("TestObserveTrials", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		t.Run("AllTrialsSingleObserver", func(t *testing.T) {
			t.Parallel()

			b := createBackend()
			defer destroyBackend(b)

			observeCtx, cancelObserve := context.WithCancel(ctx)
			defer cancelObserve()

			// Observing all new trials until `cancelObserve` is called
			observer := make(backend.TrialsInfoObserver)
			g := sync.WaitGroup{}
			g.Add(1)
			go func() {
				defer g.Done()
				err := b.ObserveTrials(
					observeCtx,
					backend.NewTrialFilter([]string{}, map[string]string{}),
					-1, -1,
					observer,
				)
				assert.ErrorIs(t, err, context.Canceled)
				close(observer)
			}()

			for i := 0; i < 10; i++ {
				trialIdx := fmt.Sprintf("trial-%d", i)

				// Create a trial
				{
					err := b.CreateOrUpdateTrials(ctx, []*backend.TrialParams{
						{
							TrialID: trialIdx,
							Params:  generateTrialParams(2, 100),
						},
					})
					assert.NoError(t, err)
				}

				// `observer` receives it
				{
					results := <-observer

					assert.Len(t, results.TrialInfos, 1)
					assert.Equal(t, trialIdx, results.TrialInfos[0].TrialID)
				}
			}

			cancelObserve()
			g.Wait()
		})

		t.Run("AllTrialsMultipleObservers", func(t *testing.T) {
			t.Parallel()

			b := createBackend()
			defer destroyBackend(b)

			observeCtx, cancelObserve := context.WithCancel(ctx)
			defer cancelObserve()

			getNextNonEmptyTrialInfoResults := func(observer backend.TrialsInfoObserver) backend.TrialsInfoResult {
				results := <-observer
				for ; len(results.TrialInfos) == 0; results = <-observer {
					continue
				}
				return results
			}

			// Observing all new trials until `cancelObserve` is called
			observer := make(backend.TrialsInfoObserver)
			g := sync.WaitGroup{}
			g.Add(1)
			go func() {
				defer g.Done()
				err := b.ObserveTrials(
					observeCtx,
					backend.NewTrialFilter([]string{}, map[string]string{}),
					-1, -1,
					observer,
				)
				assert.ErrorIs(t, err, context.Canceled)
				close(observer)
			}()

			// Observing all new trials filtered by properties until `cancelObserve` is called
			filteredObserver := make(backend.TrialsInfoObserver)
			g.Add(1)
			go func() {
				defer g.Done()
				err := b.ObserveTrials(
					observeCtx,
					backend.NewTrialFilter([]string{}, map[string]string{"key": "value"}),
					-1, -1,
					filteredObserver,
				)
				assert.ErrorIs(t, err, context.Canceled)
				close(filteredObserver)
			}()

			// Create trial1
			{
				err := b.CreateOrUpdateTrials(ctx, []*backend.TrialParams{
					{
						TrialID: "trial-1",
						Params:  generateTrialParamsWithProperties(2, 100, map[string]string{"key": "value", "key2": ""}),
					},
				})
				assert.NoError(t, err)
			}

			// `observer` receives it
			{
				results := getNextNonEmptyTrialInfoResults(observer)

				assert.Len(t, results.TrialInfos, 1)
				assert.Equal(t, "trial-1", results.TrialInfos[0].TrialID)
			}

			// `filteredObserver` receives it
			{
				results := getNextNonEmptyTrialInfoResults(filteredObserver)

				assert.Len(t, results.TrialInfos, 1)
				assert.Equal(t, "trial-1", results.TrialInfos[0].TrialID)
			}

			// Create trial2
			{
				err := b.CreateOrUpdateTrials(ctx, []*backend.TrialParams{
					{
						TrialID: "trial-2",
						Params:  generateTrialParams(2, 100),
					},
				})
				assert.NoError(t, err)
			}

			// `observer` receives it
			{
				results := getNextNonEmptyTrialInfoResults(observer)

				assert.Len(t, results.TrialInfos, 1)
				assert.Equal(t, "trial-2", results.TrialInfos[0].TrialID)
			}

			// Create trial3
			{
				err := b.CreateOrUpdateTrials(ctx, []*backend.TrialParams{
					{
						TrialID: "trial-3",
						Params:  generateTrialParamsWithProperties(2, 100, map[string]string{"key": "value"}),
					},
				})
				assert.NoError(t, err)
			}

			// `observer` receives it
			{
				results := getNextNonEmptyTrialInfoResults(observer)

				assert.Len(t, results.TrialInfos, 1)
				assert.Equal(t, "trial-3", results.TrialInfos[0].TrialID)
			}

			// `filteredObserver` receives it
			{
				results := getNextNonEmptyTrialInfoResults(filteredObserver)

				assert.Len(t, results.TrialInfos, 1)
				assert.Equal(t, "trial-3", results.TrialInfos[0].TrialID)
			}

			cancelObserve()
			g.Wait()
		})

		t.Run("LimitedTrials", func(t *testing.T) {
			t.Parallel()

			b := createBackend()
			defer destroyBackend(b)

			g := sync.WaitGroup{}

			g.Add(1)
			go func() {
				defer g.Done()
				// Waiting before actually adding the trials
				time.Sleep(100 * time.Millisecond)
				err := b.CreateOrUpdateTrials(ctx, []*backend.TrialParams{
					{
						TrialID: "trial-1",
						Params:  generateTrialParamsWithProperties(2, 100, map[string]string{"foo": "bar", "baz": ""}),
					},
					{
						TrialID: "trial-2",
						Params:  generateTrialParamsWithProperties(4, 150, map[string]string{"baz": ""}),
					},
					{
						TrialID: "trial-3",
						Params:  generateTrialParamsWithProperties(4, 150, map[string]string{"foo": "bar2"}),
					},
				})
				assert.NoError(t, err)
			}()

			g.Add(1)
			go func() {
				defer g.Done()
				observer := make(backend.TrialsInfoObserver)
				g.Add(1)
				go func() {
					defer g.Done()
					err := b.ObserveTrials(
						ctx,
						backend.NewTrialFilter([]string{}, map[string]string{"foo": "bar"}),
						0, 1,
						observer,
					)
					assert.NoError(t, err)
					close(observer)
				}()
				results := []*backend.TrialInfo{}
				for r := range observer {
					results = append(results, r.TrialInfos...)
				}

				assert.Len(t, results, 1)

				assert.Equal(t, "trial-1", results[0].TrialID)
			}()

			g.Add(1)
			go func() {
				defer g.Done()
				// Retrieve trials matching "baz"
				observer := make(backend.TrialsInfoObserver)
				g.Add(1)
				go func() {
					defer g.Done()
					err := b.ObserveTrials(
						ctx,
						backend.NewTrialFilter([]string{"trial-1", "trial-2", "trial-3"}, map[string]string{"baz": ""}),
						0, 2,
						observer,
					)
					assert.NoError(t, err)
					close(observer)
				}()
				results := []*backend.TrialInfo{}
				for r := range observer {
					results = append(results, r.TrialInfos...)
				}

				assert.Len(t, results, 2)

				assert.Equal(t, "trial-1", results[0].TrialID)
				assert.Equal(t, "trial-2", results[1].TrialID)
			}()

			g.Add(1)
			go func() {
				defer g.Done()
				// Retrieve the second trial matching "baz"
				observer := make(backend.TrialsInfoObserver)
				g.Add(1)
				go func() {
					defer g.Done()
					err := b.ObserveTrials(
						ctx,
						backend.NewTrialFilter([]string{}, map[string]string{"baz": ""}),
						1, 1,
						observer,
					)
					assert.NoError(t, err)
					close(observer)
				}()
				results := []*backend.TrialInfo{}
				for r := range observer {
					results = append(results, r.TrialInfos...)
				}

				assert.Len(t, results, 1)

				assert.Equal(t, "trial-2", results[0].TrialID)
			}()

			g.Wait()
		})
	})
	t.Run("TestDeleteTrials", func(t *testing.T) {
		b := createBackend()
		defer destroyBackend(b)

		{
			err := b.CreateOrUpdateTrials(context.Background(), []*backend.TrialParams{
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
			err := b.DeleteTrials(context.Background(), []string{"A", "C"})
			assert.NoError(t, err)
		}

		{
			r, err := b.RetrieveTrials(context.Background(), backend.NewTrialFilter([]string{}, map[string]string{}), -1, -1)
			assert.NoError(t, err)

			assert.Len(t, r.TrialInfos, 1)

			assert.Equal(t, "B", r.TrialInfos[0].TrialID)
		}

		{
			err := b.DeleteTrials(context.Background(), []string{"B", "C", "D"})
			assert.NoError(t, err)
		}

		{
			r, err := b.RetrieveTrials(context.Background(), backend.NewTrialFilter([]string{}, map[string]string{}), -1, -1)
			assert.NoError(t, err)

			assert.Len(t, r.TrialInfos, 0)
		}
	})
	t.Run("TestGetTrialParams", func(t *testing.T) {
		b := createBackend()
		defer destroyBackend(b)

		{
			err := b.CreateOrUpdateTrials(context.Background(), []*backend.TrialParams{
				{
					TrialID: "A",
					Params:  generateTrialParamsWithProperties(2, 100, map[string]string{"number": "1"}),
				},
				{
					TrialID: "B",
					Params:  generateTrialParamsWithProperties(4, 150, map[string]string{"number": "2"}),
				},
			})
			assert.NoError(t, err)
		}

		{
			r, err := b.RetrieveTrials(context.Background(), backend.NewTrialFilter([]string{}, map[string]string{}), -1, -1)
			assert.NoError(t, err)

			assert.Len(t, r.TrialInfos, 2)

			assert.ElementsMatch(t, extractTrialIDs(r.TrialInfos), []string{"A", "B"})
		}

		{
			trialsParams, err := b.GetTrialParams(context.Background(), []string{"A", "B"})
			assert.NoError(t, err)

			assert.Len(t, trialsParams, 2)

			assert.Equal(t, "A", trialsParams[0].TrialID)
			assert.Len(t, trialsParams[0].Params.Actors, 2)
			assert.Equal(t, uint32(100), trialsParams[0].Params.MaxSteps)
			assert.Equal(t, "1", trialsParams[0].Params.Properties["number"])
			assert.Equal(t, "B", trialsParams[1].TrialID)
			assert.Len(t, trialsParams[1].Params.Actors, 4)
			assert.Equal(t, uint32(150), trialsParams[1].Params.MaxSteps)
			assert.Equal(t, "2", trialsParams[1].Params.Properties["number"])
		}
	})
	t.Run("TestAddSamples", func(t *testing.T) {
		b := createBackend()
		defer destroyBackend(b)

		err := b.CreateOrUpdateTrials(context.Background(), []*backend.TrialParams{{
			TrialID: "my-trial",
			Params:  generateTrialParams(12, 100),
		}})
		assert.NoError(t, err)

		firstSample := generateSample("my-trial", 12, 512, false)
		secondSample := generateSample("my-trial", 12, 512, false)
		err = b.AddSamples(
			context.Background(),
			[]*cogmentAPI.StoredTrialSample{firstSample, secondSample, generateSample("my-trial", 12, 512, false)},
		)
		assert.NoError(t, err)

		r, err := b.RetrieveTrials(
			context.Background(),
			backend.NewTrialFilter([]string{"my-trial"}, map[string]string{}),
			-1, -1,
		)
		assert.NoError(t, err)

		assert.Len(t, r.TrialInfos, 1)

		assert.Equal(t, "my-trial", r.TrialInfos[0].TrialID)
		assert.Equal(t, cogmentAPI.TrialState_RUNNING, r.TrialInfos[0].State)
		assert.Equal(t, 3, r.TrialInfos[0].SamplesCount)
		assert.Equal(t, 3, r.TrialInfos[0].StoredSamplesCount)

		err = b.AddSamples(context.Background(), []*cogmentAPI.StoredTrialSample{
			generateSample("my-trial", 12, 512, false),
			generateSample("my-trial", 12, 512, false),
		})
		assert.NoError(t, err)

		r, err = b.RetrieveTrials(
			context.Background(),
			backend.NewTrialFilter([]string{"my-trial"}, map[string]string{}),
			-1, -1,
		)
		assert.NoError(t, err)

		assert.Len(t, r.TrialInfos, 1)

		assert.Equal(t, "my-trial", r.TrialInfos[0].TrialID)
		assert.Equal(t, cogmentAPI.TrialState_RUNNING, r.TrialInfos[0].State)
		assert.Equal(t, 5, r.TrialInfos[0].SamplesCount)
		assert.Equal(t, 5, r.TrialInfos[0].StoredSamplesCount)

		ctx, cancel := context.WithCancel(context.Background())
		observer := make(backend.TrialSampleObserver)
		go func() {
			err := b.ObserveSamples(ctx, backend.TrialSampleFilter{TrialIDs: []string{"my-trial"}}, observer)
			assert.ErrorIs(t, err, context.Canceled)
			close(observer)
		}()
		sampleResult := <-observer
		assert.Equal(t, "my-trial", sampleResult.TrialId)
		assert.Equal(t, firstSample.TickId, sampleResult.TickId)

		sampleResult = <-observer
		assert.Equal(t, "my-trial", sampleResult.TrialId)
		assert.Equal(t, secondSample.TickId, sampleResult.TickId)

		cancel()
	})
	t.Run("TestConcurrentAddAndObserveSamples", func(t *testing.T) {
		t.Parallel() // This test involves goroutines and `time.Sleep`

		b := createBackend()
		defer destroyBackend(b)

		err := b.CreateOrUpdateTrials(context.Background(), []*backend.TrialParams{{
			TrialID: "my-trial",
			Params:  generateTrialParams(12, 100),
		}})
		assert.NoError(t, err)

		samples := make([]*cogmentAPI.StoredTrialSample, 20)
		for sampleIdx := range samples {
			samples[sampleIdx] = generateSample("my-trial", 12, 512, sampleIdx == len(samples)-1)
		}

		wg := sync.WaitGroup{}
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				sampleIdx := 0
				observer := make(backend.TrialSampleObserver)
				go func() {
					err := b.ObserveSamples(
						context.Background(),
						backend.TrialSampleFilter{TrialIDs: []string{"my-trial"}},
						observer,
					)
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
			err = b.AddSamples(context.Background(), []*cogmentAPI.StoredTrialSample{sample})
			assert.NoError(t, err)
		}
		wg.Wait()
	})
	t.Run("TestObserveSamplesEmptyTrial", func(t *testing.T) {
		b := createBackend()
		defer destroyBackend(b)

		{
			err := b.CreateOrUpdateTrials(context.Background(), []*backend.TrialParams{
				{
					TrialID: "trial-1",
					Params:  generateTrialParams(2, 100),
				},
			})
			assert.NoError(t, err)
		}

		{
			observer := make(backend.TrialSampleObserver)
			ctx, cancel := context.WithCancel(context.Background())
			go func() {
				time.Sleep(1 * time.Second)
				cancel()
			}()
			err := b.ObserveSamples(ctx, backend.TrialSampleFilter{TrialIDs: []string{"trial-1"}}, observer)
			// Making sure that we don't have a `UnknownTrialError`
			assert.ErrorIs(t, err, context.Canceled)
			close(observer)
		}
	})
	t.Run("TestObserveSamplesNonExistingTrial", func(t *testing.T) {
		b := createBackend()
		defer destroyBackend(b)

		{
			observer := make(backend.TrialSampleObserver)
			ctx, cancel := context.WithCancel(context.Background())
			go func() {
				time.Sleep(1 * time.Second)
				cancel()
			}()
			err := b.ObserveSamples(ctx, backend.TrialSampleFilter{TrialIDs: []string{"trial-1"}}, observer)
			// Making sure that we have a `UnknownTrialError`
			var unknownTrialErr *backend.UnknownTrialError
			assert.ErrorAs(t, err, &unknownTrialErr)
			assert.Equal(t, "trial-1", unknownTrialErr.TrialID)
			close(observer)
		}
	})
	t.Run("TestObserveSamplesFilterActorClasses", func(t *testing.T) {
		t.Parallel() // This test involves goroutines and `time.Sleep`

		b := createBackend()
		defer destroyBackend(b)

		trial1Params := generateTrialParams(4, 100)
		trial1Params.Actors[0].ActorClass = "class1"
		trial1Params.Actors[1].ActorClass = "class2"
		trial1Params.Actors[2].ActorClass = "class1"
		trial1Params.Actors[3].ActorClass = "class1"

		err := b.CreateOrUpdateTrials(context.Background(), []*backend.TrialParams{{
			TrialID: "trial-1",
			Params:  trial1Params,
		}})
		assert.NoError(t, err)

		trial1Samples := make([]*cogmentAPI.StoredTrialSample, 1)
		for sampleIdx := range trial1Samples {
			trial1Samples[sampleIdx] = generateSample("trial-1", 4, 512, sampleIdx == len(trial1Samples)-1)
		}

		err = b.AddSamples(context.Background(), trial1Samples)
		assert.NoError(t, err)

		trial2Params := generateTrialParams(3, 36)
		trial2Params.Actors[0].ActorClass = "class2"
		trial2Params.Actors[1].ActorClass = "class2"
		trial2Params.Actors[2].ActorClass = "class1"

		err = b.CreateOrUpdateTrials(context.Background(), []*backend.TrialParams{{
			TrialID: "trial-2",
			Params:  trial2Params,
		}})
		assert.NoError(t, err)

		trial2Samples := make([]*cogmentAPI.StoredTrialSample, 1)
		for sampleIdx := range trial2Samples {
			trial2Samples[sampleIdx] = generateSample("trial-2", 3, 512, sampleIdx == len(trial2Samples)-1)
		}

		err = b.AddSamples(context.Background(), trial2Samples)
		assert.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		observer := make(backend.TrialSampleObserver)
		go func() {
			err := b.ObserveSamples(
				ctx,
				backend.TrialSampleFilter{
					TrialIDs:     []string{"trial-1", "trial-2"},
					ActorClasses: []string{"class1"},
				},
				observer,
			)
			assert.NoError(t, err)
			close(observer)
		}()

		for sampleResult := range observer {
			assert.Contains(t, []string{"trial-1", "trial-2"}, sampleResult.TrialId)
			if sampleResult.TrialId == "trial-1" {
				assert.Len(t, sampleResult.ActorSamples, 3)
			}
			if sampleResult.TrialId == "trial-2" {
				assert.Len(t, sampleResult.ActorSamples, 1)
			}
		}
	})
}
