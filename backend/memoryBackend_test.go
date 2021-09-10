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
	"testing"

	"github.com/stretchr/testify/assert"

	grpcapi "github.com/cogment/cogment-activity-logger/grpcapi/cogment/api"
)

var nextTickID uint64 // = 0

func generateTrialParams() *grpcapi.TrialParams {
	params := &grpcapi.TrialParams{}
	return params
}

func generateSample() *grpcapi.DatalogSample {
	sample := &grpcapi.DatalogSample{
		TrialData: &grpcapi.TrialData{
			TickId: nextTickID,
		},
	}
	nextTickID++
	return sample
}

func TestCreateBackend(t *testing.T) {
	b, err := CreateMemoryBackend(12)
	assert.NoError(t, err)
	defer b.Destroy()

	assert.NotNil(t, b)
}

func TestOnTrialStart(t *testing.T) {
	b, err := CreateMemoryBackend(12)
	assert.NoError(t, err)
	defer b.Destroy()

	assert.NotNil(t, b)

	trialInfo, err := b.OnTrialStart("my-trial", generateTrialParams())
	assert.NoError(t, err)
	assert.Equal(t, "my-trial", trialInfo.TrialID)
	assert.Equal(t, TrialRunning, trialInfo.Status)
	assert.Equal(t, 0, trialInfo.SamplesCount)
	assert.Equal(t, 0, trialInfo.StoredSamplesCount)

	_, err = b.OnTrialStart("my-trial", generateTrialParams())
	assert.Error(t, err)
}

func TestOnTrialEnd(t *testing.T) {
	b, err := CreateMemoryBackend(12)
	assert.NoError(t, err)
	defer b.Destroy()

	assert.NotNil(t, b)

	_, err = b.OnTrialStart("my-trial", generateTrialParams())
	assert.NoError(t, err)

	trialInfo, err := b.OnTrialEnd("my-trial")
	assert.NoError(t, err)
	assert.Equal(t, "my-trial", trialInfo.TrialID)
	assert.Equal(t, TrialEnded, trialInfo.Status)
	assert.Equal(t, 0, trialInfo.SamplesCount)
	assert.Equal(t, 0, trialInfo.StoredSamplesCount)

	_, err = b.OnTrialEnd("my-trial")
	assert.Error(t, err)

	_, err = b.OnTrialEnd("another-trial")
	assert.Error(t, err)
}

func TestGetTrialInfo(t *testing.T) {
	b, err := CreateMemoryBackend(12)
	assert.NoError(t, err)
	defer b.Destroy()

	assert.NotNil(t, b)

	_, err = b.OnTrialStart("my-trial", generateTrialParams())
	assert.NoError(t, err)

	trialInfo, err := b.GetTrialInfo("my-trial")
	assert.NoError(t, err)
	assert.Equal(t, "my-trial", trialInfo.TrialID)
	assert.Equal(t, TrialRunning, trialInfo.Status)
	assert.Equal(t, 0, trialInfo.SamplesCount)
	assert.Equal(t, 0, trialInfo.StoredSamplesCount)

	_, err = b.OnTrialEnd("my-trial")
	assert.NoError(t, err)

	trialInfo, err = b.GetTrialInfo("my-trial")
	assert.NoError(t, err)
	assert.Equal(t, "my-trial", trialInfo.TrialID)
	assert.Equal(t, TrialEnded, trialInfo.Status)
	assert.Equal(t, 0, trialInfo.SamplesCount)
	assert.Equal(t, 0, trialInfo.StoredSamplesCount)

	trialInfo, err = b.GetTrialInfo("another-trial")
	assert.NoError(t, err)
	assert.Equal(t, "another-trial", trialInfo.TrialID)
	assert.Equal(t, TrialNotStarted, trialInfo.Status)
	assert.Equal(t, 0, trialInfo.SamplesCount)
	assert.Equal(t, 0, trialInfo.StoredSamplesCount)
}

