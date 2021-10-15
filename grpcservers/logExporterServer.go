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
	"fmt"
	"io"

	"github.com/cogment/cogment-trial-datastore/backend"
	grpcapi "github.com/cogment/cogment-trial-datastore/grpcapi/cogment/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type logExporterServer struct {
	grpcapi.UnimplementedLogExporterServer
	backend backend.Backend
}

func actorIdxFromActorName(actorName string, actorIndices map[string]uint32) int32 {
	actorIdx, found := actorIndices[actorName]
	if !found {
		// If the actor is not found we consider it is the environment
		// TODO maybe use the actual environment "actor name"
		return -1
	}
	return int32(actorIdx)
}

func trialSampleFromDatalogSample(trialID string, actorIndices map[string]uint32, datalogSample *grpcapi.DatalogSample) (*grpcapi.TrialSample, error) {
	// Base stuffs
	sample := grpcapi.TrialSample{
		UserId:       datalogSample.UserId,
		TrialId:      trialID,
		TickId:       datalogSample.TrialData.TickId,
		Timestamp:    datalogSample.TrialData.Timestamp,
		State:        datalogSample.TrialData.State,
		ActorSamples: make([]*grpcapi.TrialActorSample, len(actorIndices)),
		Payloads:     make([][]byte, 0),
	}
	// Initializing the actor samples
	for actorIdx, actorSample := range sample.ActorSamples {
		actorSample.Actor = uint32(actorIdx)
	}
	// Deal with the observations
	{
		payloadIdxFromObsIdx := make([]uint32, len(datalogSample.Observations.Observations))
		for obsIdx, obsData := range datalogSample.Observations.Observations {
			if !obsData.Snapshot {
				return nil, fmt.Errorf("No support for delta observations")
			}
			payload := obsData.Content
			payloadIdxFromObsIdx[obsIdx] = uint32(len(sample.Payloads))
			sample.Payloads = append(sample.Payloads, payload)
		}
		for actorIdx, obsIdx := range datalogSample.Observations.ActorsMap {
			sample.ActorSamples[actorIdx].Observation = &payloadIdxFromObsIdx[obsIdx]
		}
	}
	// Deal with the actions
	{
		for actorIdx, action := range datalogSample.Actions {
			payload := action.Content
			payloadIdx := uint32(len(sample.Payloads))
			sample.Payloads = append(sample.Payloads, payload)
			sample.ActorSamples[actorIdx].Action = &payloadIdx
		}
	}
	// Deal with the rewards
	{
		for actorIdx, reward := range datalogSample.Rewards {
			sample.ActorSamples[actorIdx].Reward = &reward.Value
			for _, sourceReward := range reward.Sources {
				var payloadIdx *uint32
				if sourceReward.UserData != nil && len(sourceReward.UserData.Value) > 0 {
					*payloadIdx = uint32(len(sample.Payloads))
					sample.Payloads = append(sample.Payloads, sourceReward.UserData.Value)
				}

				senderActorIdx := actorIdxFromActorName(sourceReward.SenderName, actorIndices)
				receivedReward := grpcapi.TrialActorSampleReward{
					Sender:     senderActorIdx,
					Reward:     sourceReward.Value,
					Confidence: sourceReward.Confidence,
					UserData:   payloadIdx,
				}
				sample.ActorSamples[actorIdx].ReceivedRewards = append(sample.ActorSamples[actorIdx].ReceivedRewards, &receivedReward)

				if senderActorIdx >= 0 {
					sentReward := grpcapi.TrialActorSampleReward{
						Receiver:   int32(actorIdx),
						Reward:     sourceReward.Value,
						Confidence: sourceReward.Confidence,
						UserData:   payloadIdx,
					}
					sample.ActorSamples[senderActorIdx].SentRewards = append(sample.ActorSamples[senderActorIdx].SentRewards, &sentReward)
				}
			}
		}
	}

	// Deal with the messages
	{
		for _, message := range datalogSample.Messages {
			payloadIdx := uint32(len(sample.Payloads))
			sample.Payloads = append(sample.Payloads, message.Payload.Value)

			receiverActorIdx := actorIdxFromActorName(message.ReceiverName, actorIndices)
			senderActorIdx := actorIdxFromActorName(message.SenderName, actorIndices)

			if receiverActorIdx >= 0 {
				receivedMessage := grpcapi.TrialActorSampleMessage{
					Sender:  senderActorIdx,
					Payload: payloadIdx,
				}
				sample.ActorSamples[receiverActorIdx].ReceivedMessages = append(sample.ActorSamples[receiverActorIdx].ReceivedMessages, &receivedMessage)
			}

			if senderActorIdx >= 0 {
				sentMessage := grpcapi.TrialActorSampleMessage{
					Receiver: receiverActorIdx,
					Payload:  payloadIdx,
				}
				sample.ActorSamples[senderActorIdx].SentMessages = append(sample.ActorSamples[senderActorIdx].SentMessages, &sentMessage)
			}
		}
	}

	return &sample, nil
}

func (s *logExporterServer) OnLogSample(stream grpcapi.LogExporter_OnLogSampleServer) error {
	ctx := stream.Context()
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Errorf(codes.DataLoss, "LogExporterServer.OnLogSample: failed to get metadata")
	}
	trialIDs := md.Get("trial-id")
	if len(trialIDs) == 0 {
		return status.Errorf(codes.InvalidArgument, "LogExporterServer.OnLogSample: missing required header \"trial-id\"")
	}
	trialID := trialIDs[0]
	actorIndices := make(map[string]uint32)
	// Receive the first element, it should be trial data
	req, err := stream.Recv()
	if err != nil {
		return err
	}
	trialParams := req.GetTrialParams()
	if trialParams == nil {
		return status.Errorf(codes.InvalidArgument, "LogExporterServer.OnLogSample: initial message is not of type \"cogment.TrialParams\"")
	}
	err = s.backend.CreateOrUpdateTrials(ctx, []*backend.TrialParams{{TrialID: trialID, UserID: "log exporter server", Params: trialParams}})
	if err != nil {
		return status.Errorf(codes.Internal, "LogExporterServer.OnLogSample: internal error %q", err)
	}
	for actorIdx, actorConfig := range trialParams.GetActors() {
		actorIndices[actorConfig.Name] = uint32(actorIdx)
	}

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return stream.Send(&grpcapi.LogExporterSampleReply{})
		}
		if err != nil {
			return err
		}
		sample := req.GetSample()
		if sample == nil {
			return status.Errorf(codes.InvalidArgument, "LogExporterServer.OnLogSample: body message is not of type \"cogment.DatalogSample\"")
		}
		trialSample, err := trialSampleFromDatalogSample(trialID, actorIndices, sample)
		if err != nil {
			return status.Errorf(codes.Internal, "LogExporterServer.OnLogSample: internal error %q", err)
		}
		err = s.backend.AddSamples(ctx, []*grpcapi.TrialSample{trialSample})
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
	return nil, status.Errorf(codes.Unimplemented, "LogExporterServer.Version not implemented")
}

// RegisterLogExporterServer registers a LogExporterServer to a gRPC server.
func RegisterLogExporterServer(grpcServer grpc.ServiceRegistrar, backend backend.Backend) error {
	server := &logExporterServer{
		backend: backend,
	}

	grpcapi.RegisterLogExporterServer(grpcServer, server)
	return nil
}
