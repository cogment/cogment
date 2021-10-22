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

package grpcservers

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/cogment/cogment-trial-datastore/backend"
	grpcapi "github.com/cogment/cogment-trial-datastore/grpcapi/cogment/api"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type fixture struct {
	backend    backend.Backend
	ctx        context.Context
	client     grpcapi.TrialDatastoreSPClient
	connection *grpc.ClientConn
}

func createFixture() (fixture, error) {
	listener := bufconn.Listen(1024 * 1024)
	server := CreateGrpcServer(false)
	backend, err := backend.CreateMemoryBackend(backend.DefaultMaxSampleSize)
	if err != nil {
		return fixture{}, err
	}
	err = RegisterTrialDatastoreServer(server, backend)
	if err != nil {
		return fixture{}, err
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
		return fixture{}, err
	}

	return fixture{
		backend:    backend,
		ctx:        ctx,
		client:     grpcapi.NewTrialDatastoreSPClient(connection),
		connection: connection,
	}, nil
}

func (fxt *fixture) destroy() {
	fxt.connection.Close()
	fxt.backend.Destroy()
}

func TestListenToTrialSequential(t *testing.T) {
	fxt, err := createFixture()
	assert.NoError(t, err)
	defer fxt.destroy()

	trialID := "mytrial"

	{
		err = fxt.backend.CreateOrUpdateTrials(fxt.ctx, []*backend.TrialParams{{TrialID: trialID, UserID: "foo", Params: &grpcapi.TrialParams{MaxSteps: 72}}})
		assert.NoError(t, err)
		err = fxt.backend.AddSamples(fxt.ctx, []*grpcapi.StoredTrialSample{{TrialId: trialID, UserId: "foo", State: grpcapi.TrialState_RUNNING}})
		assert.NoError(t, err)
		err = fxt.backend.AddSamples(fxt.ctx, []*grpcapi.StoredTrialSample{{TrialId: trialID, UserId: "bar", State: grpcapi.TrialState_RUNNING}})
		assert.NoError(t, err)
		err = fxt.backend.AddSamples(fxt.ctx, []*grpcapi.StoredTrialSample{{TrialId: trialID, UserId: "baz", State: grpcapi.TrialState_ENDED}})
		assert.NoError(t, err)
	}
	{
		stream, err := fxt.client.RetrieveSamples(fxt.ctx, &grpcapi.RetrieveSamplesRequest{TrialIds: []string{trialID}})
		assert.NoError(t, err)

		// message #1 should be the first sample
		msg, err := stream.Recv()
		assert.NoError(t, err)
		sample := msg.GetTrialSample()
		assert.NotNil(t, sample)
		assert.Equal(t, "foo", sample.UserId)

		// message #2 should be the second sample
		msg, err = stream.Recv()
		assert.NoError(t, err)
		sample = msg.GetTrialSample()
		assert.NotNil(t, sample)
		assert.Equal(t, "bar", sample.UserId)

		// message #3 should be the third sample
		msg, err = stream.Recv()
		assert.NoError(t, err)
		sample = msg.GetTrialSample()
		assert.NotNil(t, sample)
		assert.Equal(t, "baz", sample.UserId)

		// message #5 should be EOF
		msg, err = stream.Recv()
		assert.Equal(t, io.EOF, err)
		assert.Nil(t, msg)
	}
}

