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
	"errors"
	"io"
	"strconv"
	"time"

	grpcapi "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/cogment/cogment/services/trialDatastore/backend"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type trialDatastoreServer struct {
	grpcapi.UnimplementedTrialDatastoreSPServer
	backend            backend.Backend
	addSampleChunkSize int
}

func (s *trialDatastoreServer) Version(
	ctx context.Context,
	req *grpcapi.VersionRequest,
) (*grpcapi.VersionInfo, error) {

	// TODO: Return proper version info. Current version is minimal to serve a health check for directory.
	res := &grpcapi.VersionInfo{}
	return res, nil
}

func (s *trialDatastoreServer) RetrieveTrials(
	ctx context.Context,
	req *grpcapi.RetrieveTrialsRequest,
) (*grpcapi.RetrieveTrialsReply, error) {
	pageOffset := 0
	if req.TrialHandle != "" {
		var err error
		pageOffset, err = strconv.Atoi(req.TrialHandle)
		if err != nil {
			return nil, status.Errorf(
				codes.InvalidArgument,
				"Invalid value for `page_handle` (%q) only empty or values provided by a previous call should be used",
				req.TrialHandle,
			)
		}
	}

	trialIds := make([]string, 0, req.TrialsCount)
	trialInfos := make([]*backend.TrialInfo, 0, req.TrialsCount)
	nextPageOffset := 0

	trialFilter := backend.NewTrialFilter(req.TrialIds, req.Properties)
	trialCount := int(req.TrialsCount)
	if len(req.TrialIds) > 0 && len(req.TrialIds) < trialCount {
		trialCount = len(req.TrialIds)
	}

	// 1 - Retrieve the trialIds and trialInfos
	if req.Timeout > 0 {
		ctx, cancelCtx := context.WithTimeout(ctx, time.Duration(req.Timeout)*time.Millisecond)
		defer cancelCtx()

		observer := make(backend.TrialsInfoObserver)

		g, ctx := errgroup.WithContext(ctx)
		g.Go(func() error {
			defer close(observer)
			return s.backend.ObserveTrials(ctx, trialFilter, pageOffset, trialCount, observer)
		})
		g.Go(func() error {
			for trialInfoResult := range observer {
				for _, trialInfo := range trialInfoResult.TrialInfos {
					trialIds = append(trialIds, trialInfo.TrialID)
					trialInfos = append(trialInfos, trialInfo)
				}
				nextPageOffset = trialInfoResult.NextTrialIdx
			}
			return nil
		})

		if err := g.Wait(); err != nil && !errors.Is(err, context.DeadlineExceeded) {
			// context.DeadlineExceeded errors means the timeout we allocated to retrieve the trials is exceeded
			return nil, status.Errorf(codes.Internal, "TrialDatastoreSPServer.ObserveSamples: internal error %q", err)
		}
	} else {
		results, err := s.backend.RetrieveTrials(ctx, trialFilter, pageOffset, int(req.TrialsCount))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "TrialDatastoreSPServer.ObserveSamples: internal error %q", err)
		}
		for _, trialInfo := range results.TrialInfos {
			trialIds = append(trialIds, trialInfo.TrialID)
			trialInfos = append(trialInfos, trialInfo)
		}
		nextPageOffset = results.NextTrialIdx
	}

	// 2 - Retrieve the params
	{
		params, err := s.backend.GetTrialParams(ctx, trialIds)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "TrialDatastoreSPServer.ObserveSamples: internal error %q", err)
		}

		nextTrialHandle := strconv.Itoa(nextPageOffset)

		res := &grpcapi.RetrieveTrialsReply{
			TrialInfos:      make([]*grpcapi.StoredTrialInfo, len(trialInfos)),
			NextTrialHandle: nextTrialHandle,
		}
		for trialInfoIdx, trialInfo := range trialInfos {
			res.TrialInfos[trialInfoIdx] = &grpcapi.StoredTrialInfo{
				TrialId:      trialInfo.TrialID,
				UserId:       trialInfo.UserID,
				LastState:    trialInfo.State,
				SamplesCount: uint32(trialInfo.SamplesCount),
				Params:       params[trialInfoIdx].Params,
			}
		}

		return res, nil
	}
}

