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
	"time"

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
	client     grpcapi.ModelRegistrySPClient
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
		client:     grpcapi.NewModelRegistrySPClient(connection),
		connection: connection,
	}, nil
}

func (ctx *testContext) destroy() {
	ctx.connection.Close()
	ctx.backend.Destroy()
}

func TestCreateOrUpdateModel(t *testing.T) {

	modelUserData := make(map[string]string)
	modelUserData["model_test1"] = "model_test1"
	modelUserData["model_test2"] = "model_test2"
	modelUserData["model_test3"] = "model_test3"

	ctx, err := createContext(1024 * 1024)
	assert.NoError(t, err)
	defer ctx.destroy()
	{
		_, err := ctx.client.CreateOrUpdateModel(ctx.grpcCtx, &grpcapi.CreateOrUpdateModelRequest{ModelInfo: &grpcapi.ModelInfo{ModelId: "foo", UserData: modelUserData}})
		assert.NoError(t, err)
	}
	{
		rep, err := ctx.client.RetrieveVersionInfos(ctx.grpcCtx, &grpcapi.RetrieveVersionInfosRequest{ModelId: "foo"})
		assert.NoError(t, err)
		assert.Len(t, rep.VersionInfos, 0)
		assert.Equal(t, "0", rep.NextVersionHandle)
	}
	{
		_, err := ctx.client.CreateOrUpdateModel(ctx.grpcCtx, &grpcapi.CreateOrUpdateModelRequest{ModelInfo: &grpcapi.ModelInfo{ModelId: "bar", UserData: modelUserData}})
		assert.NoError(t, err)
	}
	{
		_, err := ctx.client.DeleteModel(ctx.grpcCtx, &grpcapi.DeleteModelRequest{ModelId: "foo"})
		assert.NoError(t, err)
	}
}

func TestDeleteModel(t *testing.T) {
	modelUserData := make(map[string]string)
	modelUserData["model_test1"] = "model_test1"
	modelUserData["model_test2"] = "model_test2"
	modelUserData["model_test3"] = "model_test3"

	ctx, err := createContext(1024 * 1024)
	assert.NoError(t, err)
	defer ctx.destroy()
	{
		_, err := ctx.client.CreateOrUpdateModel(ctx.grpcCtx, &grpcapi.CreateOrUpdateModelRequest{ModelInfo: &grpcapi.ModelInfo{ModelId: "foo", UserData: modelUserData}})
		assert.NoError(t, err)
	}
	{
		_, err := ctx.client.CreateOrUpdateModel(ctx.grpcCtx, &grpcapi.CreateOrUpdateModelRequest{ModelInfo: &grpcapi.ModelInfo{ModelId: "bar", UserData: modelUserData}})
		assert.NoError(t, err)
	}
	{
		_, err := ctx.client.DeleteModel(ctx.grpcCtx, &grpcapi.DeleteModelRequest{ModelId: "bar"})
		assert.NoError(t, err)
	}
	{
		rep, err := ctx.client.DeleteModel(ctx.grpcCtx, &grpcapi.DeleteModelRequest{ModelId: "baz"})
		assert.Error(t, err)
		assert.Equal(t, codes.NotFound, status.Code(err))
		assert.Nil(t, rep)
	}
	{
		_, err := ctx.client.CreateOrUpdateModel(ctx.grpcCtx, &grpcapi.CreateOrUpdateModelRequest{ModelInfo: &grpcapi.ModelInfo{ModelId: "bar", UserData: modelUserData}})
		assert.NoError(t, err)
	}
}

