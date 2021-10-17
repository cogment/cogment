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
	"container/list"
	"context"
	"errors"
	"log"
	"sync"
	"sync/atomic"

	grpcapi "github.com/cogment/cogment-trial-datastore/grpcapi/cogment/api"
	"github.com/cogment/cogment-trial-datastore/utils"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
)

type trialData struct {
	params        *grpcapi.TrialParams
	samples       utils.ObservableList
	userID        string
	evListElement *list.Element // Element corresponding to this trial in the eviction list, nil means the trial has be evicted
	deleted       bool
	samplesSize   uint32
}

func createTrialInfo(trialID string, data *trialData) *TrialInfo {
	samplesCount := data.samples.Len()
	trialInfo := &TrialInfo{
		TrialID:            trialID,
		State:              grpcapi.TrialState_UNKNOWN,
		UserID:             data.userID,
		SamplesCount:       samplesCount,
		StoredSamplesCount: 0,
	}
	if data.evListElement != nil {
		trialInfo.StoredSamplesCount = samplesCount
	}
	lastSerializedSample, hasLastItem := data.samples.Item(samplesCount - 1)
	if hasLastItem {
		lastSample := &grpcapi.TrialSample{}
		if err := proto.Unmarshal(lastSerializedSample.([]byte), lastSample); err != nil {
			log.Fatalf("Unexpected deserialization error %v", err)
		}
		trialInfo.State = lastSample.State
	}
	return trialInfo
}

type memoryBackend struct {
	trials         map[string]*trialData
	trialsEvList   *list.List // trial eviction list, front is recently used, back is least recently used
	trialsMutex    *sync.Mutex
	trialIDs       utils.ObservableList
	samplesSize    uint32
	maxSamplesSize uint32
}

var DefaultMaxSampleSize uint32 = 1024 * 1024 * 1024 // 1GB

// CreateMemoryBackend creates a Backend that will store at most "maxSamplesSize" bytes of samples
func CreateMemoryBackend(maxSamplesSize uint32) (Backend, error) {
	backend := &memoryBackend{
		trials:         make(map[string]*trialData),
		trialsMutex:    &sync.Mutex{},
		trialIDs:       utils.CreateObservableList(),
		trialsEvList:   list.New(),
		samplesSize:    0,
		maxSamplesSize: uint32(maxSamplesSize),
	}

	return backend, nil
}

// Destroy terminates the underlying storage
func (b *memoryBackend) Destroy() {
	// Nothing
}

func (b *memoryBackend) retrieveTrialDatas(trialIDs []string) ([]*trialData, error) {
	b.trialsMutex.Lock()
	defer b.trialsMutex.Unlock()
	reply := []*trialData{}
	for _, trialID := range trialIDs {
		if data, exists := b.trials[trialID]; exists {
			if data.evListElement != nil {
				b.trialsEvList.MoveToFront(data.evListElement)
			}
			reply = append(reply, data)
		} else {
			return []*trialData{}, &UnknownTrialError{TrialID: trialID}
		}
	}
	return reply, nil
}

func (b *memoryBackend) CreateOrUpdateTrials(ctx context.Context, trialsParams []*TrialParams) error {
	b.trialsMutex.Lock()
	defer b.trialsMutex.Unlock()

	for _, trialParams := range trialsParams {
		if data, exists := b.trials[trialParams.TrialID]; exists {
			if data.evListElement != nil {
				b.trialsEvList.MoveToFront(data.evListElement)
			}
			data.params = trialParams.Params
			data.userID = trialParams.UserID
		} else {
			data := &trialData{
				params:        trialParams.Params,
				userID:        trialParams.UserID,
				samples:       utils.CreateObservableList(),
				evListElement: b.trialsEvList.PushFront(struct{}{}),
				deleted:       false,
				samplesSize:   0,
			}
			b.trials[trialParams.TrialID] = data
			b.trialIDs.Append(trialParams.TrialID, false)
		}
	}
	return nil
}

func (b *memoryBackend) preprocessRetrieveTrialsArgs(filter []string, fromTrialIdx int, count int) (filter, int, int) {
	selectedTrialIDs := createFilterFromStringArray(filter)

	if fromTrialIdx < 0 {
		fromTrialIdx = 0
	}

	if count <= 0 {
		count = b.trialIDs.Len()
		if !selectedTrialIDs.selectsAll() && count > len(selectedTrialIDs) {
			count = len(selectedTrialIDs)
		}
	}

	return selectedTrialIDs, fromTrialIdx, count
}

func (b *memoryBackend) RetrieveTrials(ctx context.Context, filter []string, fromTrialIdx int, count int) (TrialsInfoResult, error) {
	selectedTrialIDs, fromTrialIdx, count := b.preprocessRetrieveTrialsArgs(filter, fromTrialIdx, count)

	result := TrialsInfoResult{
		TrialInfos:   []*TrialInfo{},
		NextTrialIdx: 0,
	}

	// List the current trials
	for trialIdx := fromTrialIdx; trialIdx < b.trialIDs.Len(); trialIdx++ {
		if len(result.TrialInfos) >= count {
			break
		}
		trialIDItem, _ := b.trialIDs.Item(trialIdx)
		trialID := trialIDItem.(string)
		data, _ := b.retrieveTrialDatas([]string{trialID})
		if data[0].deleted {
			continue
		}
		if selectedTrialIDs.selects(trialID) {
			result.TrialInfos = append(result.TrialInfos, createTrialInfo(trialID, data[0]))
			result.NextTrialIdx = trialIdx + 1
		}
	}

	return result, nil
}