func (s *trialDatastoreServer) RetrieveSamples(
	req *grpcapi.RetrieveSamplesRequest,
	resStream grpcapi.TrialDatastoreSP_RetrieveSamplesServer,
) error {
	filter := backend.TrialSampleFilter{
		TrialIDs:             req.TrialIds,
		ActorNames:           req.ActorNames,
		ActorClasses:         req.ActorClasses,
		ActorImplementations: req.ActorImplementations,
		Fields:               req.SelectedSampleFields,
	}
	observer := make(backend.TrialSampleObserver)
	g, ctx := errgroup.WithContext(resStream.Context())
	g.Go(func() error {
		defer close(observer)
		return s.backend.ObserveSamples(ctx, filter, observer)
	})
	g.Go(func() error {
		for sampleResult := range observer {
			err := resStream.Send(&grpcapi.RetrieveSampleReply{TrialSample: sampleResult})
			if err != nil {
				return err
			}
		}
		return nil
	})
	return g.Wait()
}

func trialIDFromHeaderMetadata(ctx context.Context) (string, error) {
	headerMD, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Errorf(
			codes.InvalidArgument,
			"Missing expected 'trial-id' header metadata defining the id of the trial to add",
		)
	}
	trialIDs, ok := headerMD["trial-id"]
	if !ok || len(trialIDs) != 1 {
		return "", status.Errorf(
			codes.InvalidArgument,
			"Missing expected 'trial-id' header metadata defining the id of the trial to add",
		)
	}
	return trialIDs[0], nil
}

func (s *trialDatastoreServer) AddTrial(
	ctx context.Context,
	req *grpcapi.AddTrialRequest,
) (*grpcapi.AddTrialReply, error) {
	trialID, err := trialIDFromHeaderMetadata(ctx)
	if err != nil {
		return nil, err
	}
	err = s.backend.CreateOrUpdateTrials(ctx, []*backend.TrialParams{
		{
			TrialID: trialID,
			UserID:  req.UserId,
			Params:  req.TrialParams,
		},
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "TrialDatastoreSPServer.ObserveSamples: internal error %q", err)
	}
	return &grpcapi.AddTrialReply{}, nil
}

func (s *trialDatastoreServer) AddSample(stream grpcapi.TrialDatastoreSP_AddSampleServer) error {
	ctx := stream.Context()
	trialID, err := trialIDFromHeaderMetadata(ctx)
	if err != nil {
		return err
	}

	samplesChunk := make([]*grpcapi.StoredTrialSample, 0, s.addSampleChunkSize)
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if req.TrialSample.TrialId != "" && req.TrialSample.TrialId != trialID {
			return status.Errorf(
				codes.InvalidArgument,
				"'AddSampleRequest.TrialSample.trial_id' should be left undefined "+
					"or should match the header metadata 'trial-id'",
			)
		}
		req.TrialSample.TrialId = trialID
		samplesChunk = append(samplesChunk, req.TrialSample)
		if len(samplesChunk) == s.addSampleChunkSize {
			err = s.backend.AddSamples(ctx, samplesChunk)
			if err != nil {
				return status.Errorf(codes.Internal, "TrialDatastoreSPServer.ObserveSamples: internal error %q", err)
			}
			samplesChunk = samplesChunk[:0] // Empty the slice while preserving allocated space
		}
	}

	if len(samplesChunk) > 0 {
		err := s.backend.AddSamples(ctx, samplesChunk)
		if err != nil {
			return status.Errorf(codes.Internal, "TrialDatastoreSPServer.ObserveSamples: internal error %q", err)
		}
	}

	return stream.SendAndClose(&grpcapi.AddSamplesReply{})
}

func (s *trialDatastoreServer) DeleteTrials(
	ctx context.Context,
	req *grpcapi.DeleteTrialsRequest,
) (*grpcapi.DeleteTrialsReply, error) {
	err := s.backend.DeleteTrials(ctx, req.TrialIds)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "TrialDatastoreSPServer.DeleteTrials: internal error %q", err)
	}
	return &grpcapi.DeleteTrialsReply{}, nil
}

// RegisterTrialDatastoreServer registers an TrialDatastoreSPServer to a gRPC server.
func RegisterTrialDatastoreServer(grpcServer grpc.ServiceRegistrar, backend backend.Backend) error {
	server := &trialDatastoreServer{
		backend:            backend,
		addSampleChunkSize: 100,
	}

	grpcapi.RegisterTrialDatastoreSPServer(grpcServer, server)
	return nil
}
