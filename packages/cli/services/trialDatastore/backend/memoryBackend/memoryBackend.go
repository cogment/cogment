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

package memoryBackend

import (
	"container/list"
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	grpcapi "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/cogment/cogment/services/trialDatastore/backend"
	"github.com/cogment/cogment/services/trialDatastore/utils"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
)

type trialData struct {
	params            *grpcapi.TrialParams
	userID            string
	trialState        grpcapi.TrialState
	samplesCount      int
	storedSamplesSize uint32
	storedSamples     utils.ObservableList
	evListElement     *list.Element // Element for this trial in the eviction list, nil means the trial has be evicted
	deleted           bool
}

func createTrialInfo(trialID string, data *trialData) *backend.TrialInfo {
	return &backend.TrialInfo{
		TrialID:            trialID,
		State:              data.trialState,
		UserID:             data.userID,
		SamplesCount:       data.samplesCount,
		StoredSamplesCount: data.storedSamples.Len(),
	}
}

type memoryBackend struct {
	trials                map[string]*trialData
	trialsEvList          *list.List // trial eviction list, front is least recently used, back is recently used
	trialsMutex           *sync.Mutex
	trialIDs              utils.ObservableList
	samplesSize           uint32
	maxSamplesSize        uint32
	evictionWorkerTrigger chan struct{}
	evictionWorkerCancel  context.CancelFunc
}

var DefaultMaxSampleSize uint32 = 1024 * 1024 * 1024 // 1GB

// CreateMemoryBackend creates a Backend that will store at most "maxSamplesSize" bytes of samples
func CreateMemoryBackend(maxSamplesSize uint32) (backend.Backend, error) {
	evictionWorkerContext, evictionWorkerCancel := context.WithCancel(context.Background())
	backend := &memoryBackend{
		trials:                make(map[string]*trialData),
		trialsMutex:           &sync.Mutex{},
		trialIDs:              utils.CreateObservableList(),
		trialsEvList:          list.New(),
		samplesSize:           0,
		maxSamplesSize:        uint32(maxSamplesSize),
		evictionWorkerTrigger: make(chan struct{}),
		evictionWorkerCancel:  evictionWorkerCancel,
	}

	// Start the eviction worker
	go backend.evictionWorker(evictionWorkerContext)

	return backend, nil
}

// Destroy terminates the underlying storage
func (b *memoryBackend) Destroy() {
	b.evictionWorkerCancel()
}

func (b *memoryBackend) getSampleSize() uint32 {
	sampleSize := atomic.LoadUint32(&b.samplesSize)
	return sampleSize
}

func (b *memoryBackend) evictLruTrials(ctx context.Context) uint32 {
	for {
		doneChannel := make(chan bool)
		totalReclaimedSampleSize := uint32(0)
		go func() {
			if b.getSampleSize() <= b.maxSamplesSize {
				// Already done
				doneChannel <- true
				return
			}
			b.trialsMutex.Lock()
			defer b.trialsMutex.Unlock()
			front := b.trialsEvList.Front()
			if front == nil {
				// No trials
				doneChannel <- true
				return
			}
			frontTrialID := front.Value.(string)
			frontData := b.trials[frontTrialID]
			if !frontData.storedSamples.HasEnded() {
				// We don't want to evict ongoing trials if the LRU trials is ongoing there's probably nothing to evict
				doneChannel <- true
				return
			}
			reclaimedSampleSize := frontData.storedSamplesSize
			totalReclaimedSampleSize += reclaimedSampleSize
			// Subtract the trial size from the total
			atomic.AddUint32(&b.samplesSize, ^uint32(reclaimedSampleSize-1))
			frontData.storedSamples = utils.CreateObservableList()
			frontData.storedSamplesSize = 0
			frontData.evListElement = nil
			b.trialsEvList.Remove(front)
			doneChannel <- b.getSampleSize() <= b.maxSamplesSize
		}()

		select {
		case <-ctx.Done():
			// Cancelled
			return totalReclaimedSampleSize
		case done := <-doneChannel:
			if done {
				return totalReclaimedSampleSize
			}
		}
	}
}

