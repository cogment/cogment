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
	"strconv"
	"time"

	"github.com/cogment/cogment-model-registry/backend"
	grpcapi "github.com/cogment/cogment-model-registry/grpcapi/cogment/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func timeFromNsTimestamp(timestamp uint64) time.Time {
	return time.Unix(0, int64(timestamp))
}

func nsTimestampFromTime(timestamp time.Time) uint64 {
	return uint64(timestamp.UnixNano())
}

type ModelRegistryServer struct {
	grpcapi.UnimplementedModelRegistrySPServer
	backendPromise                BackendPromise
	sentModelVersionDataChunkSize int
}

func createPbModelVersionInfo(modelVersionInfo backend.VersionInfo) grpcapi.ModelVersionInfo {
	return grpcapi.ModelVersionInfo{
		ModelId:           modelVersionInfo.ModelID,
		VersionNumber:     uint32(modelVersionInfo.VersionNumber),
		CreationTimestamp: nsTimestampFromTime(modelVersionInfo.CreationTimestamp),
		Archived:          modelVersionInfo.Archived,
		DataHash:          modelVersionInfo.DataHash,
		DataSize:          uint64(modelVersionInfo.DataSize),
		UserData:          modelVersionInfo.UserData,
	}
}

func createBackendModelInfo(modelInfo *grpcapi.ModelInfo) backend.ModelInfo {
	return backend.ModelInfo{
		ModelID:  modelInfo.ModelId,
		UserData: modelInfo.UserData,
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

	_, err = b.CreateOrUpdateModel(modelInfo)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "unexpected error while creating model %q: %s", modelInfo.ModelID, err)
	}

	return &grpcapi.CreateOrUpdateModelReply{}, nil
}

func (s *ModelRegistryServer) DeleteModel(ctx context.Context, req *grpcapi.DeleteModelRequest) (*grpcapi.DeleteModelReply, error) {
	log.Printf("DeleteModel(req={ModelId: %q})\n", req.ModelId)

	b, err := s.backendPromise.Await(ctx)
	if err != nil {
		return nil, err
	}

	err = b.DeleteModel(req.ModelId)
	if err != nil {
		if _, ok := err.(*backend.UnknownModelError); ok {
			return nil, status.Errorf(codes.NotFound, "%s", err)
		}
		return nil, status.Errorf(codes.Internal, "unexpected error while deleting model %q: %s", req.ModelId, err)
	}

	return &grpcapi.DeleteModelReply{}, nil
}

func (s *ModelRegistryServer) RetrieveModels(ctx context.Context, req *grpcapi.RetrieveModelsRequest) (*grpcapi.RetrieveModelsReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RetrieveModels not implemented")
}

func (s *ModelRegistryServer) CreateVersion(inStream grpcapi.ModelRegistrySP_CreateVersionServer) error {
	log.Printf("CreateVersion(stream=...)\n")

	firstChunk, err := inStream.Recv()
	if err == io.EOF {
		return status.Errorf(codes.InvalidArgument, "empty request")
	}
	if err != nil {
		return err
	}
	if firstChunk.GetHeader() == nil {
		return status.Errorf(codes.InvalidArgument, "first request chunk do not include a Header")
	}

	receivedVersionInfo := firstChunk.GetHeader().GetVersionInfo()

	modelData := []byte{}

	for {
		chunk, err := inStream.Recv()
		if err == io.EOF {
			receivedDataSize := uint64(len(modelData))
			if receivedDataSize == receivedVersionInfo.DataSize && err == io.EOF {
				break
			}
			if err == io.EOF {
				return status.Errorf(codes.InvalidArgument, "stream ended while having not received the expected data, expected %d bytes, received %d bytes", receivedVersionInfo.DataSize, receivedDataSize)
			}
			if receivedDataSize > receivedVersionInfo.DataSize {
				return status.Errorf(codes.InvalidArgument, "received more data than expected, expected %d bytes, received %d bytes", receivedVersionInfo.DataSize, receivedDataSize)
			}
		}
		if err != nil {
			return err
		}
		if chunk.GetBody() == nil {
			return status.Errorf(codes.InvalidArgument, "subsequent request chunk do not include a Body")
		}
		modelData = append(modelData, chunk.GetBody().DataChunk...)
	}

	receivedHash := backend.ComputeSHA256Hash(modelData)

	if receivedVersionInfo.DataHash != "" && receivedVersionInfo.DataHash != receivedHash {
		return status.Errorf(codes.InvalidArgument, "received data did not match the expected hash, expected %q, received %q", receivedVersionInfo.DataHash, receivedHash)
	}

	b, err := s.backendPromise.Await(inStream.Context())
	if err != nil {
		return err
	}

	creationTimestamp := time.Now()
	if receivedVersionInfo.CreationTimestamp > 0 {
		creationTimestamp = timeFromNsTimestamp(receivedVersionInfo.CreationTimestamp)
	}

	versionInfo, err := b.CreateOrUpdateModelVersion(receivedVersionInfo.ModelId, backend.VersionArgs{
		VersionNumber:     -1,
		CreationTimestamp: creationTimestamp,
		Archived:          receivedVersionInfo.Archived,
		DataHash:          receivedHash,
		Data:              modelData,
		UserData:          receivedVersionInfo.UserData,
	})
	if err != nil {
		return status.Errorf(codes.Internal, "unexpected error while creating a version for model %q: %s", receivedVersionInfo.ModelId, err)
	}

	pbVersionInfo := createPbModelVersionInfo(versionInfo)
	return inStream.SendAndClose(&grpcapi.CreateVersionReply{VersionInfo: &pbVersionInfo})
}

