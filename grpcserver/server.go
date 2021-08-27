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

	"github.com/cogment/model-registry/backend"
	grpcapi "github.com/cogment/model-registry/grpcapi/cogment/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type server struct {
	grpcapi.UnimplementedModelRegistryServer
	backend backend.Backend
}

func (s *server) CreateModel(ctx context.Context, req *grpcapi.CreateModelRequest) (*grpcapi.CreateModelReply, error) {
	log.Printf("CreateModel(req={ModelId: %q})\n", req.ModelId)
	return nil, status.Errorf(codes.Unimplemented, "method CreateModel not implemented")
}
func (s *server) DeleteModel(ctx context.Context, req *grpcapi.DeleteModelRequest) (*grpcapi.DeleteModelReply, error) {
	log.Printf("CreateModel(req={ModelId: %q})\n", req.ModelId)
	return nil, status.Errorf(codes.Unimplemented, "method DeleteModel not implemented")
}
func (s *server) ListModelVersions(ctx context.Context, req *grpcapi.ListModelVersionsRequest) (*grpcapi.ListModelVersionsReply, error) {
	log.Printf("CreateModel(req={ModelId: %q, PageSize: %d, PageOffset: %d})\n", req.ModelId, req.PageSize, req.PageOffset)
	return nil, status.Errorf(codes.Unimplemented, "method ListModelVersions not implemented")
}
func (s *server) CreateModelVersion(inStream grpcapi.ModelRegistry_CreateModelVersionServer) error {
	log.Printf("CreateModelVersion(stream=...)\n")
	return status.Errorf(codes.Unimplemented, "method CreateModelVersion not implemented")
}
func (s *server) RetrieveModelInfo(ctx context.Context, req *grpcapi.RetrieveModelInfoRequest) (*grpcapi.RetrieveModelInfoReply, error) {
	log.Printf("RetrieveModelInfo(req={ModelId: %q, Number: %q})\n", req.ModelId, req.Number)
	return nil, status.Errorf(codes.Unimplemented, "method RetrieveModelInfo not implemented")
}
func (s *server) RetrieveModelData(req *grpcapi.RetrieveModelDataRequest, outStream grpcapi.ModelRegistry_RetrieveModelDataServer) error {
	log.Printf("RetrieveModelInfo(req={ModelId: %q, Number: %q})\n", req.ModelId, req.Number)
	return status.Errorf(codes.Unimplemented, "method RetrieveModelData not implemented")
}

func RegisterServer(grpcServer grpc.ServiceRegistrar, backend backend.Backend) error {
	server := &server{
		backend: backend,
	}

	grpcapi.RegisterModelRegistryServer(grpcServer, server)
	return nil
}