func (b *memoryBackend) evictionWorker(ctx context.Context) {
	for {
		doneChannel := make(chan bool)
		go func() {
			// Wait until a trial eviction is requested
			<-b.evictionWorkerTrigger
			time.Sleep(50 * time.Millisecond) // Wait for further trigger for 50ms
			for len(b.evictionWorkerTrigger) > 0 {
				<-b.evictionWorkerTrigger
			}
			reclaimedSampleSize := b.evictLruTrials(ctx)
			log.Debugf(
				"Eviction worker reclaimed %dB from total sample size, now storing %dB of samples",
				reclaimedSampleSize, b.getSampleSize(),
			)
			doneChannel <- true
		}()

		select {
		case <-ctx.Done():
			// Eviction worker canceled
			return
		case <-doneChannel:
			continue
		}
	}
}

func (b *memoryBackend) retrieveTrialDatas(trialIDs []string) ([]*trialData, error) {
	b.trialsMutex.Lock()
	defer b.trialsMutex.Unlock()
	reply := []*trialData{}
	for _, trialID := range trialIDs {
		if data, exists := b.trials[trialID]; exists {
			if data.evListElement != nil {
				b.trialsEvList.MoveToBack(data.evListElement)
			}
			reply = append(reply, data)
		} else {
			return []*trialData{}, &backend.UnknownTrialError{TrialID: trialID}
		}
	}
	return reply, nil
}

func (b *memoryBackend) CreateOrUpdateTrials(ctx context.Context, trialsParams []*backend.TrialParams) error {
	b.trialsMutex.Lock()
	defer b.trialsMutex.Unlock()

	for _, trialParams := range trialsParams {
		if data, exists := b.trials[trialParams.TrialID]; exists {
			if data.evListElement != nil {
				b.trialsEvList.MoveToBack(data.evListElement)
			}
			data.params = trialParams.Params
			data.userID = trialParams.UserID
		} else {
			data := &trialData{
				params:            trialParams.Params,
				userID:            trialParams.UserID,
				trialState:        grpcapi.TrialState_UNKNOWN,
				samplesCount:      0,
				storedSamples:     utils.CreateObservableList(),
				storedSamplesSize: 0,
				evListElement:     b.trialsEvList.PushFront(trialParams.TrialID),
				deleted:           false,
			}
			b.trials[trialParams.TrialID] = data
			b.trialIDs.Append(trialParams.TrialID, false)
		}
	}
	return nil
}

func (b *memoryBackend) preprocessRetrieveTrialsArgs(
	filter backend.TrialFilter,
	fromTrialIdx int,
	count int,
) (int, int) {

	if fromTrialIdx < 0 {
		fromTrialIdx = 0
	}

	if count <= 0 {
		count = b.trialIDs.Len()
		if !filter.IDFilter.SelectsAll() && count > len(filter.IDFilter) {
			count = len(filter.IDFilter)
		}
	}

	return fromTrialIdx, count
}

func (b *memoryBackend) RetrieveTrials(
	ctx context.Context,
	filter backend.TrialFilter,
	fromTrialIdx int,
	count int,
) (backend.TrialsInfoResult, error) {
	fromTrialIdx, count = b.preprocessRetrieveTrialsArgs(filter, fromTrialIdx, count)

	result := backend.TrialsInfoResult{
		TrialInfos:   []*backend.TrialInfo{},
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
		if filter.IDFilter.Selects(trialID) && filter.PropertiesFilter.Selects(data[0].params.Properties) {
			result.TrialInfos = append(result.TrialInfos, createTrialInfo(trialID, data[0]))
			result.NextTrialIdx = trialIdx + 1
		}
	}

	return result, nil
}

