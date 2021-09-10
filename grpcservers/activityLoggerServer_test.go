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
	"io"
	"log"
	"net"
	"testing"

	"github.com/cogment/cogment-activity-logger/backend"
	grpcapi "github.com/cogment/cogment-activity-logger/grpcapi/cogment/api"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

type testContext struct {
	backend    backend.Backend
	grpcCtx    context.Context
	client     grpcapi.ActivityLoggerClient
	connection *grpc.ClientConn
}

func createContext() (testContext, error) {
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	backend, err := backend.CreateMemoryBackend(16)
	if err != nil {
		return testContext{}, err
	}
	err = RegisterActivityLoggerServer(server, backend)
	if err != nil {
		return testContext{}, err
	}
	go func() {
		if err := server.Serve(listener); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()

	bufDialer := func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}

	grpcCtx := context.Background()

	connection, err := grpc.DialContext(grpcCtx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		return testContext{}, err
	}

	return testContext{
		backend:    backend,
		grpcCtx:    grpcCtx,
		client:     grpcapi.NewActivityLoggerClient(connection),
		connection: connection,
	}, nil
}

func (ctx *testContext) destroy() {
	ctx.connection.Close()
	ctx.backend.Destroy()
}

func TestListenToTrialSimpleSequence(t *testing.T) {
	ctx, err := createContext()
	assert.NoError(t, err)
	defer ctx.destroy()

	trialID := "mytrial"

	{
		_, err = ctx.backend.OnTrialStart(trialID, &grpcapi.TrialParams{MaxSteps: 72})
		assert.NoError(t, err)
		_, err = ctx.backend.OnTrialSample(trialID, &grpcapi.DatalogSample{UserId: "foo"})
		assert.NoError(t, err)
		_, err = ctx.backend.OnTrialSample(trialID, &grpcapi.DatalogSample{UserId: "bar"})
		assert.NoError(t, err)
		_, err = ctx.backend.OnTrialSample(trialID, &grpcapi.DatalogSample{UserId: "baz"})
		assert.NoError(t, err)
		_, err = ctx.backend.OnTrialEnd(trialID)
		assert.NoError(t, err)
	}
	{
		stream, err := ctx.client.ListenToTrial(ctx.grpcCtx, &grpcapi.ListenToTrialRequest{TrialId: trialID})
		assert.NoError(t, err)

		// message #1 should be the params
		msg, err := stream.Recv()
		assert.NoError(t, err)
		params := msg.GetTrialParams()
		assert.NotNil(t, params)
		assert.Equal(t, uint32(72), params.MaxSteps)

		// message #2 should be the first sample
		msg, err = stream.Recv()
		assert.NoError(t, err)
		sample := msg.GetSample()
		assert.NotNil(t, sample)
		assert.Equal(t, "foo", sample.UserId)

		// message #3 should be the second sample
		msg, err = stream.Recv()
		assert.NoError(t, err)
		sample = msg.GetSample()
		assert.NotNil(t, sample)
		assert.Equal(t, "bar", sample.UserId)

		// message #3 should be the third sample
		msg, err = stream.Recv()
		assert.NoError(t, err)
		sample = msg.GetSample()
		assert.NotNil(t, sample)
		assert.Equal(t, "baz", sample.UserId)

		// message #3 should be EOF
		msg, err = stream.Recv()
		assert.Equal(t, io.EOF, err)
		assert.Nil(t, msg)
	}
}
