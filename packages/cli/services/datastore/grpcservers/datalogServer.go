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
	"io"
	"strings"

	grpcapi "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/cogment/cogment/services/datastore/backend"
	"github.com/openlyinc/pointy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type datalogServer struct {
	grpcapi.UnimplementedDatalogSPServer
	backend backend.Backend
}

func actorIndexFromActorName(actorName string, actorIndices map[string]uint32) int32 {
	actorIndex, found := actorIndices[actorName]
	if !found {
		// If the actor is not found we consider it is the environment
		// TODO maybe use the actual environment "actor name"
		return -1
	}
	return int32(actorIndex)
}

func trialSampleFromDatalogSample(
	trialID string,
	actorIndices map[string]uint32,
	datalogSample *grpcapi.DatalogSample,
	trialActors []*grpcapi.ActorParams,
) (*grpcapi.StoredTrialSample, error) {
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
	for actorIndex := range sample.ActorSamples {
		sample.ActorSamples[actorIndex] = &grpcapi.StoredTrialActorSample{
			Actor: uint32(actorIndex),
		}
	}
	// Deal with the observations
	if datalogSample.Observations != nil {
		payloadIndexFromObsIndex := make([]uint32, len(datalogSample.Observations.Observations))
		for obsIndex, obsData := range datalogSample.Observations.Observations {
			payload := obsData
			payloadIndexFromObsIndex[obsIndex] = uint32(len(sample.Payloads))
			sample.Payloads = append(sample.Payloads, payload)
		}
		for actorIndex, obsIndex := range datalogSample.Observations.ActorsMap {
			sample.ActorSamples[actorIndex].Observation = &payloadIndexFromObsIndex[obsIndex]
		}
	}
	// Deal with the actions
	{
		for actorIndex, action := range datalogSample.Actions {
			payload := action.Content
			payloadIndex := uint32(len(sample.Payloads))
			sample.Payloads = append(sample.Payloads, payload)
			sample.ActorSamples[actorIndex].Action = &payloadIndex
		}
	}
	// Deal with the rewards
	{
		rewardAccumulator := make(map[uint32][]*grpcapi.RewardSource)

		for _, reward := range datalogSample.Rewards {

			// Potential wildcards in receiver name
			var rewardedActorIndices []uint32
			splitName := strings.Split(reward.ReceiverName, ".")

			if reward.ReceiverName == "*" || reward.ReceiverName == "*.*" {
				for _, index := range actorIndices {
					rewardedActorIndices = append(rewardedActorIndices, index)
				}
			} else if len(splitName) > 1 {
				className := splitName[0]
				actorName := splitName[1]

				for index, actorParam := range trialActors {
					if actorParam.GetActorClass() == className && (actorName == "*" || actorName == actorParam.Name) {
						rewardedActorIndices = append(rewardedActorIndices, uint32(index))
					}
				}
			} else {
				// Find actor sample that matches the reward
				index, exists := actorIndices[reward.ReceiverName]
				if exists {
					rewardedActorIndices = append(rewardedActorIndices, index)
				} else {
					log.WithField("receiver_name", reward.ReceiverName).Debug(
						"Unrecognized receiver name, not logging the reward")
				}
			}

			for _, actorIndex := range rewardedActorIndices {

				// Store the reward sources for aggregation later
				_, exists := rewardAccumulator[actorIndex]
				if !exists {
					rewardAccumulator[actorIndex] = make([]*grpcapi.RewardSource, 0)
				}

				for _, sourceReward := range reward.Sources {
					rewardAccumulator[actorIndex] = append(rewardAccumulator[actorIndex], sourceReward)

					var payloadIndex *uint32
					if sourceReward.UserData != nil && len(sourceReward.UserData.Value) > 0 {
						payloadIndex = pointy.Uint32(uint32(len(sample.Payloads)))
						sample.Payloads = append(sample.Payloads, sourceReward.UserData.Value)
					}

					senderactorIndex := actorIndexFromActorName(sourceReward.SenderName, actorIndices)
					receivedReward := grpcapi.StoredTrialActorSampleReward{
						Sender:     senderactorIndex,
						Reward:     sourceReward.Value,
						Confidence: sourceReward.Confidence,
						UserData:   payloadIndex,
					}
					sample.ActorSamples[actorIndex].ReceivedRewards = append(
						sample.ActorSamples[actorIndex].ReceivedRewards,
						&receivedReward,
					)

					if senderactorIndex >= 0 {
						sentReward := grpcapi.StoredTrialActorSampleReward{
							Receiver:   int32(actorIndex),
							Reward:     sourceReward.Value,
							Confidence: sourceReward.Confidence,
							UserData:   payloadIndex,
						}
						sample.ActorSamples[senderactorIndex].SentRewards = append(
							sample.ActorSamples[senderactorIndex].SentRewards,
							&sentReward,
						)
					}
				}
			}
		}

		for actorIndex, sourcesList := range rewardAccumulator {

			// The total reward for an actor is the average value weighted by the confidence
			valueAccum := float32(0.0)
			confidenceAccum := float32(0.0)

			for _, source := range sourcesList {
				if source.Confidence > 0 {
					valueAccum += source.Value * source.Confidence
					confidenceAccum += source.Confidence
				}
			}

			if confidenceAccum > 0 {
				valueAccum /= confidenceAccum
			}

			sample.ActorSamples[actorIndex].Reward = pointy.Float32(valueAccum)
		}
	}

	// Deal with the messages
	{
		for _, message := range datalogSample.Messages {
			payloadIndex := uint32(len(sample.Payloads))
			sample.Payloads = append(sample.Payloads, message.Payload.Value)

			receiveractorIndex := actorIndexFromActorName(message.ReceiverName, actorIndices)
			senderactorIndex := actorIndexFromActorName(message.SenderName, actorIndices)

			if receiveractorIndex >= 0 {
				receivedMessage := grpcapi.StoredTrialActorSampleMessage{
					Sender:  senderactorIndex,
					Payload: payloadIndex,
				}
				sample.ActorSamples[receiveractorIndex].ReceivedMessages = append(
					sample.ActorSamples[receiveractorIndex].ReceivedMessages,
					&receivedMessage,
				)
			}

			if senderactorIndex >= 0 {
				sentMessage := grpcapi.StoredTrialActorSampleMessage{
					Receiver: receiveractorIndex,
					Payload:  payloadIndex,
				}
				sample.ActorSamples[senderactorIndex].SentMessages = append(
					sample.ActorSamples[senderactorIndex].SentMessages,
					&sentMessage,
				)
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
		return status.Errorf(
			codes.InvalidArgument,
			"DatalogServer.RunTrialDatalog: initial message is not of type \"cogment.TrialParams\"",
		)
	}
	err = s.backend.CreateOrUpdateTrials(
		ctx,
		[]*backend.TrialParams{{TrialID: trialID, UserID: "log exporter server", Params: trialParams}},
	)
	if err != nil {
		return status.Errorf(codes.Internal, "DatalogServer.RunTrialDatalog: internal error %q", err)
	}

	// Acknowledge the handling of the first "params" message
	err = stream.Send(&grpcapi.RunTrialDatalogOutput{})
	if err != nil {
		return err
	}

	for actorIndex, actorConfig := range trialParams.GetActors() {
		actorIndices[actorConfig.Name] = uint32(actorIndex)
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
			return status.Errorf(
				codes.InvalidArgument,
				"DatalogServer.RunTrialDatalog: body message is not of type \"cogment.DatalogSample\"",
			)
		}
		trialSample, err := trialSampleFromDatalogSample(trialID, actorIndices, sample, trialParams.GetActors())
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
	// TODO: Return proper version info. Current version is minimal to serve a health check for directory.
	res := &grpcapi.VersionInfo{}
	return res, nil
}

// gRPC interface
func (s *datalogServer) Status(_ context.Context, request *grpcapi.StatusRequest) (*grpcapi.StatusReply, error) {
	reply := grpcapi.StatusReply{}

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

// RegisterDatalogServer registers a DatalogServer to a gRPC server.
func RegisterDatalogServer(grpcServer grpc.ServiceRegistrar, backend backend.Backend) error {
	server := &datalogServer{
		backend: backend,
	}

	grpcapi.RegisterDatalogSPServer(grpcServer, server)
	return nil
}