func (b *memoryBackend) ObserveTrials(ctx context.Context, filter []string, fromTrialIdx int, count int, out chan<- TrialsInfoResult) error {
	selectedTrialIDs, fromTrialIdx, _ := b.preprocessRetrieveTrialsArgs(filter, fromTrialIdx, count)
	if !selectedTrialIDs.selectsAll() && (count <= 0 || count > len(selectedTrialIDs)) {
		count = len(selectedTrialIDs)
	}

	returnedResults := 0

	// Observe the trials
	trialIdx := fromTrialIdx
	observer := make(utils.ObservableListObserver)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		defer close(observer)
		return b.trialIDs.Observe(ctx, fromTrialIdx, observer)
	})
	g.Go(func() error {
		defer cancel()
		for trialIDItem := range observer {
			trialID := trialIDItem.(string)
			data, _ := b.retrieveTrialDatas([]string{trialID})
			if !data[0].deleted && selectedTrialIDs.selects(trialID) {
				unitResult := TrialsInfoResult{
					TrialInfos:   []*TrialInfo{createTrialInfo(trialID, data[0])},
					NextTrialIdx: trialIdx + 1,
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case out <- unitResult:
					returnedResults++
					if count > 0 && returnedResults >= count {
						return nil
					}
				}
			}
			trialIdx++
		}
		return nil
	})

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		// Ignoring canceled errors as we use those to stop the observation once the target count is reached
		return err
	}
	return nil
}

func (b *memoryBackend) DeleteTrials(ctx context.Context, trialIDs []string) error {
	b.trialsMutex.Lock()
	defer b.trialsMutex.Unlock()
	for _, trialID := range trialIDs {
		if data, exists := b.trials[trialID]; exists {
			if data.evListElement != nil {
				b.trialsEvList.Remove(data.evListElement)
			}
			b.trials[trialID] = &trialData{
				deleted: true,
			}
		}
	}
	return nil
}

func (b *memoryBackend) GetTrialParams(ctx context.Context, trialIDs []string) ([]*TrialParams, error) {
	trialDatas, err := b.retrieveTrialDatas(trialIDs)
	if err != nil {
		return []*TrialParams{}, err
	}
	trialParams := make([]*TrialParams, len(trialIDs))
	for idx, trialData := range trialDatas {
		trialParams[idx] = &TrialParams{TrialID: trialIDs[idx], Params: trialData.params}
	}
	return trialParams, nil
}

func (b *memoryBackend) AddSamples(ctx context.Context, samples []*grpcapi.TrialSample) error {
	trialIDs := make([]string, len(samples))
	for idx, sample := range samples {
		trialIDs[idx] = sample.TrialId
	}
	trialDatas, err := b.retrieveTrialDatas(trialIDs)
	if err != nil {
		return err
	}
	addedSamplesSize := uint32(0)
	for idx, sample := range samples {
		t := trialDatas[idx]
		serializedSample, err := proto.Marshal(sample)
		if err != nil {
			log.Fatalf("Unexpected serialization error %v", err)
		}
		sampleSize := uint32(len(serializedSample))
		addedSamplesSize += sampleSize
		t.samplesSize += sampleSize
		t.samples.Append(serializedSample, sample.State == grpcapi.TrialState_ENDED)
	}

	atomic.AddUint32(&b.samplesSize, addedSamplesSize)
	// TODO trigger eviction if needed.
	return nil
}

func (b *memoryBackend) ObserveSamples(ctx context.Context, filter TrialSampleFilter, out chan<- *grpcapi.TrialSample) error {
	trialDatas, err := b.retrieveTrialDatas(filter.TrialIDs)
	if err != nil {
		return err
	}

	actorNamesFilter := createFilterFromStringArray(filter.ActorNames)
	actorClassesFilter := createFilterFromStringArray(filter.ActorClasses)
	actorImplementationsFilter := createFilterFromStringArray(filter.ActorImplementations)
	fieldsFilter := createSampleFieldsFilter(filter.Fields)

	g, ctx := errgroup.WithContext(ctx)

	for _, td := range trialDatas {
		td := td // Create a new 'td' that gets captured by the goroutine's closure https://golang.org/doc/faq#closures_and_goroutines
		actorFilter := createActorFilter(actorNamesFilter, actorClassesFilter, actorImplementationsFilter, td.params)
		observer := make(utils.ObservableListObserver)
		g.Go(func() error {
			defer close(observer)
			return td.samples.Observe(ctx, 0, observer)
		})
		if actorFilter.selectsAll() && fieldsFilter.selectsAll() {
			// No filtering done on this trial's samples
			g.Go(func() error {
				for serializedSample := range observer {
					sample := &grpcapi.TrialSample{}
					if err := proto.Unmarshal(serializedSample.([]byte), sample); err != nil {
						log.Fatalf("Unexpected deserialization error %v", err)
					}
					out <- sample
				}
				return nil
			})
		} else {
			// Some filtering done on this trial samples
			g.Go(func() error {
				for serializedSample := range observer {
					sample := &grpcapi.TrialSample{}
					if err := proto.Unmarshal(serializedSample.([]byte), sample); err != nil {
						log.Fatalf("Unexpected deserialization error %v", err)
					}
					filteredSample := filterTrialSample(sample, actorFilter, fieldsFilter)
					out <- filteredSample
				}
				return nil
			})
		}
	}
	return g.Wait()
}