func TestListenToTrialConcurrentTrials(t *testing.T) {
	fxt, err := createFixture()
	assert.NoError(t, err)
	defer fxt.destroy()

	wg := sync.WaitGroup{}
	createAndSamplesToTrial := func(trialID string, ticks []uint64, delay time.Duration) {
		defer wg.Done()
		err := fxt.backend.CreateOrUpdateTrials(fxt.ctx, []*backend.TrialParams{{TrialID: trialID, UserID: "test", Params: &grpcapi.TrialParams{MaxSteps: 72}}})
		assert.NoError(t, err)

		for tickIdx, tickID := range ticks {
			time.Sleep(delay)
			if tickIdx < len(ticks)-1 {
				err = fxt.backend.AddSamples(fxt.ctx, []*grpcapi.StoredTrialSample{{TrialId: trialID, UserId: "test", TickId: tickID, State: grpcapi.TrialState_RUNNING}})
				assert.NoError(t, err)
			} else {
				err = fxt.backend.AddSamples(fxt.ctx, []*grpcapi.StoredTrialSample{{TrialId: trialID, UserId: "test", TickId: tickID, State: grpcapi.TrialState_ENDED}})
				assert.NoError(t, err)
			}
		}
	}
	trial1Ticks := []uint64{1, 2, 3, 4, 5, 6, 7}
	trial2Ticks := []uint64{8, 9, 10, 11, 12, 13, 14}
	wg.Add(1)
	go createAndSamplesToTrial("trial1", trial1Ticks, 10*time.Millisecond)
	wg.Add(1)
	go createAndSamplesToTrial("trial2", trial2Ticks, 5*time.Millisecond)

	retrieveSamples := func(trialIDs []string, expectedSampleTicks []uint64, delay time.Duration) {
		defer wg.Done()
		retrievedSamplesTicks := []uint64{}

		stream, err := fxt.client.RetrieveSamples(fxt.ctx, &grpcapi.RetrieveSamplesRequest{TrialIds: trialIDs})
		assert.NoError(t, err)

		for {
			time.Sleep(delay)
			msg, err := stream.Recv()

			if len(retrievedSamplesTicks) == len(expectedSampleTicks) {
				// Should have received everything
				msg, err = stream.Recv()
				assert.Equal(t, io.EOF, err)
				assert.Nil(t, msg)
				break
			}

			assert.NoError(t, err)
			sample := msg.GetTrialSample()
			assert.NotNil(t, sample)
			assert.Equal(t, "test", sample.UserId)
			assert.Contains(t, trialIDs, sample.TrialId)
			assert.Contains(t, expectedSampleTicks, sample.TickId)
			retrievedSamplesTicks = append(retrievedSamplesTicks, sample.TickId)
		}

		assert.ElementsMatch(t, expectedSampleTicks, retrievedSamplesTicks)
	}

	wg.Add(1)
	go retrieveSamples([]string{"trial1", "trial2"}, append(trial1Ticks, trial2Ticks...), 100*time.Millisecond)
	wg.Add(1)
	go retrieveSamples([]string{"trial1"}, trial1Ticks, 10*time.Millisecond)
	wg.Add(1)
	go retrieveSamples([]string{"trial2"}, trial2Ticks, 60*time.Millisecond)

	wg.Wait()
}

func createTrials(t *testing.T, fxt *fixture, trialCount int) {
	for i := 0; i < trialCount; i++ {
		ctx := metadata.AppendToOutgoingContext(fxt.ctx, "trial-id", fmt.Sprintf("trial%d", i))
		_, err := fxt.client.AddTrial(ctx, &grpcapi.AddTrialRequest{
			UserId: "test",
			TrialParams: &grpcapi.TrialParams{
				MaxSteps: uint32(10 * (i + 1)),
			},
		})
		assert.NoError(t, err)

	}
}

func TestAddAndListTrials(t *testing.T) {
	fxt, err := createFixture()
	assert.NoError(t, err)
	defer fxt.destroy()

	createTrials(t, &fxt, 10)

	// Retrieve everything in one page
	rep, err := fxt.client.RetrieveTrials(fxt.ctx, &grpcapi.RetrieveTrialsRequest{})
	assert.NoError(t, err)

	assert.Len(t, rep.TrialInfos, 10)
	for i := 0; i < 10; i++ {
		assert.Equal(t, fmt.Sprintf("trial%d", i), rep.TrialInfos[i].TrialId)
		assert.Equal(t, grpcapi.TrialState_UNKNOWN, rep.TrialInfos[i].LastState)
		assert.Equal(t, "test", rep.TrialInfos[i].UserId)
	}

	assert.Equal(t, "10", rep.NextTrialHandle)
}

