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

package grpcservers

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"

	cogmentAPI "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/cogment/cogment/services/registry/backend"
	"github.com/sirupsen/logrus"
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
	cogmentAPI.UnimplementedModelRegistrySPServer
	backendPromise                BackendPromise
	sentModelVersionDataChunkSize int
	newVersion                    *sync.Cond
}

func makePbModelVersionInfo(modelVersionInfo backend.VersionInfo) cogmentAPI.ModelVersionInfo {
	return cogmentAPI.ModelVersionInfo{
		ModelId:           modelVersionInfo.ModelID,
		VersionNumber:     uint32(modelVersionInfo.VersionNumber),
		CreationTimestamp: nsTimestampFromTime(modelVersionInfo.CreationTimestamp),
		Archived:          modelVersionInfo.Archived,
		DataHash:          modelVersionInfo.DataHash,
		DataSize:          uint64(modelVersionInfo.DataSize),
		UserData:          modelVersionInfo.UserData,
	}
}

func (s *ModelRegistryServer) SetBackend(b backend.Backend) {
	s.backendPromise.Set(b)
}

func (s *ModelRegistryServer) Version(
	_ context.Context,
	_ *cogmentAPI.VersionRequest,
) (*cogmentAPI.VersionInfo, error) {

	// TODO: Return proper version info. Current version is minimal to serve a health check for directory.
	res := &cogmentAPI.VersionInfo{}
	return res, nil
}

// gRPC interface
func (s *ModelRegistryServer) Status(_ context.Context, request *cogmentAPI.StatusRequest,
) (*cogmentAPI.StatusReply, error) {
	reply := cogmentAPI.StatusReply{}

	if len(request.Names) == 0 {
		return &reply, nil
	}
	reply.Statuses = make(map[string]string)

	// We purposefully don't scan for "*" ahead of time to allow explicit values before.
	all := false
	for _, name := range request.Names {
		if name == "*" {
			all = true
		}
		if all || name == "overall_load" {
			reply.Statuses["overall_load"] = "0"
		}
		if all {
			break
		}
	}

	return &reply, nil
}

func (s *ModelRegistryServer) CreateOrUpdateModel(
	ctx context.Context,
	req *cogmentAPI.CreateOrUpdateModelRequest,
) (*cogmentAPI.CreateOrUpdateModelReply, error) {
	log := log.WithFields(logrus.Fields{
		"model_id": req.ModelInfo.ModelId,
		"method":   "CreateOrUpdateModel",
	})

	log.Debug("Call received")

	modelInfo := backend.ModelInfo{
		ModelID:  req.ModelInfo.ModelId,
		UserData: req.ModelInfo.UserData,
	}

	b, err := s.backendPromise.Await(ctx)
	if err != nil {
		return nil, err
	}

	_, err = b.CreateOrUpdateModel(modelInfo)
	if err != nil {
		log.WithField("error", err).Error("Unexpected error while creating model")
		return nil, status.Errorf(codes.Internal, "unexpected error while creating model %q: %s", modelInfo.ModelID, err)
	}

	return &cogmentAPI.CreateOrUpdateModelReply{}, nil
}

func (s *ModelRegistryServer) DeleteModel(
	ctx context.Context,
	req *cogmentAPI.DeleteModelRequest,
) (*cogmentAPI.DeleteModelReply, error) {
	log := log.WithFields(logrus.Fields{
		"model_id": req.ModelId,
		"method":   "DeleteModel",
	})

	log.Debug("Call received")

	b, err := s.backendPromise.Await(ctx)
	if err != nil {
		return nil, err
	}

	err = b.DeleteModel(req.ModelId)
	if err != nil {
		if _, ok := err.(*backend.UnknownModelError); ok {
			return nil, status.Errorf(codes.NotFound, "%s", err)
		}
		log.WithField("error", err).Error("Unexpected error while deleting model")
		return nil, status.Errorf(codes.Internal, "unexpected error while deleting model %q: %s", req.ModelId, err)
	}

	return &cogmentAPI.DeleteModelReply{}, nil
}