func TestCreateVersion(t *testing.T) {

	modelUserData := make(map[string]string)
	modelUserData["model_test1"] = "model_test1"
	modelUserData["model_test2"] = "model_test2"
	modelUserData["model_test3"] = "model_test3"

	versionUserData := make(map[string]string)
	versionUserData["version_test1"] = "version_test1"
	versionUserData["version_test2"] = "version_test2"
	versionUserData["version_test3"] = "version_test3"

	ctx, err := createContext(1024 * 1024)
	assert.NoError(t, err)
	defer ctx.destroy()
	{
		_, err := ctx.client.CreateOrUpdateModel(ctx.grpcCtx, &grpcapi.CreateOrUpdateModelRequest{ModelInfo: &grpcapi.ModelInfo{ModelId: "foo", UserData: modelUserData}})
		assert.NoError(t, err)
	}
	{
		stream, err := ctx.client.CreateVersion(ctx.grpcCtx)
		assert.NoError(t, err)
		err = stream.Send(&grpcapi.CreateVersionRequestChunk{
			Msg: &grpcapi.CreateVersionRequestChunk_Header_{
				Header: &grpcapi.CreateVersionRequestChunk_Header{
					VersionInfo: &grpcapi.ModelVersionInfo{
						ModelId:           "foo",
						CreationTimestamp: nsTimestampFromTime(time.Now()),
						Archived:          false,
						DataHash:          backend.ComputeSHA256Hash(modelData),
						DataSize:          uint64(len(modelData)),
						UserData:          versionUserData,
					},
				},
			},
		})
		assert.NoError(t, err)
		err = stream.Send(&grpcapi.CreateVersionRequestChunk{
			Msg: &grpcapi.CreateVersionRequestChunk_Body_{
				Body: &grpcapi.CreateVersionRequestChunk_Body{
					DataChunk: modelData,
				},
			},
		})
		assert.NoError(t, err)
		rep, err := stream.CloseAndRecv()
		assert.NoError(t, err)
		assert.Equal(t, "foo", rep.VersionInfo.ModelId)
		assert.Equal(t, uint(1), uint(rep.VersionInfo.VersionNumber))
		assert.False(t, rep.VersionInfo.Archived)
		assert.Equal(t, backend.ComputeSHA256Hash(modelData), rep.VersionInfo.DataHash)
		assert.NotZero(t, uint64(len(modelData)), rep.VersionInfo.DataHash)
		assert.Equal(t, versionUserData, rep.VersionInfo.UserData)
	}
	{
		stream, err := ctx.client.CreateVersion(ctx.grpcCtx)
		assert.NoError(t, err)
		err = stream.Send(&grpcapi.CreateVersionRequestChunk{
			Msg: &grpcapi.CreateVersionRequestChunk_Header_{
				Header: &grpcapi.CreateVersionRequestChunk_Header{
					VersionInfo: &grpcapi.ModelVersionInfo{
						ModelId:           "foo",
						CreationTimestamp: nsTimestampFromTime(time.Now()),
						Archived:          true,
						DataHash:          backend.ComputeSHA256Hash(modelData),
						DataSize:          uint64(len(modelData)),
						UserData:          versionUserData,
					},
				},
			},
		})
		assert.NoError(t, err)
		err = stream.Send(&grpcapi.CreateVersionRequestChunk{
			Msg: &grpcapi.CreateVersionRequestChunk_Body_{
				Body: &grpcapi.CreateVersionRequestChunk_Body{
					DataChunk: modelData[:20],
				},
			},
		})
		assert.NoError(t, err)
		err = stream.Send(&grpcapi.CreateVersionRequestChunk{
			Msg: &grpcapi.CreateVersionRequestChunk_Body_{
				Body: &grpcapi.CreateVersionRequestChunk_Body{
					DataChunk: modelData[20:],
				},
			},
		})
		assert.NoError(t, err)
		rep, err := stream.CloseAndRecv()
		assert.NoError(t, err)
		assert.Equal(t, "foo", rep.VersionInfo.ModelId)
		assert.Equal(t, 2, int(rep.VersionInfo.VersionNumber))
		assert.True(t, rep.VersionInfo.Archived)
		assert.Equal(t, backend.ComputeSHA256Hash(modelData), rep.VersionInfo.DataHash)
		assert.NotZero(t, uint64(len(modelData)), rep.VersionInfo.DataHash)
		assert.Equal(t, versionUserData, rep.VersionInfo.UserData)
	}
	{
		rep, err := ctx.client.RetrieveVersionInfos(ctx.grpcCtx, &grpcapi.RetrieveVersionInfosRequest{ModelId: "foo"})
		assert.NoError(t, err)
		assert.Equal(t, "2", rep.NextVersionHandle)
		assert.Len(t, rep.VersionInfos, 2)
		assert.Equal(t, "foo", rep.VersionInfos[0].ModelId)
		assert.Equal(t, 1, int(rep.VersionInfos[0].VersionNumber))
		assert.False(t, rep.VersionInfos[0].Archived)
		assert.NotZero(t, rep.VersionInfos[0].DataHash)

		assert.Equal(t, "foo", rep.VersionInfos[1].ModelId)
		assert.Equal(t, 2, int(rep.VersionInfos[1].VersionNumber))
		assert.True(t, rep.VersionInfos[1].Archived)
		assert.Equal(t, rep.VersionInfos[0].DataHash, rep.VersionInfos[1].DataHash)
	}
}

