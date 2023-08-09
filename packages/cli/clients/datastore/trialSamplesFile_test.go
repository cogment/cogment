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

package datastore

import (
	"bytes"
	"io"
	"testing"
	"time"

	grpcapi "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/openlyinc/pointy"
	"github.com/stretchr/testify/assert"
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
			Implementation: "my-actor-implementation",
			Config: &grpcapi.SerializedMessage{
				Content: []byte("an actor config"),
			},
		},
		{
			Name:           "my-actor-2",
			ActorClass:     "my-actor-class-2",
			Endpoint:       "grpc://actor:9000",
			Implementation: "my-actor-implementation",
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

func TestReadWriteProtobufMessage(t *testing.T) {
	var buf bytes.Buffer
	m := &grpcapi.StoredTrialInfo{TrialId: "foo", LastState: grpcapi.TrialState_PENDING, UserId: "bar"}
	n, err := writeProtobufMessage(&buf, m)
	assert.NoError(t, err)
	assert.Greater(t, n, 0)

	reader := bytes.NewReader(buf.Bytes())
	readM := &grpcapi.StoredTrialInfo{}
	err = readProtobufMessage(reader, readM)
	assert.NoError(t, err)

	assert.Equal(t, m.TrialId, readM.TrialId)
	assert.Equal(t, m.LastState, readM.LastState)
	assert.Equal(t, m.UserId, readM.UserId)
}

func TestReadWriteTrialSamplesFile(t *testing.T) {
	var buf bytes.Buffer
	fileWriter := CreateTrialSamplesFileWriter(&buf)

	assert.Equal(t, fileWriter.Bytes, 0)

	err := fileWriter.WriteHeader(map[string]*grpcapi.TrialParams{
		"trial_1": trialParams,
		"trial_2": trialParams,
	})
	assert.NoError(t, err)
	assert.Greater(t, fileWriter.Bytes, 0)
	previousBytes := fileWriter.Bytes

	err = fileWriter.WriteSample(trialSample1)
	assert.NoError(t, err)
	assert.Greater(t, fileWriter.Bytes, previousBytes)
	previousBytes = fileWriter.Bytes

	err = fileWriter.WriteSample(trialSample1)
	assert.NoError(t, err)
	assert.Greater(t, fileWriter.Bytes, previousBytes)

	fileReader := CreateTrialSamplesFileReader(bytes.NewReader(buf.Bytes()))
	h, err := fileReader.ReadHeader()
	assert.NoError(t, err)

	assert.Contains(t, h.TrialParams, "trial_1")
	assert.Contains(t, h.TrialParams, "trial_2")

	s1, err := fileReader.ReadSample()
	assert.NoError(t, err)
	assert.Equal(t, trialSample1.TickId, s1.TickId)

	s2, err := fileReader.ReadSample()
	assert.NoError(t, err)
	assert.Equal(t, trialSample1.TickId, s2.TickId)

	_, err = fileReader.ReadSample()
	assert.Error(t, err)
	assert.Equal(t, io.EOF, err)
}
