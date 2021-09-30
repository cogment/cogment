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

	"github.com/cogment/cogment-model-registry/backend"
	"github.com/cogment/cogment-model-registry/backend/db"
	grpcapi "github.com/cogment/cogment-model-registry/grpcapi/cogment/api"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type testContext struct {
	backend    backend.Backend
	grpcCtx    context.Context
	client     grpcapi.ModelRegistryClient
	connection *grpc.ClientConn
}

var modelData = []byte(`Lorem ipsum dolor sit amet, consectetuer adipiscing elit. Aenean commodo ligula
eget dolor. Aenean massa. Cum sociis natoque penatibus et magnis dis parturient
montes, nascetur ridiculus mus. Donec quam felis, ultricies nec, pellentesque
eu, pretium quis, sem. Nulla consequat massa quis enim. Donec pede justo,
fringilla vel, aliquet nec, vulputate eget, arcu. In enim justo, rhoncus ut,
imperdiet a, venenatis vitae, justo. Nullam dictum felis eu pede mollis pretium.
Integer tincidunt. Cras dapibus. Vivamus elementum semper nisi. Aenean vulputate
eleifend tellus. Aenean leo ligula, porttitor eu, consequat vitae, eleifend ac,
enim. Aliquam lorem ante, dapibus in, viverra quis, feugiat a, tellus. Phasellus
viverra nulla ut metus varius laoreet.`)