func TestRetrieveVersionInfosAll(t *testing.T) {
	modelUserData := make(map[string]string)
	modelUserData["model_test1"] = "model_test1"
	modelUserData["model_test2"] = "model_test2"
	modelUserData["model_test3"] = "model_test3"

	versionUserData := make(map[string]string)
	versionUserData["version_test1"] = "version_test1"
	versionUserData["version_test2"] = "version_test2"
	versionUserData["version_test3"] = "version_test3"

	ctx, err := createContext(1024 * 1024)
	assert.NoError(t, err)
	defer ctx.destroy()
	{
		_, err := ctx.client.CreateOrUpdateModel(ctx.grpcCtx, &grpcapi.CreateOrUpdateModelRequest{ModelInfo: &grpcapi.ModelInfo{ModelId: "bar", UserData: modelUserData}})
		assert.NoError(t, err)
	}
	{
		for i := 1; i <= 10; i++ {
			stream, err := ctx.client.CreateVersion(ctx.grpcCtx)
			assert.NoError(t, err)
			err = stream.Send(&grpcapi.CreateVersionRequestChunk{
				Msg: &grpcapi.CreateVersionRequestChunk_Header_{
					Header: &grpcapi.CreateVersionRequestChunk_Header{
						VersionInfo: &grpcapi.ModelVersionInfo{
							ModelId:           "bar",
							CreationTimestamp: nsTimestampFromTime(time.Now()),
							Archived:          i%5 == 0,
							DataHash:          backend.ComputeSHA256Hash(modelData),
							DataSize:          uint64(len(modelData)),
							UserData:          versionUserData,
						},
					},
				},
			})
			assert.NoError(t, err)
			err = stream.Send(&grpcapi.CreateVersionRequestChunk{
				Msg: &grpcapi.CreateVersionRequestChunk_Body_{Body: &grpcapi.CreateVersionRequestChunk_Body{
					DataChunk: modelData,
				}}})
			assert.NoError(t, err)
			_, err = stream.CloseAndRecv()
			assert.NoError(t, err)
		}
	}
	{
		rep, err := ctx.client.RetrieveVersionInfos(ctx.grpcCtx, &grpcapi.RetrieveVersionInfosRequest{ModelId: "bar", VersionsCount: 5})
		assert.NoError(t, err)

		assert.Equal(t, "5", rep.NextVersionHandle)
		assert.Len(t, rep.VersionInfos, 5)

		assert.Equal(t, "bar", rep.VersionInfos[0].ModelId)
		assert.Equal(t, uint(1), uint(rep.VersionInfos[0].VersionNumber))
		assert.False(t, rep.VersionInfos[0].Archived)
		assert.NotZero(t, rep.VersionInfos[0].DataHash)
		assert.NotZero(t, rep.VersionInfos[0].CreationTimestamp)

		assert.Equal(t, "bar", rep.VersionInfos[4].ModelId)
		assert.Equal(t, uint(5), uint(rep.VersionInfos[4].VersionNumber))
		assert.True(t, rep.VersionInfos[4].Archived)
		assert.Equal(t, rep.VersionInfos[0].DataHash, rep.VersionInfos[4].DataHash)
		assert.GreaterOrEqual(t, rep.VersionInfos[4].CreationTimestamp, rep.VersionInfos[0].CreationTimestamp)
	}
	{
		rep, err := ctx.client.RetrieveVersionInfos(ctx.grpcCtx, &grpcapi.RetrieveVersionInfosRequest{ModelId: "bar", VersionsCount: 5, VersionHandle: "7"})
		assert.NoError(t, err)

		assert.Equal(t, "10", rep.NextVersionHandle)
		assert.Len(t, rep.VersionInfos, 3)

		assert.Equal(t, "bar", rep.VersionInfos[0].ModelId)
		assert.Equal(t, uint(8), uint(rep.VersionInfos[0].VersionNumber))
		assert.False(t, rep.VersionInfos[0].Archived)
		assert.NotZero(t, rep.VersionInfos[0].DataHash)
		assert.NotZero(t, rep.VersionInfos[0].CreationTimestamp)

		assert.Equal(t, "bar", rep.VersionInfos[2].ModelId)
		assert.Equal(t, uint(10), uint(rep.VersionInfos[2].VersionNumber))
		assert.True(t, rep.VersionInfos[2].Archived)
		assert.Equal(t, rep.VersionInfos[0].DataHash, rep.VersionInfos[2].DataHash)
		assert.GreaterOrEqual(t, rep.VersionInfos[2].CreationTimestamp, rep.VersionInfos[0].CreationTimestamp)
	}
	{
		rep, err := ctx.client.RetrieveVersionInfos(ctx.grpcCtx, &grpcapi.RetrieveVersionInfosRequest{ModelId: "bar", VersionsCount: 5, VersionHandle: "10"})
		assert.NoError(t, err)

		assert.Equal(t, "10", rep.NextVersionHandle)
		assert.Len(t, rep.VersionInfos, 0)
	}
}