func (s *ModelRegistryServer) RetrieveModels(
	ctx context.Context,
	req *cogmentAPI.RetrieveModelsRequest,
) (*cogmentAPI.RetrieveModelsReply, error) {
	log := log.WithFields(logrus.Fields{
		"model_ids":    req.ModelIds,
		"models_count": req.ModelsCount,
		"model_handle": req.ModelHandle,
		"method":       "RetrieveModels",
	})

	log.Debug("Call received")

	offset := 0
	if req.ModelHandle != "" {
		var err error
		offset64, err := strconv.ParseInt(req.ModelHandle, 10, 0)
		if err != nil {
			return nil, status.Errorf(
				codes.InvalidArgument,
				"Invalid value for `model_handle` (%q) only empty or values provided by a previous call should be used",
				req.ModelHandle,
			)
		}
		offset = int(offset64)
	}

	b, err := s.backendPromise.Await(ctx)
	if err != nil {
		return nil, err
	}

	pbModelInfos := []*cogmentAPI.ModelInfo{}

	if len(req.ModelIds) == 0 {
		// Retrieve all models
		modelInfos, err := b.ListModels(offset, int(req.ModelsCount))
		if err != nil {
			log.WithField("error", err).Error("Unexpected error while retrieving models")
			return nil, status.Errorf(codes.Internal, "unexpected error while retrieving models: %s", err)
		}

		for _, modelInfo := range modelInfos {
			pbModelInfo := cogmentAPI.ModelInfo{ModelId: modelInfo.ModelID, UserData: modelInfo.UserData}
			pbModelInfos = append(pbModelInfos, &pbModelInfo)
		}
	} else {
		modelIDsSlice := req.ModelIds[offset:]
		if req.ModelsCount > 0 {
			modelIDsSlice = modelIDsSlice[:req.ModelsCount]
		}
		for _, modelID := range modelIDsSlice {
			modelInfo, err := b.RetrieveModelInfo(modelID)
			if err != nil {
				if _, ok := err.(*backend.UnknownModelError); ok {
					return nil, status.Errorf(codes.NotFound, "%s", err)
				}
				log.WithField("error", err).Error("Unexpected error while retrieving models")
				return nil, status.Errorf(codes.Internal, `unexpected error while retrieving models: %s`, err)
			}

			pbModelInfo := cogmentAPI.ModelInfo{ModelId: modelInfo.ModelID, UserData: modelInfo.UserData}
			pbModelInfos = append(pbModelInfos, &pbModelInfo)
		}
	}

	nextOffset := offset + len(pbModelInfos)

	return &cogmentAPI.RetrieveModelsReply{
		ModelInfos:      pbModelInfos,
		NextModelHandle: strconv.FormatInt(int64(nextOffset), 10),
	}, nil
}

func (s *ModelRegistryServer) CreateVersion(inStream cogmentAPI.ModelRegistrySP_CreateVersionServer) error {
	log := log.WithFields(logrus.Fields{
		"method": "CreateVersion",
	})

	log.Debug("Call received")

	firstChunk, err := inStream.Recv()
	if err == io.EOF {
		return status.Errorf(codes.InvalidArgument, "empty request")
	}
	if err != nil {
		return err
	}
	if firstChunk.GetHeader() == nil {
		return status.Errorf(codes.InvalidArgument, "first request chunk does not include a Header")
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
				return status.Errorf(
					codes.InvalidArgument,
					"stream ended while having not received the expected data, expected %d bytes, received %d bytes",
					receivedVersionInfo.DataSize, receivedDataSize,
				)
			}
			if receivedDataSize > receivedVersionInfo.DataSize {
				return status.Errorf(
					codes.InvalidArgument,
					"received more data than expected, expected %d bytes, received %d bytes",
					receivedVersionInfo.DataSize, receivedDataSize,
				)
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
		return status.Errorf(
			codes.InvalidArgument,
			"received data did not match the expected hash, expected %q, received %q",
			receivedVersionInfo.DataHash, receivedHash,
		)
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
		CreationTimestamp: creationTimestamp,
		Archived:          receivedVersionInfo.Archived,
		DataHash:          receivedHash,
		Data:              modelData,
		UserData:          receivedVersionInfo.UserData,
	})
	if err != nil {
		log.WithField("error", err).Error("Unexpected error while creating a version")
		return status.Errorf(
			codes.Internal,
			"unexpected error while creating a version for model %q: %s",
			receivedVersionInfo.ModelId, err,
		)
	}
	s.newVersion.Broadcast()

	pbVersionInfo := makePbModelVersionInfo(versionInfo)
	return inStream.SendAndClose(&cogmentAPI.CreateVersionReply{VersionInfo: &pbVersionInfo})
}

