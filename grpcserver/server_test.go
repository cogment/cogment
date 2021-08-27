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

package grpcserver

import (
	"context"
	"log"
	"net"
	"testing"

	"github.com/cogment/model-registry/backend"
	"github.com/cogment/model-registry/backend/db"
	grpcapi "github.com/cogment/model-registry/grpcapi/cogment/api"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

type testContext struct {
	backend    backend.Backend
	grpcCtx    context.Context
	client     grpcapi.ModelRegistryClient
	connection *grpc.ClientConn
}

func createContext() (testContext, error) {
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	backend, err := db.CreateBackend()
	if err != nil {
		return testContext{}, err
	}
	err = RegisterServer(server, backend)
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
		client:     grpcapi.NewModelRegistryClient(connection),
		connection: connection,
	}, nil
}

func (ctx *testContext) destroy() {
	ctx.connection.Close()
	ctx.backend.Destroy()
}

func TestCreateModel(t *testing.T) {
	ctx, err := createContext()
	assert.NoError(t, err)
	defer ctx.destroy()
	_, err = ctx.client.CreateModel(ctx.grpcCtx, &grpcapi.CreateModelRequest{ModelId: "foo"})
	assert.Error(t, err) // Not implemented yet
}