func TestRetrieveVersionInfosSome(t *testing.T) {
	modelUserData := make(map[string]string)
	modelUserData["model_test1"] = "model_test1"
	modelUserData["model_test2"] = "model_test2"
	modelUserData["model_test3"] = "model_test3"

	versionUserData := make(map[string]string)
	versionUserData["version_test1"] = "version_test1"
	versionUserData["version_test2"] = "version_test2"
	versionUserData["version_test3"] = "version_test3"

	ctx, err := createContext(1024 * 1024)
	assert.NoError(t, err)
	defer ctx.destroy()
	{
		_, err := ctx.client.CreateOrUpdateModel(ctx.grpcCtx, &grpcapi.CreateOrUpdateModelRequest{ModelInfo: &grpcapi.ModelInfo{ModelId: "bar", UserData: modelUserData}})
		assert.NoError(t, err)
	}
	{
		for i := 1; i <= 10; i++ {
			stream, err := ctx.client.CreateVersion(ctx.grpcCtx)
			assert.NoError(t, err)
			err = stream.Send(&grpcapi.CreateVersionRequestChunk{
				Msg: &grpcapi.CreateVersionRequestChunk_Header_{
					Header: &grpcapi.CreateVersionRequestChunk_Header{
						VersionInfo: &grpcapi.ModelVersionInfo{
							ModelId:           "bar",
							CreationTimestamp: nsTimestampFromTime(time.Now()),
							Archived:          i%5 == 0,
							DataHash:          backend.ComputeSHA256Hash(modelData),
							DataSize:          uint64(len(modelData)),
							UserData:          versionUserData,
						},
					},
				},
			})
			assert.NoError(t, err)
			err = stream.Send(&grpcapi.CreateVersionRequestChunk{
				Msg: &grpcapi.CreateVersionRequestChunk_Body_{
					Body: &grpcapi.CreateVersionRequestChunk_Body{
						DataChunk: modelData,
					},
				},
			})
			assert.NoError(t, err)
			_, err = stream.CloseAndRecv()
			assert.NoError(t, err)
		}
	}
	{
		rep, err := ctx.client.RetrieveVersionInfos(ctx.grpcCtx, &grpcapi.RetrieveVersionInfosRequest{ModelId: "bar", VersionNumbers: []int32{1}})
		assert.NoError(t, err)
		assert.Len(t, rep.VersionInfos, 1)
		assert.Equal(t, "1", rep.NextVersionHandle)

		assert.Equal(t, "bar", rep.VersionInfos[0].ModelId)
		assert.Equal(t, 1, int(rep.VersionInfos[0].VersionNumber))
		assert.False(t, rep.VersionInfos[0].Archived)
		assert.NotZero(t, rep.VersionInfos[0].DataHash)
		assert.NotZero(t, rep.VersionInfos[0].CreationTimestamp)
	}
	{
		rep, err := ctx.client.RetrieveVersionInfos(ctx.grpcCtx, &grpcapi.RetrieveVersionInfosRequest{ModelId: "bar", VersionNumbers: []int32{5}})
		assert.NoError(t, err)
		assert.Len(t, rep.VersionInfos, 1)
		assert.Equal(t, "1", rep.NextVersionHandle)

		assert.Equal(t, "bar", rep.VersionInfos[0].ModelId)
		assert.Equal(t, 5, int(rep.VersionInfos[0].VersionNumber))
		assert.True(t, rep.VersionInfos[0].Archived)
		assert.NotZero(t, rep.VersionInfos[0].DataHash)
		assert.NotZero(t, rep.VersionInfos[0].CreationTimestamp)
	}
	{
		rep, err := ctx.client.RetrieveVersionInfos(ctx.grpcCtx, &grpcapi.RetrieveVersionInfosRequest{ModelId: "bar", VersionNumbers: []int32{-1}})
		assert.NoError(t, err)
		assert.Len(t, rep.VersionInfos, 1)
		assert.Equal(t, "1", rep.NextVersionHandle)

		assert.Equal(t, "bar", rep.VersionInfos[0].ModelId)
		assert.Equal(t, 10, int(rep.VersionInfos[0].VersionNumber))
		assert.True(t, rep.VersionInfos[0].Archived)
		assert.NotZero(t, rep.VersionInfos[0].DataHash)
		assert.NotZero(t, rep.VersionInfos[0].CreationTimestamp)
	}
	{
		rep, err := ctx.client.RetrieveVersionInfos(ctx.grpcCtx, &grpcapi.RetrieveVersionInfosRequest{ModelId: "bar", VersionNumbers: []int32{28}})
		assert.Error(t, err)
		assert.Equal(t, codes.NotFound, status.Code(err))
		assert.Nil(t, rep)
	}
}

