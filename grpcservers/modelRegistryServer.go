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

	"github.com/cogment/cogment-model-registry/backend"
	grpcapi "github.com/cogment/cogment-model-registry/grpcapi/cogment/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ModelRegistryServer struct {
	grpcapi.UnimplementedModelRegistryServer
	backendPromise                BackendPromise
	sentModelVersionDataChunkSize int
}

func createPbModelInfo(modelID string) grpcapi.ModelInfo {
	return grpcapi.ModelInfo{ModelId: modelID}
}

func createPbModelVersionInfo(modelVersionInfo backend.VersionInfo) grpcapi.ModelVersionInfo {
	return grpcapi.ModelVersionInfo{
		ModelId:   modelVersionInfo.ModelID,
		CreatedAt: timestamppb.New(modelVersionInfo.CreatedAt),
		Number:    uint32(modelVersionInfo.Number),
		Archive:   modelVersionInfo.Archive,
		Hash:      modelVersionInfo.Hash,
	}
}

func createBackendModelInfo(modelInfo *grpcapi.ModelInfo) backend.ModelInfo {
	return backend.ModelInfo{
		ModelID:  modelInfo.ModelId,
		Metadata: modelInfo.Metadata,
	}
}

func (s *ModelRegistryServer) SetBackend(b backend.Backend) {
	s.backendPromise.Set(b)
}

func (s *ModelRegistryServer) CreateOrUpdateModel(ctx context.Context, req *grpcapi.CreateOrUpdateModelRequest) (*grpcapi.CreateOrUpdateModelReply, error) {
	log.Printf("CreateOrUpdateModel(req={ModelId: %q})\n", req.ModelInfo.ModelId)

	modelInfo := createBackendModelInfo(req.ModelInfo)

	b, err := s.backendPromise.Await(ctx)
	if err != nil {
		return nil, err
	}

	newModelInfo, err := b.CreateOrUpdateModel(modelInfo)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "unexpected error while creating model %q: %s", modelInfo.ModelID, err)
	}

	return &grpcapi.CreateOrUpdateModelReply{ModelInfo: &grpcapi.ModelInfo{ModelId: newModelInfo.ModelID, Metadata: newModelInfo.Metadata}}, nil
}

func (s *ModelRegistryServer) DeleteModel(ctx context.Context, req *grpcapi.DeleteModelRequest) (*grpcapi.DeleteModelReply, error) {
	log.Printf("CreateOrUpdateModel(req={ModelId: %q})\n", req.ModelId)

	modelInfo := createPbModelInfo(req.ModelId)

	b, err := s.backendPromise.Await(ctx)
	if err != nil {
		return nil, err
	}

	err = b.DeleteModel(modelInfo.ModelId)
	if err != nil {
		if _, ok := err.(*backend.UnknownModelError); ok {
			return nil, status.Errorf(codes.NotFound, "%s", err)
		}
		return nil, status.Errorf(codes.Internal, "unexpected error while deleting model %q: %s", modelInfo.ModelId, err)
	}

	return &grpcapi.DeleteModelReply{Model: &modelInfo}, nil
}

func (s *ModelRegistryServer) CreateOrUpdateModelVersion(inStream grpcapi.ModelRegistry_CreateOrUpdateModelVersionServer) error {
	log.Printf("CreateOrUpdateModelVersion(stream=...)\n")

	firstChunk, err := inStream.Recv()
	if err == io.EOF {
		return status.Errorf(codes.InvalidArgument, "empty request")
	}
	if err != nil {
		return err
	}
	if firstChunk.ModelId == "" {
		return status.Errorf(codes.InvalidArgument, "required `model_id` was not defined in the first request chunk")
	}

	modelID := firstChunk.ModelId
	archive := firstChunk.Archive
	metadata := firstChunk.Metadata
	modelData := firstChunk.DataChunk
	lastChunk := firstChunk.LastChunk

	for {
		chunk, err := inStream.Recv()
		if err == io.EOF || lastChunk {
			if lastChunk && err == io.EOF {
				break
			}
			if err == io.EOF {
				return status.Errorf(codes.InvalidArgument, "no chunk having `last_chunk=True` received")
			}
			if lastChunk {
				return status.Errorf(codes.InvalidArgument, "chunk received after `last_chunk=True` received")
			}
		}
		if err != nil {
			return err
		}
		modelData = append(modelData, chunk.DataChunk...)
		lastChunk = chunk.LastChunk
	}

	b, err := s.backendPromise.Await(inStream.Context())
	if err != nil {
		return err
	}

	versionInfo, err := b.CreateOrUpdateModelVersion(modelID, backend.VersionInfoArgs{
		VersionNumber: -1,
		Data:          modelData,
		Archive:       archive,
		Metadata:      metadata,
	})
	if err != nil {
		return status.Errorf(codes.Internal, "unexpected error while creating a version for model %q: %s", modelID, err)
	}

	pbVersionInfo := createPbModelVersionInfo(versionInfo)
	return inStream.SendAndClose(&grpcapi.CreateOrUpdateModelVersionReply{VersionInfo: &pbVersionInfo})
}