func createContext(sentModelVersionDataChunkSize int) (testContext, error) {
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	backend, err := db.CreateBackend()
	if err != nil {
		return testContext{}, err
	}
	modelRegistryServer, err := RegisterModelRegistryServer(server, sentModelVersionDataChunkSize)
	if err != nil {
		return testContext{}, err
	}
	modelRegistryServer.SetBackend(backend)
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

func TestCreateOrUpdateModel(t *testing.T) {

	modelMetadata := make(map[string]string)
	modelMetadata["model_test1"] = "model_test1"
	modelMetadata["model_test2"] = "model_test2"
	modelMetadata["model_test3"] = "model_test3"

	ctx, err := createContext(1024 * 1024)
	assert.NoError(t, err)
	defer ctx.destroy()
	{
		rep, err := ctx.client.CreateOrUpdateModel(ctx.grpcCtx, &grpcapi.CreateOrUpdateModelRequest{ModelInfo: &grpcapi.ModelInfo{ModelId: "foo", Metadata: modelMetadata}})
		assert.NoError(t, err)
		assert.Equal(t, "foo", rep.ModelInfo.ModelId)
	}
	{
		rep, err := ctx.client.ListModelVersions(ctx.grpcCtx, &grpcapi.ListModelVersionsRequest{ModelId: "foo", PageOffset: -1, PageSize: -1})
		assert.NoError(t, err)
		assert.Len(t, rep.Versions, 0)
		assert.Equal(t, int32(0), rep.NextPageOffset)
	}
	{
		rep, err := ctx.client.CreateOrUpdateModel(ctx.grpcCtx, &grpcapi.CreateOrUpdateModelRequest{ModelInfo: &grpcapi.ModelInfo{ModelId: "bar", Metadata: modelMetadata}})
		assert.NoError(t, err)
		assert.Equal(t, "bar", rep.ModelInfo.ModelId)
	}
	{
		rep, err := ctx.client.DeleteModel(ctx.grpcCtx, &grpcapi.DeleteModelRequest{ModelId: "foo"})
		assert.NoError(t, err)
		assert.Equal(t, "foo", rep.Model.ModelId)
	}
}

func TestDeleteModel(t *testing.T) {
	modelMetadata := make(map[string]string)
	modelMetadata["model_test1"] = "model_test1"
	modelMetadata["model_test2"] = "model_test2"
	modelMetadata["model_test3"] = "model_test3"

	ctx, err := createContext(1024 * 1024)
	assert.NoError(t, err)
	defer ctx.destroy()
	{
		rep, err := ctx.client.CreateOrUpdateModel(ctx.grpcCtx, &grpcapi.CreateOrUpdateModelRequest{ModelInfo: &grpcapi.ModelInfo{ModelId: "foo", Metadata: modelMetadata}})
		assert.NoError(t, err)
		assert.Equal(t, "foo", rep.ModelInfo.ModelId)
	}
	{
		rep, err := ctx.client.CreateOrUpdateModel(ctx.grpcCtx, &grpcapi.CreateOrUpdateModelRequest{ModelInfo: &grpcapi.ModelInfo{ModelId: "bar", Metadata: modelMetadata}})
		assert.NoError(t, err)
		assert.Equal(t, "bar", rep.ModelInfo.ModelId)
	}
	{
		rep, err := ctx.client.DeleteModel(ctx.grpcCtx, &grpcapi.DeleteModelRequest{ModelId: "bar"})
		assert.NoError(t, err)
		assert.Equal(t, "bar", rep.Model.ModelId)
	}
	{
		rep, err := ctx.client.DeleteModel(ctx.grpcCtx, &grpcapi.DeleteModelRequest{ModelId: "baz"})
		assert.Error(t, err)
		assert.Equal(t, codes.NotFound, status.Code(err))
		assert.Nil(t, rep)
	}
	{
		rep, err := ctx.client.CreateOrUpdateModel(ctx.grpcCtx, &grpcapi.CreateOrUpdateModelRequest{ModelInfo: &grpcapi.ModelInfo{ModelId: "bar", Metadata: modelMetadata}})
		assert.NoError(t, err)
		assert.Equal(t, "bar", rep.ModelInfo.ModelId)
	}
}

func TestCreateOrUpdateModelVersion(t *testing.T) {

	modelMetadata := make(map[string]string)
	modelMetadata["model_test1"] = "model_test1"
	modelMetadata["model_test2"] = "model_test2"
	modelMetadata["model_test3"] = "model_test3"

	versionMetadata := make(map[string]string)
	versionMetadata["version_test1"] = "version_test1"
	versionMetadata["version_test2"] = "version_test2"
	versionMetadata["version_test3"] = "version_test3"

	ctx, err := createContext(1024 * 1024)
	assert.NoError(t, err)
	defer ctx.destroy()
	{
		rep, err := ctx.client.CreateOrUpdateModel(ctx.grpcCtx, &grpcapi.CreateOrUpdateModelRequest{ModelInfo: &grpcapi.ModelInfo{ModelId: "foo", Metadata: modelMetadata}})
		assert.NoError(t, err)
		assert.Equal(t, "foo", rep.ModelInfo.ModelId)
	}
	{
		stream, err := ctx.client.CreateOrUpdateModelVersion(ctx.grpcCtx)
		assert.NoError(t, err)
		err = stream.Send(&grpcapi.CreateOrUpdateModelVersionRequestChunk{ModelId: "foo", Archive: false, Metadata: versionMetadata, DataChunk: modelData, LastChunk: true})
		assert.NoError(t, err)
		rep, err := stream.CloseAndRecv()
		assert.NoError(t, err)
		assert.Equal(t, "foo", rep.VersionInfo.ModelId)
		assert.Equal(t, uint(1), uint(rep.VersionInfo.Number))
		assert.False(t, rep.VersionInfo.Archive)
		assert.NotZero(t, rep.VersionInfo.Hash)
		assert.NotZero(t, rep.VersionInfo.CreatedAt)
	}
	{
		stream, err := ctx.client.CreateOrUpdateModelVersion(ctx.grpcCtx)
		assert.NoError(t, err)
		err = stream.Send(&grpcapi.CreateOrUpdateModelVersionRequestChunk{ModelId: "foo", Archive: true, Metadata: versionMetadata, DataChunk: modelData[0:10], LastChunk: false})
		assert.NoError(t, err)
		err = stream.Send(&grpcapi.CreateOrUpdateModelVersionRequestChunk{DataChunk: modelData[10:30], LastChunk: false})
		assert.NoError(t, err)
		err = stream.Send(&grpcapi.CreateOrUpdateModelVersionRequestChunk{DataChunk: modelData[30:], LastChunk: true})
		assert.NoError(t, err)
		rep, err := stream.CloseAndRecv()
		assert.NoError(t, err)
		assert.Equal(t, "foo", rep.VersionInfo.ModelId)
		assert.Equal(t, uint(2), uint(rep.VersionInfo.Number))
		assert.True(t, rep.VersionInfo.Archive)
		assert.NotZero(t, rep.VersionInfo.Hash)
		assert.NotZero(t, rep.VersionInfo.CreatedAt)
	}
	{
		rep, err := ctx.client.ListModelVersions(ctx.grpcCtx, &grpcapi.ListModelVersionsRequest{ModelId: "foo", PageOffset: -1, PageSize: -1})
		assert.NoError(t, err)
		assert.Equal(t, 2, int(rep.NextPageOffset))
		assert.Len(t, rep.Versions, 2)
		assert.Equal(t, "foo", rep.Versions[0].ModelId)
		assert.Equal(t, uint(1), uint(rep.Versions[0].Number))
		assert.False(t, rep.Versions[0].Archive)
		assert.NotZero(t, rep.Versions[0].Hash)
		assert.NotZero(t, rep.Versions[0].CreatedAt)

		assert.Equal(t, "foo", rep.Versions[1].ModelId)
		assert.Equal(t, uint(2), uint(rep.Versions[1].Number))
		assert.True(t, rep.Versions[1].Archive)
		assert.Equal(t, rep.Versions[0].Hash, rep.Versions[1].Hash)
		assert.True(t, rep.Versions[1].CreatedAt.AsTime().After(rep.Versions[0].CreatedAt.AsTime()))
	}
}

func TestListModelVersions(t *testing.T) {
	modelMetadata := make(map[string]string)
	modelMetadata["model_test1"] = "model_test1"
	modelMetadata["model_test2"] = "model_test2"
	modelMetadata["model_test3"] = "model_test3"

	versionMetadata := make(map[string]string)
	versionMetadata["version_test1"] = "version_test1"
	versionMetadata["version_test2"] = "version_test2"
	versionMetadata["version_test3"] = "version_test3"

	ctx, err := createContext(1024 * 1024)
	assert.NoError(t, err)
	defer ctx.destroy()
	{
		rep, err := ctx.client.CreateOrUpdateModel(ctx.grpcCtx, &grpcapi.CreateOrUpdateModelRequest{ModelInfo: &grpcapi.ModelInfo{ModelId: "bar", Metadata: modelMetadata}})
		assert.NoError(t, err)
		assert.Equal(t, "bar", rep.ModelInfo.ModelId)
	}
	{
		for i := 1; i <= 10; i++ {
			stream, err := ctx.client.CreateOrUpdateModelVersion(ctx.grpcCtx)
			assert.NoError(t, err)
			err = stream.Send(&grpcapi.CreateOrUpdateModelVersionRequestChunk{ModelId: "bar", Metadata: versionMetadata, Archive: i%5 == 0, DataChunk: modelData, LastChunk: true})
			assert.NoError(t, err)
			_, err = stream.CloseAndRecv()
			assert.NoError(t, err)
		}
	}
	{
		rep, err := ctx.client.ListModelVersions(ctx.grpcCtx, &grpcapi.ListModelVersionsRequest{ModelId: "bar", PageOffset: 0, PageSize: 5})
		assert.NoError(t, err)

		assert.Equal(t, 5, int(rep.NextPageOffset))
		assert.Len(t, rep.Versions, 5)

		assert.Equal(t, "bar", rep.Versions[0].ModelId)
		assert.Equal(t, uint(1), uint(rep.Versions[0].Number))
		assert.False(t, rep.Versions[0].Archive)
		assert.NotZero(t, rep.Versions[0].Hash)
		assert.NotZero(t, rep.Versions[0].CreatedAt)

		assert.Equal(t, "bar", rep.Versions[4].ModelId)
		assert.Equal(t, uint(5), uint(rep.Versions[4].Number))
		assert.True(t, rep.Versions[4].Archive)
		assert.Equal(t, rep.Versions[0].Hash, rep.Versions[4].Hash)
		assert.True(t, rep.Versions[4].CreatedAt.AsTime().After(rep.Versions[0].CreatedAt.AsTime()))
	}
	{
		rep, err := ctx.client.ListModelVersions(ctx.grpcCtx, &grpcapi.ListModelVersionsRequest{ModelId: "bar", PageOffset: 7, PageSize: 5})
		assert.NoError(t, err)

		assert.Equal(t, 10, int(rep.NextPageOffset))
		assert.Len(t, rep.Versions, 3)

		assert.Equal(t, "bar", rep.Versions[0].ModelId)
		assert.Equal(t, uint(8), uint(rep.Versions[0].Number))
		assert.False(t, rep.Versions[0].Archive)
		assert.NotZero(t, rep.Versions[0].Hash)
		assert.NotZero(t, rep.Versions[0].CreatedAt)

		assert.Equal(t, "bar", rep.Versions[2].ModelId)
		assert.Equal(t, uint(10), uint(rep.Versions[2].Number))
		assert.True(t, rep.Versions[2].Archive)
		assert.Equal(t, rep.Versions[0].Hash, rep.Versions[2].Hash)
		assert.True(t, rep.Versions[2].CreatedAt.AsTime().After(rep.Versions[0].CreatedAt.AsTime()))
	}
	{
		rep, err := ctx.client.ListModelVersions(ctx.grpcCtx, &grpcapi.ListModelVersionsRequest{ModelId: "bar", PageOffset: 10, PageSize: 5})
		assert.NoError(t, err)

		assert.Equal(t, 10, int(rep.NextPageOffset))
		assert.Len(t, rep.Versions, 0)
	}
}

func TestGetModelVersionInfo(t *testing.T) {
	modelMetadata := make(map[string]string)
	modelMetadata["model_test1"] = "model_test1"
	modelMetadata["model_test2"] = "model_test2"
	modelMetadata["model_test3"] = "model_test3"

	versionMetadata := make(map[string]string)
	versionMetadata["version_test1"] = "version_test1"
	versionMetadata["version_test2"] = "version_test2"
	versionMetadata["version_test3"] = "version_test3"

	ctx, err := createContext(1024 * 1024)
	assert.NoError(t, err)
	defer ctx.destroy()
	{
		rep, err := ctx.client.CreateOrUpdateModel(ctx.grpcCtx, &grpcapi.CreateOrUpdateModelRequest{ModelInfo: &grpcapi.ModelInfo{ModelId: "bar", Metadata: modelMetadata}})
		assert.NoError(t, err)
		assert.Equal(t, "bar", rep.ModelInfo.ModelId)
	}
	{
		for i := 1; i <= 10; i++ {
			stream, err := ctx.client.CreateOrUpdateModelVersion(ctx.grpcCtx)
			assert.NoError(t, err)
			err = stream.Send(&grpcapi.CreateOrUpdateModelVersionRequestChunk{ModelId: "bar", Metadata: versionMetadata, Archive: i%5 == 0, DataChunk: modelData, LastChunk: true})
			assert.NoError(t, err)
			_, err = stream.CloseAndRecv()
			assert.NoError(t, err)
		}
	}
	{
		rep, err := ctx.client.RetrieveModelVersionInfo(ctx.grpcCtx, &grpcapi.RetrieveModelVersionInfoRequest{ModelId: "bar", Number: 1})
		assert.NoError(t, err)
		assert.Equal(t, "bar", rep.VersionInfo.ModelId)
		assert.Equal(t, uint(1), uint(rep.VersionInfo.Number))
		assert.False(t, rep.VersionInfo.Archive)
		assert.NotZero(t, rep.VersionInfo.Hash)
		assert.NotZero(t, rep.VersionInfo.CreatedAt)
	}
	{
		rep, err := ctx.client.RetrieveModelVersionInfo(ctx.grpcCtx, &grpcapi.RetrieveModelVersionInfoRequest{ModelId: "bar", Number: 5})
		assert.NoError(t, err)
		assert.Equal(t, "bar", rep.VersionInfo.ModelId)
		assert.Equal(t, uint(5), uint(rep.VersionInfo.Number))
		assert.True(t, rep.VersionInfo.Archive)
		assert.NotZero(t, rep.VersionInfo.Hash)
		assert.NotZero(t, rep.VersionInfo.CreatedAt)
	}
	{
		rep, err := ctx.client.RetrieveModelVersionInfo(ctx.grpcCtx, &grpcapi.RetrieveModelVersionInfoRequest{ModelId: "bar", Number: -1})
		assert.NoError(t, err)
		assert.Equal(t, "bar", rep.VersionInfo.ModelId)
		assert.Equal(t, uint(10), uint(rep.VersionInfo.Number))
		assert.True(t, rep.VersionInfo.Archive)
		assert.NotZero(t, rep.VersionInfo.Hash)
		assert.NotZero(t, rep.VersionInfo.CreatedAt)
	}
	{
		rep, err := ctx.client.RetrieveModelVersionInfo(ctx.grpcCtx, &grpcapi.RetrieveModelVersionInfoRequest{ModelId: "bar", Number: 28})
		assert.Error(t, err)
		assert.Equal(t, codes.NotFound, status.Code(err))
		assert.Nil(t, rep)
	}
}

func TestGetModelVersionData(t *testing.T) {
	modelMetadata := make(map[string]string)
	modelMetadata["model_test1"] = "model_test1"
	modelMetadata["model_test2"] = "model_test2"
	modelMetadata["model_test3"] = "model_test3"

	versionMetadata := make(map[string]string)
	versionMetadata["version_test1"] = "version_test1"
	versionMetadata["version_test2"] = "version_test2"
	versionMetadata["version_test3"] = "version_test3"

	ctx, err := createContext(16) // For the purpose of the test we limit the sent chunk size drastically
	assert.NoError(t, err)
	defer ctx.destroy()
	{
		rep, err := ctx.client.CreateOrUpdateModel(ctx.grpcCtx, &grpcapi.CreateOrUpdateModelRequest{ModelInfo: &grpcapi.ModelInfo{ModelId: "baz", Metadata: modelMetadata}})
		assert.NoError(t, err)
		assert.Equal(t, "baz", rep.ModelInfo.ModelId)
	}
	{
		stream, err := ctx.client.CreateOrUpdateModelVersion(ctx.grpcCtx)
		assert.NoError(t, err)
		err = stream.Send(&grpcapi.CreateOrUpdateModelVersionRequestChunk{ModelId: "baz", Archive: false, Metadata: versionMetadata, DataChunk: modelData, LastChunk: true})
		assert.NoError(t, err)
		_, err = stream.CloseAndRecv()
		assert.NoError(t, err)
	}
	{
		stream, err := ctx.client.RetrieveModelVersionData(ctx.grpcCtx, &grpcapi.RetrieveModelVersionDataRequest{ModelId: "baz", Number: -1})
		assert.NoError(t, err)
		chunks := []*grpcapi.RetrieveModelVersionDataReplyChunk{}
		data := []byte{}
		for {
			chunk, err := stream.Recv()
			if err == io.EOF {
				break
			}
			assert.NoError(t, err)
			chunks = append(chunks, chunk)
			data = append(data, chunk.DataChunk...)
		}
		for chunkIdx, chunk := range chunks {
			if chunkIdx == len(chunks)-1 {
				assert.True(t, chunk.LastChunk)
				assert.GreaterOrEqual(t, 16, len(chunk.DataChunk))
			} else {
				assert.False(t, chunk.LastChunk)
				assert.Equal(t, 16, len(chunk.DataChunk))
			}
		}
		assert.Equal(t, modelData, data)
	}
	{
		stream, err := ctx.client.RetrieveModelVersionData(ctx.grpcCtx, &grpcapi.RetrieveModelVersionDataRequest{ModelId: "baz", Number: 4})
		assert.NoError(t, err)
		chunk, err := stream.Recv()
		assert.Equal(t, codes.NotFound, status.Code(err))
		assert.Nil(t, chunk)
	}
}