func TestAddAndListTrialsPaginated(t *testing.T) {
	fxt, err := createFixture()
	assert.NoError(t, err)
	defer fxt.destroy()

	createTrials(t, &fxt, 10)

	rep, err := fxt.client.RetrieveTrials(fxt.ctx, &grpcapi.RetrieveTrialsRequest{TrialsCount: 5})
	assert.NoError(t, err)

	assert.Len(t, rep.TrialInfos, 5)
	for i := 0; i < 5; i++ {
		assert.Equal(t, fmt.Sprintf("trial%d", i), rep.TrialInfos[i].TrialId)
		assert.Equal(t, grpcapi.TrialState_UNKNOWN, rep.TrialInfos[i].LastState)
		assert.Equal(t, "test", rep.TrialInfos[i].UserId)
	}

	assert.Equal(t, "5", rep.NextTrialHandle)

	rep, err = fxt.client.RetrieveTrials(fxt.ctx, &grpcapi.RetrieveTrialsRequest{TrialsCount: 5, TrialHandle: rep.NextTrialHandle})
	assert.NoError(t, err)

	assert.Len(t, rep.TrialInfos, 5)
	for i := 0; i < 5; i++ {
		assert.Equal(t, fmt.Sprintf("trial%d", i+5), rep.TrialInfos[i].TrialId)
		assert.Equal(t, grpcapi.TrialState_UNKNOWN, rep.TrialInfos[i].LastState)
		assert.Equal(t, "test", rep.TrialInfos[i].UserId)
	}

	assert.Equal(t, "10", rep.NextTrialHandle)
}

func TestAddAndListTrialsSelectedAndPaginated(t *testing.T) {
	fxt, err := createFixture()
	assert.NoError(t, err)
	defer fxt.destroy()

	createTrials(t, &fxt, 10)

	selectedTrialIds := []string{"trial0", "trial9", "trial5", "trial2", "trial3"}
	expectedTrialIds := []string{"trial0", "trial2", "trial3", "trial5", "trial9"} // Trials should be returned according to their creation order
	rep, err := fxt.client.RetrieveTrials(fxt.ctx, &grpcapi.RetrieveTrialsRequest{TrialIds: selectedTrialIds, TrialsCount: 3})
	assert.NoError(t, err)

	assert.Len(t, rep.TrialInfos, 3)
	for i := 0; i < 3; i++ {
		assert.Equal(t, expectedTrialIds[i], rep.TrialInfos[i].TrialId)
		assert.Equal(t, grpcapi.TrialState_UNKNOWN, rep.TrialInfos[i].LastState)
		assert.Equal(t, "test", rep.TrialInfos[i].UserId)
	}

	assert.Equal(t, "4", rep.NextTrialHandle)

	rep, err = fxt.client.RetrieveTrials(fxt.ctx, &grpcapi.RetrieveTrialsRequest{TrialIds: selectedTrialIds, TrialsCount: 3, TrialHandle: rep.NextTrialHandle})
	assert.NoError(t, err)

	assert.Len(t, rep.TrialInfos, 2)
	for i := 0; i < 2; i++ {
		assert.Equal(t, expectedTrialIds[i+3], rep.TrialInfos[i].TrialId)
		assert.Equal(t, grpcapi.TrialState_UNKNOWN, rep.TrialInfos[i].LastState)
		assert.Equal(t, "test", rep.TrialInfos[i].UserId)
	}

	assert.Equal(t, "10", rep.NextTrialHandle)
}

func TestAddAndListTrialsSelectedAndPaginatedAsync(t *testing.T) {
	fxt, err := createFixture()
	assert.NoError(t, err)
	defer fxt.destroy()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		time.Sleep(100 * time.Millisecond)
		defer wg.Done()
		createTrials(t, &fxt, 10)
	}()

	selectedTrialIds := []string{"trial0", "trial9", "trial5", "trial2", "trial3"}
	expectedTrialIds := []string{"trial0", "trial2", "trial3", "trial5", "trial9"} // Trials should be returned according to their creation order
	rep, err := fxt.client.RetrieveTrials(fxt.ctx, &grpcapi.RetrieveTrialsRequest{TrialIds: selectedTrialIds, TrialsCount: 3, Timeout: 1000})
	assert.NoError(t, err)

	assert.Len(t, rep.TrialInfos, 3)
	for i := 0; i < 3; i++ {
		assert.Equal(t, expectedTrialIds[i], rep.TrialInfos[i].TrialId)
		assert.Equal(t, grpcapi.TrialState_UNKNOWN, rep.TrialInfos[i].LastState)
		assert.Equal(t, "test", rep.TrialInfos[i].UserId)
	}

	assert.Equal(t, "4", rep.NextTrialHandle)

	rep, err = fxt.client.RetrieveTrials(fxt.ctx, &grpcapi.RetrieveTrialsRequest{TrialIds: selectedTrialIds, TrialsCount: 3, TrialHandle: rep.NextTrialHandle, Timeout: 1000})
	assert.NoError(t, err)

	assert.Len(t, rep.TrialInfos, 2)
	for i := 0; i < 2; i++ {
		assert.Equal(t, expectedTrialIds[i+3], rep.TrialInfos[i].TrialId)
		assert.Equal(t, grpcapi.TrialState_UNKNOWN, rep.TrialInfos[i].LastState)
		assert.Equal(t, "test", rep.TrialInfos[i].UserId)
	}

	assert.Equal(t, "10", rep.NextTrialHandle)

	wg.Wait()
}