func (s *ModelRegistryServer) RetrieveVersionInfos(ctx context.Context, req *grpcapi.RetrieveVersionInfosRequest) (*grpcapi.RetrieveVersionInfosReply, error) {
	log.Printf("RetrieveVersionInfos(req={ModelId: %q, VersionNumbers: %#v, VersionsCount: %d, VersionHandle: %q})\n", req.ModelId, req.VersionNumbers, req.VersionsCount, req.VersionHandle)

	offset := 0
	if req.VersionHandle != "" {
		var err error
		offset, err = strconv.Atoi(req.VersionHandle)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid value for `version_handle` (%q) only empty or values provided by a previous call should be used", req.VersionHandle)
		}
	}

	b, err := s.backendPromise.Await(ctx)
	if err != nil {
		return nil, err
	}

	if len(req.VersionNumbers) == 0 {
		// Retrieve all version infos
		versionInfos, err := b.ListModelVersionInfos(req.ModelId, offset, int(req.VersionsCount))
		if err != nil {
			if _, ok := err.(*backend.UnknownModelError); ok {
				return nil, status.Errorf(codes.NotFound, "%s", err)
			}
			return nil, status.Errorf(codes.Internal, "unexpected error while deleting model %q: %s", req.ModelId, err)
		}

		pbVersionInfos := []*grpcapi.ModelVersionInfo{}

		nextVersionNumber := offset
		for _, versionInfo := range versionInfos {
			pbVersionInfo := createPbModelVersionInfo(versionInfo)
			pbVersionInfos = append(pbVersionInfos, &pbVersionInfo)
			nextVersionNumber = versionInfo.VersionNumber + 1
		}

		return &grpcapi.RetrieveVersionInfosReply{
			VersionInfos:      pbVersionInfos,
			NextVersionHandle: strconv.Itoa(nextVersionNumber),
		}, nil
	}

	pbVersionInfos := []*grpcapi.ModelVersionInfo{}
	versionNumberSlice := req.VersionNumbers[offset:]
	if req.VersionsCount > 0 {
		versionNumberSlice = versionNumberSlice[:req.VersionsCount]
	}
	nextVersionNumber := offset
	for _, versionNumber := range versionNumberSlice {
		versionInfo, err := b.RetrieveModelVersionInfo(req.ModelId, int(versionNumber))
		if err != nil {
			if _, ok := err.(*backend.UnknownModelError); ok {
				return nil, status.Errorf(codes.NotFound, "%s", err)
			}
			if _, ok := err.(*backend.UnknownModelVersionError); ok {
				return nil, status.Errorf(codes.NotFound, "%s", err)
			}
			return nil, status.Errorf(codes.Internal, `unexpected error while retrieving version "%d" for model %q: %s`, versionNumber, req.ModelId, err)
		}

		pbVersionInfo := createPbModelVersionInfo(versionInfo)
		pbVersionInfos = append(pbVersionInfos, &pbVersionInfo)
		nextVersionNumber = versionInfo.VersionNumber + 1
	}

	return &grpcapi.RetrieveVersionInfosReply{
		VersionInfos:      pbVersionInfos,
		NextVersionHandle: strconv.Itoa(nextVersionNumber),
	}, nil
}

func (s *ModelRegistryServer) RetrieveVersionData(req *grpcapi.RetrieveVersionDataRequest, outStream grpcapi.ModelRegistrySP_RetrieveVersionDataServer) error {
	log.Printf("RetrieveVersionData(req={ModelId: %q, VersionNumber: %d})\n", req.ModelId, req.VersionNumber)

	b, err := s.backendPromise.Await(outStream.Context())
	if err != nil {
		return err
	}

	modelData, err := b.RetrieveModelVersionData(req.ModelId, int(req.VersionNumber))
	if err != nil {
		if _, ok := err.(*backend.UnknownModelError); ok {
			return status.Errorf(codes.NotFound, "%s", err)
		}
		if _, ok := err.(*backend.UnknownModelVersionError); ok {
			return status.Errorf(codes.NotFound, "%s", err)
		}
		return status.Errorf(codes.Internal, `unexpected error while retrieving version "%d" for model %q: %s`, req.VersionNumber, req.ModelId, err)
	}

	dataLen := len(modelData)
	if dataLen == 0 {
		return outStream.Send(&grpcapi.RetrieveVersionDataReplyChunk{})
	}

	for i := 0; i < dataLen; i += s.sentModelVersionDataChunkSize {
		var replyChunk grpcapi.RetrieveVersionDataReplyChunk
		if i+s.sentModelVersionDataChunkSize >= dataLen {
			replyChunk = grpcapi.RetrieveVersionDataReplyChunk{DataChunk: modelData[i:dataLen]}
		} else {
			replyChunk = grpcapi.RetrieveVersionDataReplyChunk{DataChunk: modelData[i : i+s.sentModelVersionDataChunkSize]}
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

	grpcapi.RegisterModelRegistrySPServer(grpcServer, server)
	return server, nil
}