func TestOnTrialSample(t *testing.T) {
	b, err := CreateMemoryBackend(3)
	assert.NoError(t, err)
	defer b.Destroy()

	assert.NotNil(t, b)

	_, err = b.OnTrialStart("my-trial", generateTrialParams())
	assert.NoError(t, err)

	_, err = b.OnTrialSample("another-trial", generateSample())
	assert.Error(t, err)

	trialInfo, err := b.OnTrialSample("my-trial", generateSample())
	assert.NoError(t, err)
	assert.Equal(t, "my-trial", trialInfo.TrialID)
	assert.Equal(t, TrialRunning, trialInfo.Status)
	assert.Equal(t, 1, trialInfo.SamplesCount)
	assert.Equal(t, 1, trialInfo.StoredSamplesCount)

	secondSample := generateSample()
	trialInfo, err = b.OnTrialSample("my-trial", secondSample)
	assert.NoError(t, err)
	assert.Equal(t, "my-trial", trialInfo.TrialID)
	assert.Equal(t, TrialRunning, trialInfo.Status)
	assert.Equal(t, 2, trialInfo.SamplesCount)
	assert.Equal(t, 2, trialInfo.StoredSamplesCount)

	trialInfo, err = b.OnTrialSample("my-trial", generateSample())
	assert.NoError(t, err)
	assert.Equal(t, "my-trial", trialInfo.TrialID)
	assert.Equal(t, TrialRunning, trialInfo.Status)
	assert.Equal(t, 3, trialInfo.SamplesCount)
	assert.Equal(t, 3, trialInfo.StoredSamplesCount)

	trialInfo, err = b.OnTrialSample("my-trial", generateSample())
	assert.NoError(t, err)
	assert.Equal(t, "my-trial", trialInfo.TrialID)
	assert.Equal(t, TrialRunning, trialInfo.Status)
	assert.Equal(t, 4, trialInfo.SamplesCount)
	assert.Equal(t, 3, trialInfo.StoredSamplesCount)

	sampleStream := b.ConsumeTrialSamples("my-trial")
	sampleResult := <-sampleStream
	assert.NoError(t, sampleResult.Err)
	assert.Equal(t, secondSample.TrialData.TickId, sampleResult.Ok.TrialData.TickId)

	trialInfo, err = b.GetTrialInfo("my-trial")
	assert.NoError(t, err)
	assert.Equal(t, "my-trial", trialInfo.TrialID)
	assert.Equal(t, TrialRunning, trialInfo.Status)
	assert.Equal(t, 4, trialInfo.SamplesCount)
	assert.Equal(t, 1, trialInfo.StoredSamplesCount) // 4 samples produced - 1 discarded for over capacity - 1 consumed - 1 in the SampleStream goroutine
}

func TestConsumeTrialSamplesAfterTrialEnded(t *testing.T) {
	b, err := CreateMemoryBackend(20)
	assert.NoError(t, err)
	defer b.Destroy()

	assert.NotNil(t, b)

	_, err = b.OnTrialStart("my-trial", generateTrialParams())
	assert.NoError(t, err)

	for i := 0; i < 5; i++ {
		trialInfo, err := b.OnTrialSample("my-trial", generateSample())
		assert.NoError(t, err)
		assert.Equal(t, "my-trial", trialInfo.TrialID)
		assert.Equal(t, TrialRunning, trialInfo.Status)
		assert.Equal(t, i+1, trialInfo.SamplesCount)
		assert.Equal(t, i+1, trialInfo.StoredSamplesCount)
	}

	trialInfo, err := b.OnTrialEnd("my-trial")
	assert.NoError(t, err)
	assert.Equal(t, "my-trial", trialInfo.TrialID)
	assert.Equal(t, TrialEnded, trialInfo.Status)
	assert.Equal(t, 5, trialInfo.SamplesCount)
	assert.Equal(t, 5, trialInfo.StoredSamplesCount)

	sampleStream := b.ConsumeTrialSamples("my-trial")
	for i := 0; i < 5; i++ {
		sampleResult, ok := <-sampleStream
		assert.True(t, ok)
		assert.NoError(t, sampleResult.Err)
	}
	_, ok := <-sampleStream
	assert.False(t, ok)
}

func TestConsumeTrialSamplesBeforeTrialStarts(t *testing.T) {
	b, err := CreateMemoryBackend(20)
	assert.NoError(t, err)
	defer b.Destroy()

	assert.NotNil(t, b)

	go func() {
		sampleStream := b.ConsumeTrialSamples("my-trial")
		for i := 0; i < 5; i++ {
			sampleResult, ok := <-sampleStream
			assert.True(t, ok)
			assert.NoError(t, sampleResult.Err)
		}
		_, ok := <-sampleStream
		assert.False(t, ok)
	}()

	_, err = b.OnTrialStart("my-trial", generateTrialParams())
	assert.NoError(t, err)

	for i := 0; i < 5; i++ {
		trialInfo, err := b.OnTrialSample("my-trial", generateSample())
		assert.NoError(t, err)
		assert.Equal(t, "my-trial", trialInfo.TrialID)
		assert.Equal(t, TrialRunning, trialInfo.Status)
		assert.Equal(t, i+1, trialInfo.SamplesCount)
		assert.Equal(t, i+1, trialInfo.StoredSamplesCount)
	}

	trialInfo, err := b.OnTrialEnd("my-trial")
	assert.NoError(t, err)
	assert.Equal(t, "my-trial", trialInfo.TrialID)
	assert.Equal(t, TrialEnded, trialInfo.Status)
	assert.Equal(t, 5, trialInfo.SamplesCount)
	assert.Equal(t, 5, trialInfo.StoredSamplesCount)
}