func TestAddSamplesSimple(t *testing.T) {
	fxt, err := createFixture()
	assert.NoError(t, err)
	defer fxt.destroy()

	createTrials(t, &fxt, 1)

	{
		ctx := metadata.AppendToOutgoingContext(fxt.ctx, "trial-id", "trial0")
		stream, err := fxt.client.AddSample(ctx)
		assert.NoError(t, err)
		err = stream.Send(&grpcapi.AddSampleRequest{
			TrialSample: &grpcapi.StoredTrialSample{TrialId: "trial0", UserId: "my_user", State: grpcapi.TrialState_RUNNING},
		})
		assert.NoError(t, err)
		_, err = stream.CloseAndRecv()
		assert.NoError(t, err)
	}

	{
		ctx := metadata.AppendToOutgoingContext(fxt.ctx, "trial-id", "trial0")
		stream, err := fxt.client.AddSample(ctx)
		assert.NoError(t, err)
		err = stream.Send(&grpcapi.AddSampleRequest{
			TrialSample: &grpcapi.StoredTrialSample{UserId: "my_user", State: grpcapi.TrialState_RUNNING},
		})
		assert.NoError(t, err)
		_, err = stream.CloseAndRecv()
		assert.NoError(t, err)
	}
}

func TestAddSamplesInconsistentTrialId(t *testing.T) {
	fxt, err := createFixture()
	assert.NoError(t, err)
	defer fxt.destroy()

	createTrials(t, &fxt, 1)

	ctx := metadata.AppendToOutgoingContext(fxt.ctx, "trial-id", "trial0")
	stream, err := fxt.client.AddSample(ctx)
	assert.NoError(t, err)
	err = stream.Send(&grpcapi.AddSampleRequest{
		TrialSample: &grpcapi.StoredTrialSample{TrialId: "foo", UserId: "my_user", State: grpcapi.TrialState_RUNNING},
	})
	assert.NoError(t, err)
	_, err = stream.CloseAndRecv()
	assert.Error(t, err)
	s, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, s.Code(), codes.InvalidArgument)
}

func TestAddAndRetrieveSamplesConcurrent(t *testing.T) {
	fxt, err := createFixture()
	assert.NoError(t, err)
	defer fxt.destroy()

	createTrials(t, &fxt, 2)

	wg := sync.WaitGroup{}

	for _, trialID := range []string{"trial0", "trial1"} {
		wg.Add(1)
		trialID := trialID
		go func() {
			defer wg.Done()

			ctx := metadata.AppendToOutgoingContext(fxt.ctx, "trial-id", trialID)
			stream, err := fxt.client.AddSample(ctx)
			assert.NoError(t, err)

			for i := 0; i < 1000; i++ {
				err = stream.Send(&grpcapi.AddSampleRequest{
					TrialSample: &grpcapi.StoredTrialSample{UserId: "my_user", State: grpcapi.TrialState_RUNNING, TickId: uint64(i)},
				})
				assert.NoError(t, err)
			}

			err = stream.Send(&grpcapi.AddSampleRequest{
				TrialSample: &grpcapi.StoredTrialSample{UserId: "my_user", State: grpcapi.TrialState_ENDED, TickId: uint64(1000)},
			})
			assert.NoError(t, err)

			_, err = stream.CloseAndRecv()
			assert.NoError(t, err)
		}()
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			stream, err := fxt.client.RetrieveSamples(fxt.ctx, &grpcapi.RetrieveSamplesRequest{TrialIds: []string{"trial0", "trial1"}})
			assert.NoError(t, err)

			nextTrial0TickID := 0
			nextTrial1TickID := 0

			for i := 0; i < 2002; i++ {
				msg, err := stream.Recv()
				assert.NoError(t, err)
				sample := msg.GetTrialSample()
				assert.NotNil(t, sample)
				assert.Equal(t, "my_user", sample.UserId)
				assert.Contains(t, []string{"trial0", "trial1"}, sample.TrialId)
				if sample.TrialId == "trial0" {
					assert.Equal(t, nextTrial0TickID, int(sample.TickId))
					nextTrial0TickID++
				} else {
					assert.Equal(t, nextTrial1TickID, int(sample.TickId))
					nextTrial1TickID++
				}
			}

			_, err = stream.Recv()
			assert.Equal(t, io.EOF, err)
		}()
	}

	wg.Wait()
}

