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

package backend

import (
	"testing"
	"time"

	grpcapi "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/openlyinc/pointy"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

var trialParams *grpcapi.TrialParams = &grpcapi.TrialParams{
	TrialConfig: &grpcapi.SerializedMessage{
		Content: []byte("a trial config"),
	},
	MaxSteps:      12,
	MaxInactivity: 600,
	Environment: &grpcapi.EnvironmentParams{
		Endpoint:       "grpc://environment:9000",
		Implementation: "my-environment-implementation",
		Config: &grpcapi.SerializedMessage{
			Content: []byte("an environment config"),
		},
	},
	Actors: []*grpcapi.ActorParams{
		{
			Name:           "my-actor-1",
			ActorClass:     "my-actor-class-1",
			Endpoint:       "grpc://actor:9000",
			Implementation: "my-actor-implementation-1",
			Config: &grpcapi.SerializedMessage{
				Content: []byte("an actor config"),
			},
		},
		{
			Name:           "my-actor-2",
			ActorClass:     "my-actor-class-2",
			Endpoint:       "grpc://actor:9000",
			Implementation: "my-actor-implementation-2",
			Config: &grpcapi.SerializedMessage{
				Content: []byte("another actor config"),
			},
		},
	},
}

var trialSample1 *grpcapi.StoredTrialSample = &grpcapi.StoredTrialSample{
	UserId:    "my-user-id",
	TrialId:   "my-trial",
	TickId:    12,
	Timestamp: uint64(time.Now().Unix()),
	ActorSamples: []*grpcapi.StoredTrialActorSample{
		{
			Actor:       0,
			Observation: pointy.Uint32(0),
			Action:      pointy.Uint32(1),
			Reward:      pointy.Float32(0.5),
			ReceivedRewards: []*grpcapi.StoredTrialActorSampleReward{
				{
					Sender:     -1,
					Reward:     0.5,
					Confidence: 1,
					UserData:   nil,
				},
				{
					Sender:     1,
					Reward:     0.5,
					Confidence: 0.2,
					UserData:   pointy.Uint32(2),
				},
			},
		},
		{
			Actor:       1,
			Observation: pointy.Uint32(0),
			Action:      pointy.Uint32(3),
			SentRewards: []*grpcapi.StoredTrialActorSampleReward{
				{
					Receiver:   0,
					Reward:     0.5,
					Confidence: 0.2,
					UserData:   pointy.Uint32(2),
				},
			},
			ReceivedMessages: []*grpcapi.StoredTrialActorSampleMessage{
				{
					Sender:  -1,
					Payload: 4,
				},
			},
			SentMessages: []*grpcapi.StoredTrialActorSampleMessage{
				{
					Receiver: -1,
					Payload:  5,
				},
			},
		},
	},
	Payloads: [][]byte{
		[]byte("an observation"),
		[]byte("an action"),
		[]byte("a reward user data"),
		[]byte("another action"),
		[]byte("a message payload"),
		[]byte("another message payload"),
	},
}

func TestNoFilters(t *testing.T) {
	f := NewAppliedTrialSampleFilter(TrialSampleFilter{}, trialParams)
	filteredTrialSample1 := f.Filter(trialSample1)
	assert.True(t, proto.Equal(trialSample1, filteredTrialSample1))
}

func TestFieldFiltersFilterOutRewardsAndMessages(t *testing.T) {
	f := NewAppliedTrialSampleFilter(TrialSampleFilter{
		Fields: []grpcapi.StoredTrialSampleField{
			grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_ACTION,
			grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_OBSERVATION,
			grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_REWARD,
		},
	}, trialParams)

	filteredTrialSample1 := f.Filter(trialSample1)

	assert.False(t, proto.Equal(trialSample1, filteredTrialSample1))
	assert.Less(t, proto.Size(filteredTrialSample1), proto.Size(trialSample1))
	t.Logf("Full size=%vB", proto.Size(trialSample1))
	t.Logf("Filtered size=%vB", proto.Size(filteredTrialSample1))

	assert.Len(t, filteredTrialSample1.ActorSamples[0].ReceivedRewards, 0)
	assert.Len(t, filteredTrialSample1.ActorSamples[0].SentRewards, 0)
	assert.Len(t, filteredTrialSample1.ActorSamples[0].ReceivedMessages, 0)
	assert.Len(t, filteredTrialSample1.ActorSamples[0].SentMessages, 0)
	assert.Len(t, filteredTrialSample1.ActorSamples[1].ReceivedRewards, 0)
	assert.Len(t, filteredTrialSample1.ActorSamples[1].SentRewards, 0)
	assert.Len(t, filteredTrialSample1.ActorSamples[1].ReceivedMessages, 0)
	assert.Len(t, filteredTrialSample1.ActorSamples[1].SentMessages, 0)

	assert.NotEmpty(t, filteredTrialSample1.Payloads[0])
	assert.NotEmpty(t, filteredTrialSample1.Payloads[1])
	assert.Empty(t, filteredTrialSample1.Payloads[2])
	assert.NotEmpty(t, filteredTrialSample1.Payloads[3])
	assert.Empty(t, filteredTrialSample1.Payloads[4])
	assert.Empty(t, filteredTrialSample1.Payloads[5])

	twiceFilteredTrialSample1 := f.Filter(filteredTrialSample1)

	assert.True(t, proto.Equal(twiceFilteredTrialSample1, filteredTrialSample1))
}

func TestFieldFiltersFilterOutReward(t *testing.T) {
	f := NewAppliedTrialSampleFilter(TrialSampleFilter{
		Fields: []grpcapi.StoredTrialSampleField{
			grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_ACTION,
			grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_OBSERVATION,
			grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_RECEIVED_REWARDS,
			grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_SENT_REWARDS,
			grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_RECEIVED_MESSAGES,
			grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_SENT_MESSAGES,
		},
	}, trialParams)

	filteredTrialSample1 := f.Filter(trialSample1)

	assert.False(t, proto.Equal(trialSample1, filteredTrialSample1))
	assert.Less(t, proto.Size(filteredTrialSample1), proto.Size(trialSample1))
	t.Logf("Full size=%vB", proto.Size(trialSample1))
	t.Logf("Filtered size=%vB", proto.Size(filteredTrialSample1))

	assert.Nil(t, filteredTrialSample1.ActorSamples[0].Reward)
	assert.Nil(t, filteredTrialSample1.ActorSamples[1].Reward)
}

func TestActorNameFilters(t *testing.T) {
	f := NewAppliedTrialSampleFilter(TrialSampleFilter{
		ActorNames: []string{"my-actor-2"},
		Fields: []grpcapi.StoredTrialSampleField{
			grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_ACTION,
			grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_OBSERVATION,
			grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_RECEIVED_REWARDS,
			grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_SENT_REWARDS,
			grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_RECEIVED_MESSAGES,
			grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_SENT_MESSAGES,
		},
	}, trialParams)

	filteredTrialSample1 := f.Filter(trialSample1)

	assert.False(t, proto.Equal(trialSample1, filteredTrialSample1))
	assert.Less(t, proto.Size(filteredTrialSample1), proto.Size(trialSample1))

	assert.Len(t, filteredTrialSample1.ActorSamples, 1)

	assert.NotEmpty(t, filteredTrialSample1.Payloads[0])
	assert.Empty(t, filteredTrialSample1.Payloads[1])
	assert.NotEmpty(t, filteredTrialSample1.Payloads[2])
	assert.NotEmpty(t, filteredTrialSample1.Payloads[3])
	assert.NotEmpty(t, filteredTrialSample1.Payloads[4])
	assert.NotEmpty(t, filteredTrialSample1.Payloads[5])

	twiceFilteredTrialSample1 := f.Filter(filteredTrialSample1)

	assert.True(t, proto.Equal(twiceFilteredTrialSample1, filteredTrialSample1))
}

func TestActorClassesFilters(t *testing.T) {
	f := NewAppliedTrialSampleFilter(TrialSampleFilter{
		ActorClasses: []string{"my-actor-class-1"},
		Fields: []grpcapi.StoredTrialSampleField{
			grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_ACTION,
			grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_OBSERVATION,
			grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_RECEIVED_REWARDS,
			grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_SENT_REWARDS,
			grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_RECEIVED_MESSAGES,
			grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_SENT_MESSAGES,
		},
	}, trialParams)

	filteredTrialSample1 := f.Filter(trialSample1)

	assert.False(t, proto.Equal(trialSample1, filteredTrialSample1))
	assert.Less(t, proto.Size(filteredTrialSample1), proto.Size(trialSample1))

	assert.Len(t, filteredTrialSample1.ActorSamples, 1)

	assert.NotEmpty(t, filteredTrialSample1.Payloads[0])
	assert.NotEmpty(t, filteredTrialSample1.Payloads[1])
	assert.NotEmpty(t, filteredTrialSample1.Payloads[2])
	assert.Empty(t, filteredTrialSample1.Payloads[3])
	assert.Empty(t, filteredTrialSample1.Payloads[4])
	assert.Empty(t, filteredTrialSample1.Payloads[5])

	twiceFilteredTrialSample1 := f.Filter(filteredTrialSample1)

	assert.True(t, proto.Equal(twiceFilteredTrialSample1, filteredTrialSample1))
}

func TestActorImplementationsFilters(t *testing.T) {
	f := NewAppliedTrialSampleFilter(TrialSampleFilter{
		ActorImplementations: []string{"my-actor-implementation-2"},
		Fields: []grpcapi.StoredTrialSampleField{
			grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_ACTION,
			grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_OBSERVATION,
			grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_RECEIVED_REWARDS,
			grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_SENT_REWARDS,
			grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_RECEIVED_MESSAGES,
			grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_SENT_MESSAGES,
		},
	}, trialParams)

	filteredTrialSample1 := f.Filter(trialSample1)

	assert.False(t, proto.Equal(trialSample1, filteredTrialSample1))
	assert.Less(t, proto.Size(filteredTrialSample1), proto.Size(trialSample1))

	assert.Len(t, filteredTrialSample1.ActorSamples, 1)

	assert.NotEmpty(t, filteredTrialSample1.Payloads[0])
	assert.Empty(t, filteredTrialSample1.Payloads[1])
	assert.NotEmpty(t, filteredTrialSample1.Payloads[2])
	assert.NotEmpty(t, filteredTrialSample1.Payloads[3])
	assert.NotEmpty(t, filteredTrialSample1.Payloads[4])
	assert.NotEmpty(t, filteredTrialSample1.Payloads[5])

	twiceFilteredTrialSample1 := f.Filter(filteredTrialSample1)

	assert.True(t, proto.Equal(twiceFilteredTrialSample1, filteredTrialSample1))
}

func TestActorImplementationsAndClassesFilters(t *testing.T) {
	f := NewAppliedTrialSampleFilter(TrialSampleFilter{
		ActorClasses:         []string{"my-actor-class-2"},
		ActorImplementations: []string{"my-actor-implementation-1"},
		Fields: []grpcapi.StoredTrialSampleField{
			grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_ACTION,
			grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_OBSERVATION,
			grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_RECEIVED_REWARDS,
			grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_SENT_REWARDS,
			grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_RECEIVED_MESSAGES,
			grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_SENT_MESSAGES,
		},
	}, trialParams)

	filteredTrialSample1 := f.Filter(trialSample1)

	assert.False(t, proto.Equal(trialSample1, filteredTrialSample1))
	assert.Less(t, proto.Size(filteredTrialSample1), proto.Size(trialSample1))

	assert.Len(t, filteredTrialSample1.ActorSamples, 0)

	assert.Empty(t, filteredTrialSample1.Payloads[0])
	assert.Empty(t, filteredTrialSample1.Payloads[1])
	assert.Empty(t, filteredTrialSample1.Payloads[2])
	assert.Empty(t, filteredTrialSample1.Payloads[3])
	assert.Empty(t, filteredTrialSample1.Payloads[4])
	assert.Empty(t, filteredTrialSample1.Payloads[5])

	twiceFilteredTrialSample1 := f.Filter(filteredTrialSample1)

	assert.True(t, proto.Equal(twiceFilteredTrialSample1, filteredTrialSample1))
}
