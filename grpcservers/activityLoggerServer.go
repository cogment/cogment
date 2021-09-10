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
	"log"

	"github.com/cogment/cogment-activity-logger/backend"
	grpcapi "github.com/cogment/cogment-activity-logger/grpcapi/cogment/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type activityLoggerServer struct {
	grpcapi.UnimplementedActivityLoggerServer
	backend backend.Backend
}

func (s *activityLoggerServer) ListenToTrial(req *grpcapi.ListenToTrialRequest, stream grpcapi.ActivityLogger_ListenToTrialServer) error {
	log.Printf("ListenToTrial(req={TrialId: %q})\n", req.TrialId)
	trialInfo, err := s.backend.GetTrialInfo(req.TrialId)
	if err != nil {
		return status.Errorf(codes.Internal, "ActivityLoggerServer.ListenToTrial: internal error %q", err)
	}

	trialParams, err := s.backend.GetTrialParams(trialInfo.TrialID)
	if err != nil {
		return status.Errorf(codes.Internal, "ActivityLoggerServer.ListenToTrial: internal error %q", err)
	}

	err = stream.Send(&grpcapi.ListenToTrialReply{Msg: &grpcapi.ListenToTrialReply_TrialParams{TrialParams: trialParams}})
	if err != nil {
		return err
	}

	trialSampleStream := s.backend.ConsumeTrialSamples(trialInfo.TrialID)

	for sampleResult := range trialSampleStream {
		if sampleResult.Err != nil {
			return status.Errorf(codes.Internal, "ActivityLoggerServer.ListenToTrial: internal error %q", sampleResult.Err)
		}
		err = stream.Send(&grpcapi.ListenToTrialReply{Msg: &grpcapi.ListenToTrialReply_Sample{Sample: sampleResult.Ok}})
		if err != nil {
			return err
		}
	}

	return nil
}

// RegisterActivityLoggerServer registers an ActivityLoggerServer to a gRPC server.
func RegisterActivityLoggerServer(grpcServer grpc.ServiceRegistrar, backend backend.Backend) error {
	server := &activityLoggerServer{
		backend: backend,
	}

	grpcapi.RegisterActivityLoggerServer(grpcServer, server)
	return nil
}