func TestDeleteTrials(t *testing.T) {
	fxt, err := createFixture()
	assert.NoError(t, err)
	defer fxt.destroy()

	createTrials(t, &fxt, 10)
	selectedTrialIds := []string{"trial0", "trial9", "trial5", "trial2", "trial3"}

	{
		// Make sure retrieval works as expected
		expectedTrialIds := []string{"trial0", "trial2", "trial3", "trial5", "trial9"} // Trials should be returned according to their creation order
		rep, err := fxt.client.RetrieveTrials(fxt.ctx, &grpcapi.RetrieveTrialsRequest{TrialIds: selectedTrialIds, Timeout: 500})
		assert.NoError(t, err)
		assert.Equal(t, rep.NextTrialHandle, "10")
		assert.Len(t, rep.TrialInfos, len(expectedTrialIds))
		for i, trialInfo := range rep.TrialInfos {
			assert.Equal(t, expectedTrialIds[i], trialInfo.TrialId)
			assert.Equal(t, grpcapi.TrialState_UNKNOWN, trialInfo.LastState)
			assert.Equal(t, "test", trialInfo.UserId)
		}
	}

	{
		// Delete a few trials
		_, err = fxt.client.DeleteTrials(fxt.ctx, &grpcapi.DeleteTrialsRequest{TrialIds: []string{"trial2", "trial3", "trial4"}})
		assert.NoError(t, err)
	}

	t.Run("DeletedTrialsNotRetrievedWTimeout", func(t *testing.T) {
		expectedTrialIds := []string{"trial0", "trial5", "trial9"} // Trials should be returned according to their creation order
		rep, err := fxt.client.RetrieveTrials(fxt.ctx, &grpcapi.RetrieveTrialsRequest{TrialIds: selectedTrialIds, Timeout: 500})
		assert.NoError(t, err)
		assert.Equal(t, rep.NextTrialHandle, "10")
		assert.Len(t, rep.TrialInfos, len(expectedTrialIds))
		for i, trialInfo := range rep.TrialInfos {
			assert.Equal(t, expectedTrialIds[i], trialInfo.TrialId)
			assert.Equal(t, grpcapi.TrialState_UNKNOWN, trialInfo.LastState)
			assert.Equal(t, "test", trialInfo.UserId)
		}
	})

	t.Run("DeletedTrialsNotRetrievedWOTimeout", func(t *testing.T) {
		expectedTrialIds := []string{"trial0", "trial5", "trial9"} // Trials should be returned according to their creation order
		rep, err := fxt.client.RetrieveTrials(fxt.ctx, &grpcapi.RetrieveTrialsRequest{TrialIds: selectedTrialIds})
		assert.NoError(t, err)
		assert.Equal(t, rep.NextTrialHandle, "10")
		assert.Len(t, rep.TrialInfos, len(expectedTrialIds))
		for i, trialInfo := range rep.TrialInfos {
			assert.Equal(t, expectedTrialIds[i], trialInfo.TrialId)
			assert.Equal(t, grpcapi.TrialState_UNKNOWN, trialInfo.LastState)
			assert.Equal(t, "test", trialInfo.UserId)
		}
	})

	t.Run("CantAddSampleToDeletedTrial", func(t *testing.T) {
		ctx := metadata.AppendToOutgoingContext(fxt.ctx, "trial-id", "trial3")
		stream, err := fxt.client.AddSample(ctx)
		assert.NoError(t, err)
		err = stream.Send(&grpcapi.AddSampleRequest{
			TrialSample: &grpcapi.StoredTrialSample{TrialId: "foo", UserId: "my_user", State: grpcapi.TrialState_RUNNING},
		})
		assert.NoError(t, err)
		_, err = stream.CloseAndRecv()
		assert.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, s.Code(), codes.InvalidArgument)
	})
}
