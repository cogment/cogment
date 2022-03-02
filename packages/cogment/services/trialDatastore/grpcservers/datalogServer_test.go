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

package grpcservers

import (
	"context"
	"io"
	"net"
	"testing"

	grpcapi "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/cogment/cogment/services/trialDatastore/backend"
	"github.com/cogment/cogment/services/trialDatastore/backend/memoryBackend"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

type datalogServerTestFixture struct {
	backend    backend.Backend
	ctx        context.Context
	client     grpcapi.DatalogSPClient
	connection *grpc.ClientConn
}

func createDatalogServerTestFixture() (datalogServerTestFixture, error) {
	listener := bufconn.Listen(1024 * 1024)
	server := CreateGrpcServer(false)
	backend, err := memoryBackend.CreateMemoryBackend(memoryBackend.DefaultMaxSampleSize)
	if err != nil {
		return datalogServerTestFixture{}, err
	}
	err = RegisterDatalogServer(server, backend)
	if err != nil {
		return datalogServerTestFixture{}, err
	}
	go func() {
		if err := server.Serve(listener); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()

	bufDialer := func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}

	ctx := context.Background()

	connection, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		return datalogServerTestFixture{}, err
	}

	return datalogServerTestFixture{
		backend:    backend,
		ctx:        ctx,
		client:     grpcapi.NewDatalogSPClient(connection),
		connection: connection,
	}, nil
}

func (fxt *datalogServerTestFixture) destroy() {
	fxt.connection.Close()
	fxt.backend.Destroy()
}

func TestRunTrialDatalogSimple(t *testing.T) {
	fxt, err := createDatalogServerTestFixture()
	assert.NoError(t, err)
	defer fxt.destroy()

	trialID := "mytrial"

	ctx := metadata.AppendToOutgoingContext(fxt.ctx, "trial-id", trialID)
	stream, err := fxt.client.RunTrialDatalog(ctx)
	assert.NoError(t, err)
	ack := make(chan int)
	go func() {
		index := 0
		for {
			_, err := stream.Recv()
			if err == io.EOF {
				// read done.
				close(ack)
				return
			}
			assert.NoError(t, err)
			ack <- index
			index++
		}
	}()
	{
		// Send the trial params
		err = stream.Send(&grpcapi.RunTrialDatalogInput{
			Msg: &grpcapi.RunTrialDatalogInput_TrialParams{
				TrialParams: &grpcapi.TrialParams{
					Actors: []*grpcapi.ActorParams{{
						Name: "myactor",
					}},
					MaxSteps: 72,
				},
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, 0, <-ack)
	}
	{
		// Make sure they are retrieved
		res, err := fxt.backend.RetrieveTrials(fxt.ctx, []string{trialID}, 0, -1)
		assert.NoError(t, err)
		assert.Len(t, res.TrialInfos, 1)
		assert.Equal(t, res.TrialInfos[0].TrialID, trialID)
		assert.Equal(t, res.TrialInfos[0].SamplesCount, 0)
	}
	{
		// Send a sample with only a reward
		err = stream.Send(&grpcapi.RunTrialDatalogInput{
			Msg: &grpcapi.RunTrialDatalogInput_Sample{
				Sample: &grpcapi.DatalogSample{
					Info: &grpcapi.SampleInfo{
						TickId: 0,
					},
					Rewards: []*grpcapi.Reward{{
						ReceiverName: "myactor",
						Value:        12,
						TickId:       0,
					}},
				},
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, <-ack)
	}
	{
		// Make sure it is retrieved
		samples := make(chan *grpcapi.StoredTrialSample)
		go func() {
			err := fxt.backend.ObserveSamples(fxt.ctx, backend.TrialSampleFilter{TrialIDs: []string{trialID}}, samples)
			assert.NoError(t, err)
		}()
		sample := <-samples
		assert.Equal(t, trialID, sample.TrialId)
		assert.Equal(t, uint64(0), sample.TickId)
		assert.Nil(t, sample.ActorSamples[0].Action)
		assert.Nil(t, sample.ActorSamples[0].Observation)
		assert.Equal(t, float32(12.0), *sample.ActorSamples[0].Reward)
	}
	{
		// Send a sample with an observation and an action
		err = stream.Send(&grpcapi.RunTrialDatalogInput{
			Msg: &grpcapi.RunTrialDatalogInput_Sample{
				Sample: &grpcapi.DatalogSample{
					Info: &grpcapi.SampleInfo{
						TickId: 1,
					},
					Observations: &grpcapi.ObservationSet{
						TickId:       1,
						ActorsMap:    []int32{0},
						Observations: [][]byte{[]byte("an_observation")},
					},
					Actions: []*grpcapi.Action{{
						TickId:  1,
						Content: []byte("an_action"),
					}},
				},
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, 2, <-ack)
	}
	{
		// Make sure it is retrieved
		samples := make(chan *grpcapi.StoredTrialSample)
		go func() {
			err := fxt.backend.ObserveSamples(fxt.ctx, backend.TrialSampleFilter{TrialIDs: []string{trialID}}, samples)
			assert.NoError(t, err)
		}()
		<-samples // Skip the first sample
		sample := <-samples
		assert.Equal(t, trialID, sample.TrialId)
		assert.Equal(t, uint64(1), sample.TickId)
		assert.NotNil(t, sample.ActorSamples[0].Action)
		assert.Equal(t, []byte("an_action"), sample.Payloads[*sample.ActorSamples[0].Action])
		assert.NotNil(t, sample.ActorSamples[0].Observation)
		assert.Equal(t, []byte("an_observation"), sample.Payloads[*sample.ActorSamples[0].Observation])
		assert.Nil(t, sample.ActorSamples[0].Reward)
	}
	err = stream.CloseSend()
	assert.NoError(t, err)
	<-ack
}