func TestGetModelVersionData(t *testing.T) {
	modelUserData := make(map[string]string)
	modelUserData["model_test1"] = "model_test1"
	modelUserData["model_test2"] = "model_test2"
	modelUserData["model_test3"] = "model_test3"

	versionUserData := make(map[string]string)
	versionUserData["version_test1"] = "version_test1"
	versionUserData["version_test2"] = "version_test2"
	versionUserData["version_test3"] = "version_test3"

	ctx, err := createContext(16) // For the purpose of the test we limit the sent chunk size drastically
	assert.NoError(t, err)
	defer ctx.destroy()
	{
		_, err := ctx.client.CreateOrUpdateModel(ctx.grpcCtx, &grpcapi.CreateOrUpdateModelRequest{ModelInfo: &grpcapi.ModelInfo{ModelId: "baz", UserData: modelUserData}})
		assert.NoError(t, err)
	}
	{
		stream, err := ctx.client.CreateVersion(ctx.grpcCtx)
		assert.NoError(t, err)
		err = stream.Send(&grpcapi.CreateVersionRequestChunk{
			Msg: &grpcapi.CreateVersionRequestChunk_Header_{
				Header: &grpcapi.CreateVersionRequestChunk_Header{
					VersionInfo: &grpcapi.ModelVersionInfo{
						ModelId:           "baz",
						CreationTimestamp: nsTimestampFromTime(time.Now()),
						Archived:          false,
						DataHash:          backend.ComputeSHA256Hash(modelData),
						DataSize:          uint64(len(modelData)),
						UserData:          versionUserData,
					},
				},
			},
		})
		assert.NoError(t, err)
		err = stream.Send(&grpcapi.CreateVersionRequestChunk{Msg: &grpcapi.CreateVersionRequestChunk_Body_{Body: &grpcapi.CreateVersionRequestChunk_Body{
			DataChunk: modelData,
		}}})
		assert.NoError(t, err)
		_, err = stream.CloseAndRecv()
		assert.NoError(t, err)
	}
	{
		stream, err := ctx.client.RetrieveVersionData(ctx.grpcCtx, &grpcapi.RetrieveVersionDataRequest{ModelId: "baz", VersionNumber: -1})
		assert.NoError(t, err)
		chunks := []*grpcapi.RetrieveVersionDataReplyChunk{}
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
				assert.GreaterOrEqual(t, 16, len(chunk.DataChunk))
			} else {
				assert.Equal(t, 16, len(chunk.DataChunk))
			}
		}
		assert.Equal(t, modelData, data)
	}
	{
		stream, err := ctx.client.RetrieveVersionData(ctx.grpcCtx, &grpcapi.RetrieveVersionDataRequest{ModelId: "baz", VersionNumber: 4})
		assert.NoError(t, err)
		chunk, err := stream.Recv()
		assert.Equal(t, codes.NotFound, status.Code(err))
		assert.Nil(t, chunk)
	}
}
