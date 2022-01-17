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

	"github.com/cogment/cogment-trial-datastore/backend"
	grpcapi "github.com/cogment/cogment-trial-datastore/grpcapi/cogment/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type datalogServer struct {
	grpcapi.UnimplementedDatalogSPServer
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

func trialSampleFromDatalogSample(trialID string, actorIndices map[string]uint32, datalogSample *grpcapi.DatalogSample) (*grpcapi.StoredTrialSample, error) {
	// Base stuffs
	sample := grpcapi.StoredTrialSample{
		UserId:       "",
		TrialId:      trialID,
		TickId:       datalogSample.Info.TickId,
		Timestamp:    datalogSample.Info.Timestamp,
		State:        datalogSample.Info.State,
		ActorSamples: make([]*grpcapi.StoredTrialActorSample, len(actorIndices)),
		Payloads:     make([][]byte, 0),
	}
	// Initializing the actor samples
	for actorIdx := range sample.ActorSamples {
		sample.ActorSamples[actorIdx] = &grpcapi.StoredTrialActorSample{
			Actor: uint32(actorIdx),
		}
	}
	// Deal with the observations
	if datalogSample.Observations != nil {
		payloadIdxFromObsIdx := make([]uint32, len(datalogSample.Observations.Observations))
		for obsIdx, obsData := range datalogSample.Observations.Observations {
			payload := obsData
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
				receivedReward := grpcapi.StoredTrialActorSampleReward{
					Sender:     senderActorIdx,
					Reward:     sourceReward.Value,
					Confidence: sourceReward.Confidence,
					UserData:   payloadIdx,
				}
				sample.ActorSamples[actorIdx].ReceivedRewards = append(sample.ActorSamples[actorIdx].ReceivedRewards, &receivedReward)

				if senderActorIdx >= 0 {
					sentReward := grpcapi.StoredTrialActorSampleReward{
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
				receivedMessage := grpcapi.StoredTrialActorSampleMessage{
					Sender:  senderActorIdx,
					Payload: payloadIdx,
				}
				sample.ActorSamples[receiverActorIdx].ReceivedMessages = append(sample.ActorSamples[receiverActorIdx].ReceivedMessages, &receivedMessage)
			}

			if senderActorIdx >= 0 {
				sentMessage := grpcapi.StoredTrialActorSampleMessage{
					Receiver: receiverActorIdx,
					Payload:  payloadIdx,
				}
				sample.ActorSamples[senderActorIdx].SentMessages = append(sample.ActorSamples[senderActorIdx].SentMessages, &sentMessage)
			}
		}
	}

	return &sample, nil
}

func (s *datalogServer) RunTrialDatalog(stream grpcapi.DatalogSP_RunTrialDatalogServer) error {
	ctx := stream.Context()
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Errorf(codes.DataLoss, "DatalogServer.RunTrialDatalog: failed to get metadata")
	}
	trialIDs := md.Get("trial-id")
	if len(trialIDs) == 0 {
		return status.Errorf(codes.InvalidArgument, "DatalogServer.RunTrialDatalog: missing required header \"trial-id\"")
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
		return status.Errorf(codes.InvalidArgument, "DatalogServer.RunTrialDatalog: initial message is not of type \"cogment.TrialParams\"")
	}
	err = s.backend.CreateOrUpdateTrials(ctx, []*backend.TrialParams{{TrialID: trialID, UserID: "log exporter server", Params: trialParams}})
	if err != nil {
		return status.Errorf(codes.Internal, "DatalogServer.RunTrialDatalog: internal error %q", err)
	}

	// Acknowledge the handling of the first "params" message
	err = stream.Send(&grpcapi.RunTrialDatalogOutput{})
	if err != nil {
		return err
	}

	for actorIdx, actorConfig := range trialParams.GetActors() {
		actorIndices[actorConfig.Name] = uint32(actorIdx)
	}

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		sample := req.GetSample()
		if sample == nil {
			return status.Errorf(codes.InvalidArgument, "DatalogServer.RunTrialDatalog: body message is not of type \"cogment.DatalogSample\"")
		}
		trialSample, err := trialSampleFromDatalogSample(trialID, actorIndices, sample)
		if err != nil {
			return status.Errorf(codes.Internal, "DatalogServer.RunTrialDatalog: internal error %q", err)
		}
		err = s.backend.AddSamples(ctx, []*grpcapi.StoredTrialSample{trialSample})
		if err != nil {
			return status.Errorf(codes.Internal, "DatalogServer.RunTrialDatalog: internal error %q", err)
		}

		// Acknowledge the handling of the following "sample" message
		err = stream.Send(&grpcapi.RunTrialDatalogOutput{})
		if err != nil {
			return err
		}
	}
}

func (s *datalogServer) Version(context.Context, *grpcapi.VersionRequest) (*grpcapi.VersionInfo, error) {
	return nil, status.Errorf(codes.Unimplemented, "DatalogServer.Version not implemented")
}

// RegisterDatalogServer registers a DatalogServer to a gRPC server.
func RegisterDatalogServer(grpcServer grpc.ServiceRegistrar, backend backend.Backend) error {
	server := &datalogServer{
		backend: backend,
	}

	grpcapi.RegisterDatalogSPServer(grpcServer, server)
	return nil
}