func (s *ModelRegistryServer) RetrieveVersionInfos(
	ctx context.Context,
	req *cogmentAPI.RetrieveVersionInfosRequest,
) (*cogmentAPI.RetrieveVersionInfosReply, error) {
	log := log.WithFields(logrus.Fields{
		"model_id":        req.ModelId,
		"versions_number": req.VersionNumbers,
		"versions_count":  req.VersionsCount,
		"version_handle":  req.VersionHandle,
		"method":          "RetrieveVersionInfos",
	})

	log.Debug("Call received")

	initialVersionNumber := uint(0)
	if req.VersionHandle != "" {
		var err error
		initialVersionNumber64, err := strconv.ParseUint(req.VersionHandle, 10, 0)
		if err != nil {
			return nil, status.Errorf(
				codes.InvalidArgument,
				"Invalid value for `version_handle` (%q) only empty or values provided by a previous call should be used",
				req.VersionHandle,
			)
		}
		initialVersionNumber = uint(initialVersionNumber64)
	}

	b, err := s.backendPromise.Await(ctx)
	if err != nil {
		return nil, err
	}

	if len(req.VersionNumbers) == 0 {
		// Retrieve all version infos
		versionInfos, err := b.ListModelVersionInfos(req.ModelId, initialVersionNumber, int(req.VersionsCount))
		if err != nil {
			if _, ok := err.(*backend.UnknownModelError); ok {
				return nil, status.Errorf(codes.NotFound, "%s", err)
			}
			log.WithField("error", err).Error("Unexpected error while listing versions")
			return nil, status.Errorf(
				codes.Internal,
				"unexpected error while listing versions for model %q: %s", req.ModelId, err,
			)
		}

		pbVersionInfos := []*cogmentAPI.ModelVersionInfo{}

		nextVersionNumber := initialVersionNumber
		for _, versionInfo := range versionInfos {
			pbVersionInfo := makePbModelVersionInfo(versionInfo)
			pbVersionInfos = append(pbVersionInfos, &pbVersionInfo)
			nextVersionNumber = versionInfo.VersionNumber + 1
		}

		return &cogmentAPI.RetrieveVersionInfosReply{
			VersionInfos:      pbVersionInfos,
			NextVersionHandle: strconv.FormatUint(uint64(nextVersionNumber), 10),
		}, nil
	}

	pbVersionInfos := []*cogmentAPI.ModelVersionInfo{}
	versionNumberSlice := req.VersionNumbers[initialVersionNumber:]
	if req.VersionsCount > 0 {
		versionNumberSlice = versionNumberSlice[:req.VersionsCount]
	}
	nextVersionNumber := initialVersionNumber
	for _, versionNumber := range versionNumberSlice {
		versionInfo, err := b.RetrieveModelVersionInfo(req.ModelId, int(versionNumber))
		if err != nil {
			if _, ok := err.(*backend.UnknownModelError); ok {
				return nil, status.Errorf(codes.NotFound, "%s", err)
			}
			if _, ok := err.(*backend.UnknownModelVersionError); ok {
				return nil, status.Errorf(codes.NotFound, "%s", err)
			}
			log.WithFields(logrus.Fields{
				"version_number": versionNumber,
				"error":          err,
			}).Error("Unexpected error while retrieving version info")
			return nil, status.Errorf(
				codes.Internal,
				`unexpected error while retrieving version "%d" for model %q: %s`,
				versionNumber, req.ModelId, err,
			)
		}

		pbVersionInfo := makePbModelVersionInfo(versionInfo)
		pbVersionInfos = append(pbVersionInfos, &pbVersionInfo)
		nextVersionNumber = versionInfo.VersionNumber + 1
	}

	return &cogmentAPI.RetrieveVersionInfosReply{
		VersionInfos:      pbVersionInfos,
		NextVersionHandle: strconv.FormatUint(uint64(nextVersionNumber), 10),
	}, nil
}