func (b *memoryBackend) ObserveTrials(
	ctx context.Context,
	filter backend.TrialFilter,
	fromTrialIdx int,
	count int,
	out chan<- backend.TrialsInfoResult,
) error {
	fromTrialIdx, _ = b.preprocessRetrieveTrialsArgs(filter, fromTrialIdx, count)
	selectedCount := 0

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
			if !data[0].deleted &&
				filter.IDFilter.Selects(trialID) &&
				filter.PropertiesFilter.Selects(data[0].params.Properties) {
				unitResult := backend.TrialsInfoResult{
					TrialInfos:   []*backend.TrialInfo{createTrialInfo(trialID, data[0])},
					NextTrialIdx: trialIdx + 1,
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case out <- unitResult:
					selectedCount++
					if count > 0 && selectedCount >= count {
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
			// Subtract the trial size from the total
			atomic.AddUint32(&b.samplesSize, ^uint32(data.storedSamplesSize-1))
			b.trials[trialID] = &trialData{
				deleted: true,
			}
		}
	}
	return nil
}

func (b *memoryBackend) GetTrialParams(ctx context.Context, trialIDs []string) ([]*backend.TrialParams, error) {
	trialDatas, err := b.retrieveTrialDatas(trialIDs)
	if err != nil {
		return []*backend.TrialParams{}, err
	}
	trialParams := make([]*backend.TrialParams, len(trialIDs))
	for idx, trialData := range trialDatas {
		trialParams[idx] = &backend.TrialParams{TrialID: trialIDs[idx], Params: trialData.params}
	}
	return trialParams, nil
}

func (b *memoryBackend) AddSamples(ctx context.Context, samples []*grpcapi.StoredTrialSample) error {
	trialIDs := make([]string, len(samples))
	for idx, sample := range samples {
		trialIDs[idx] = sample.TrialId
	}
	trialDatas, err := b.retrieveTrialDatas(trialIDs)
	if err != nil {
		return err
	}
	for idx, sample := range samples {
		t := trialDatas[idx]
		serializedSample, err := proto.Marshal(sample)
		if err != nil {
			return backend.NewUnexpectedError("unable to serialize sample (%w)", err)
		}
		sampleSize := uint32(len(serializedSample))
		atomic.AddUint32(&b.samplesSize, sampleSize)
		t.storedSamplesSize += sampleSize
		t.storedSamples.Append(serializedSample, sample.State == grpcapi.TrialState_ENDED)
		t.trialState = sample.State
		t.samplesCount++
	}

	if b.getSampleSize() > b.maxSamplesSize {
		go func() {
			b.evictionWorkerTrigger <- struct{}{}
		}()
	}
	return nil
}

func (b *memoryBackend) ObserveSamples(
	ctx context.Context,
	filter backend.TrialSampleFilter,
	out chan<- *grpcapi.StoredTrialSample,
) error {
	trialDatas, err := b.retrieveTrialDatas(filter.TrialIDs)
	if err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(ctx)

	for _, td := range trialDatas {
		td := td // New 'td' that gets captured by the goroutine's closure
		appliedFilter := backend.NewAppliedTrialSampleFilter(filter, td.params)
		observer := make(utils.ObservableListObserver)
		g.Go(func() error {
			defer close(observer)
			err := td.storedSamples.Observe(ctx, 0, observer)
			return err
		})
		if appliedFilter.SelectsAll() {
			// No filtering done on this trial's samples
			g.Go(func() error {
				for serializedSample := range observer {
					sample := &grpcapi.StoredTrialSample{}
					if err := proto.Unmarshal(serializedSample.([]byte), sample); err != nil {
						return backend.NewUnexpectedError("unable to deserialize sample (%w)", err)
					}
					out <- sample
				}
				return nil
			})
		} else {
			// Some filtering done on this trial samples
			g.Go(func() error {
				for serializedSample := range observer {
					sample := &grpcapi.StoredTrialSample{}
					if err := proto.Unmarshal(serializedSample.([]byte), sample); err != nil {
						return backend.NewUnexpectedError("unable to deserialize sample (%w)", err)
					}
					filteredSample := appliedFilter.Filter(sample)
					out <- filteredSample
				}
				return nil
			})
		}
	}
	return g.Wait()
}