func (s *ModelRegistryServer) ListModelVersions(ctx context.Context, req *grpcapi.ListModelVersionsRequest) (*grpcapi.ListModelVersionsReply, error) {
	log.Printf("ListModelVersions(req={ModelId: %q, PageOffset: %d, PageSize: %d})\n", req.ModelId, req.PageOffset, req.PageSize)

	offset := int(req.PageOffset)
	if offset < 0 {
		offset = 0
	}

	b, err := s.backendPromise.Await(ctx)
	if err != nil {
		return nil, err
	}

	versionInfos, err := b.ListModelVersionInfos(req.ModelId, offset, int(req.PageSize))
	if err != nil {
		if _, ok := err.(*backend.UnknownModelError); ok {
			return nil, status.Errorf(codes.NotFound, "%s", err)
		}
		return nil, status.Errorf(codes.Internal, "unexpected error while deleting model %q: %s", req.ModelId, err)
	}

	pbVersionInfos := []*grpcapi.ModelVersionInfo{}

	for _, versionInfo := range versionInfos {
		pbVersionInfo := createPbModelVersionInfo(versionInfo)
		pbVersionInfos = append(pbVersionInfos, &pbVersionInfo)
	}

	return &grpcapi.ListModelVersionsReply{
		Versions:       pbVersionInfos,
		NextPageOffset: int32(offset + len(pbVersionInfos)),
	}, nil
}

func (s *ModelRegistryServer) RetrieveModelVersionInfo(ctx context.Context, req *grpcapi.RetrieveModelVersionInfoRequest) (*grpcapi.RetrieveModelVersionInfoReply, error) {
	log.Printf("RetrieveModelVersionInfo(req={ModelId: %q, Number: %d})\n", req.ModelId, req.Number)

	b, err := s.backendPromise.Await(ctx)
	if err != nil {
		return nil, err
	}

	versionInfo, err := b.RetrieveModelVersionInfo(req.ModelId, int(req.Number))
	if err != nil {
		if _, ok := err.(*backend.UnknownModelError); ok {
			return nil, status.Errorf(codes.NotFound, "%s", err)
		}
		if _, ok := err.(*backend.UnknownModelVersionError); ok {
			return nil, status.Errorf(codes.NotFound, "%s", err)
		}
		return nil, status.Errorf(codes.Internal, `unexpected error while retrieving version "%d" for model %q: %s`, req.Number, req.ModelId, err)
	}

	pbVersionInfo := createPbModelVersionInfo(versionInfo)
	return &grpcapi.RetrieveModelVersionInfoReply{VersionInfo: &pbVersionInfo}, nil
}

func (s *ModelRegistryServer) RetrieveModelVersionData(req *grpcapi.RetrieveModelVersionDataRequest, outStream grpcapi.ModelRegistry_RetrieveModelVersionDataServer) error {
	log.Printf("RetrieveModelVersionData(req={ModelId: %q, Number: %d})\n", req.ModelId, req.Number)

	b, err := s.backendPromise.Await(outStream.Context())
	if err != nil {
		return err
	}

	modelData, err := b.RetrieveModelVersionData(req.ModelId, int(req.Number))
	if err != nil {
		if _, ok := err.(*backend.UnknownModelError); ok {
			return status.Errorf(codes.NotFound, "%s", err)
		}
		if _, ok := err.(*backend.UnknownModelVersionError); ok {
			return status.Errorf(codes.NotFound, "%s", err)
		}
		return status.Errorf(codes.Internal, `unexpected error while retrieving version "%d" for model %q: %s`, req.Number, req.ModelId, err)
	}

	dataLen := len(modelData)
	if dataLen == 0 {
		return outStream.Send(&grpcapi.RetrieveModelVersionDataReplyChunk{LastChunk: true})
	}

	for i := 0; i < dataLen; i += s.sentModelVersionDataChunkSize {
		var replyChunk grpcapi.RetrieveModelVersionDataReplyChunk
		if i+s.sentModelVersionDataChunkSize >= dataLen {
			replyChunk = grpcapi.RetrieveModelVersionDataReplyChunk{DataChunk: modelData[i:dataLen], LastChunk: true}
		} else {
			replyChunk = grpcapi.RetrieveModelVersionDataReplyChunk{DataChunk: modelData[i : i+s.sentModelVersionDataChunkSize], LastChunk: false}
		}
		err := outStream.Send(&replyChunk)
		if err != nil {
			return err
		}
	}

	return nil
}

func RegisterModelRegistryServer(grpcServer grpc.ServiceRegistrar, sentModelVersionDataChunkSize int) (*ModelRegistryServer, error) {
	server := &ModelRegistryServer{
		sentModelVersionDataChunkSize: sentModelVersionDataChunkSize,
	}

	grpcapi.RegisterModelRegistryServer(grpcServer, server)
	return server, nil
}