func (s *ModelRegistryServer) RetrieveVersionData(
	req *cogmentAPI.RetrieveVersionDataRequest,
	outStream cogmentAPI.ModelRegistrySP_RetrieveVersionDataServer,
) error {
	log := log.WithFields(logrus.Fields{
		"model_id":       req.ModelId,
		"version_number": req.VersionNumber,
		"method":         "RetrieveVersionData",
	})

	log.Debug("Call received")

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
		log.WithField("error", err).Error("Unexpected error while retrieving version data")
		return status.Errorf(
			codes.Internal,
			`unexpected error while retrieving version "%d" for model %q: %s`,
			req.VersionNumber, req.ModelId, err,
		)
	}

	dataLen := len(modelData)
	if dataLen == 0 {
		return outStream.Send(&cogmentAPI.RetrieveVersionDataReplyChunk{})
	}

	for i := 0; i < dataLen; i += s.sentModelVersionDataChunkSize {
		var replyChunk cogmentAPI.RetrieveVersionDataReplyChunk
		if i+s.sentModelVersionDataChunkSize >= dataLen {
			replyChunk = cogmentAPI.RetrieveVersionDataReplyChunk{DataChunk: modelData[i:dataLen]}
		} else {
			replyChunk = cogmentAPI.RetrieveVersionDataReplyChunk{DataChunk: modelData[i : i+s.sentModelVersionDataChunkSize]}
		}
		err := outStream.Send(&replyChunk)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *ModelRegistryServer) VersionUpdate(
	req *cogmentAPI.VersionUpdateRequest,
	outStream cogmentAPI.ModelRegistrySP_VersionUpdateServer,
) error {
	log := log.WithFields(logrus.Fields{
		"model_id": req.ModelId,
		"method":   "VersionUpdate",
	})

	log.Debug("Call received")

	be, err := s.backendPromise.Await(outStream.Context())
	if err != nil {
		return err
	}

	modelExists, err := be.HasModel(req.ModelId)
	if err != nil {
		return err
	}
	if !modelExists {
		return status.Errorf(codes.InvalidArgument, "unknown model id [%s]", req.ModelId)
	}

	var lastVersion uint
	for {
		versionInfo, err := be.RetrieveModelLastVersionInfo(req.ModelId)
		if err != nil {
			return err
		}

		// In case no version is available, 'VersionNumber' is set to default (0).
		if versionInfo.VersionNumber > lastVersion {
			pbVersionInfo := makePbModelVersionInfo(versionInfo)
			reply := cogmentAPI.VersionUpdateReply{VersionInfo: &pbVersionInfo}
			err = outStream.Send(&reply)
			if err != nil {
				if lastVersion == 0 {
					// First "send": most probably a connection problem
					return err
				}
				log.Debug("Stream probably closed by client: ", err)
				break
			}
			lastVersion = versionInfo.VersionNumber
		}

		s.newVersion.L.Lock()
		s.newVersion.Wait()
		s.newVersion.L.Unlock()

		if outStream.Context().Err() != nil {
			log.Debug("Stream context ended: ", outStream.Context().Err())
			break
		}
	}

	return nil
}

func RegisterModelRegistryServer(
	grpcServer grpc.ServiceRegistrar,
	sentModelVersionDataChunkSize int,
) (*ModelRegistryServer, error) {
	if sentModelVersionDataChunkSize <= 0 {
		return nil, fmt.Errorf(
			"Unable to create the model registry server, `sentModelVersionDataChunkSize` needs to strictly positive",
		)
	}

	lock := sync.Mutex{}
	server := &ModelRegistryServer{
		sentModelVersionDataChunkSize: sentModelVersionDataChunkSize,
		newVersion:                    sync.NewCond(&lock),
	}

	cogmentAPI.RegisterModelRegistrySPServer(grpcServer, server)
	return server, nil
}
