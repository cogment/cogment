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

	"github.com/cogment/cogment-activity-logger/backend"
	grpcapi "github.com/cogment/cogment-activity-logger/grpcapi/cogment/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type logExporterServer struct {
	grpcapi.UnimplementedLogExporterServer
	backend backend.Backend
}

func (s *logExporterServer) OnLogSample(stream grpcapi.LogExporter_OnLogSampleServer) error {
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		return status.Errorf(codes.DataLoss, "LogExporterServer.OnLogSample: failed to get metadata")
	}
	trialIDs := md.Get("trial-id")
	if len(trialIDs) == 0 {
		return status.Errorf(codes.InvalidArgument, "LogExporterServer.OnLogSample: missing required header \"trial-id\"")
	}
	trialID := trialIDs[0]
	// Receive the first element, it should be trial data
	req, err := stream.Recv()
	if err != nil {
		return err
	}
	trialParams := req.GetTrialParams()
	if trialParams == nil {
		return status.Errorf(codes.InvalidArgument, "LogExporterServer.OnLogSample: initial message is not of type \"cogment.TrialParams\"")
	}
	_, err = s.backend.OnTrialStart(trialID, trialParams)
	if err != nil {
		return status.Errorf(codes.Internal, "LogExporterServer.OnLogSample: internal error %q", err)
	}

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			_, err = s.backend.OnTrialEnd(trialID)
			if err != nil {
				return status.Errorf(codes.Internal, "LogExporterServer.OnLogSample: internal error %q", err)
			}
			return stream.Send(&grpcapi.LogExporterSampleReply{})
		}
		if err != nil {
			return err
		}
		sample := req.GetSample()
		if sample == nil {
			return status.Errorf(codes.InvalidArgument, "LogExporterServer.OnLogSample: body message is not of type \"cogment.DatalogSample\"")
		}
		_, err = s.backend.OnTrialSample(trialID, sample)
		if err != nil {
			return status.Errorf(codes.Internal, "LogExporterServer.OnLogSample: internal error %q", err)
		}
		err = stream.Send(&grpcapi.LogExporterSampleReply{})
		if err != nil {
			return err
		}
	}
}

func (s *logExporterServer) Version(context.Context, *grpcapi.VersionRequest) (*grpcapi.VersionInfo, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Version not implemented")
}

// RegisterLogExporterServer registers a LogExporterServer to a gRPC server.
func RegisterLogExporterServer(grpcServer grpc.ServiceRegistrar, backend backend.Backend) error {
	server := &logExporterServer{
		backend: backend,
	}

	grpcapi.RegisterLogExporterServer(grpcServer, server)
	return nil
}
