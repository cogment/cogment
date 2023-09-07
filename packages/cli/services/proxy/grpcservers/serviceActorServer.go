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
	"time"

	grpcapi "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/cogment/cogment/services/proxy/actor"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

type actorServer struct {
	grpcapi.UnimplementedServiceActorSPServer
	actorManager *actor.Manager
}

func RegisterServiceActorServer(grpcServer grpc.ServiceRegistrar, actorManager *actor.Manager) error {
	server := &actorServer{
		actorManager: actorManager,
	}

	grpcapi.RegisterServiceActorSPServer(grpcServer, server)
	return nil
}

func (s *actorServer) Version(context.Context, *grpcapi.VersionRequest) (*grpcapi.VersionInfo, error) {
	// TODO: Return proper version info. Current version is minimal to serve a health check for directory.
	res := &grpcapi.VersionInfo{}
	return res, nil
}

// gRPC interface
func (s *actorServer) Status(_ context.Context, request *grpcapi.StatusRequest) (*grpcapi.StatusReply, error) {
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

func (s *actorServer) RunTrial(stream grpcapi.ServiceActorSP_RunTrialServer) error {
	ctx := stream.Context()
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Errorf(codes.DataLoss, "ServiceActorServer.RunTrial: failed to get metadata")
	}
	trialIDs := md.Get("trial-id")
	if len(trialIDs) == 0 {
		return status.Errorf(codes.InvalidArgument, "ServiceActorServer.RunTrial: missing required header \"trial-id\"")
	}
	trialID := trialIDs[0]
	currentTickID := uint64(0)
	actorName := ""
	actorClass := ""
	log = log.WithField("trial_id", trialID)

	trialInitialized := false
	receivedRewards := []actor.RecvReward{}
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			log.Info("The orchestrator disconnected the actor")
			return nil
		}
		if err != nil {
			return err
		}
		switch in.State {
		case grpcapi.CommunicationState_HEARTBEAT:
			log.Trace("Received 'HEARTBEAT' and responding in kind")
			err := stream.Send(&grpcapi.ActorRunTrialOutput{State: grpcapi.CommunicationState_HEARTBEAT})
			if err != nil {
				return err
			}
		case grpcapi.CommunicationState_LAST:
			log.Trace("Received 'LAST' state")
			err := stream.Send(&grpcapi.ActorRunTrialOutput{State: grpcapi.CommunicationState_LAST_ACK})
			if err != nil {
				return err
			}
		case grpcapi.CommunicationState_LAST_ACK:
			log.Warn("Receiving unexpected 'LAST_ACK' state")
		case grpcapi.CommunicationState_END:
			if details := in.GetDetails(); details != "" {
				log = log.WithField("details", details)
			}
			log.Debug("Trial ending")
			currentTickID++
			err := s.actorManager.EndTrial(
				trialID,
				actorName,
				actor.RecvEvent{
					TickID:  currentTickID,
					Rewards: receivedRewards,
				},
			)
			if err != nil {
				return status.Errorf(
					codes.InvalidArgument,
					"ServiceActorServer.RunTrial: failed to end the trial: %v",
					err,
				)
			}
			// Proper end of the trial
			return nil
		case grpcapi.CommunicationState_NORMAL:
			if !trialInitialized {
				log.Debug("Trial initialization")
				initInput := in.GetInitInput()
				if initInput == nil {
					return status.Errorf(
						codes.InvalidArgument,
						"ServiceActorServer.RunTrial: expect initial message to be of type \"cogment.ActorInitialInput\"",
					)
				}

				actorName = initInput.ActorName
				log = log.WithField("actor_name", actorName)

				actorClass = initInput.ActorClass

				actorConfig, err := s.actorManager.Spec().NewActorConfig(actorClass)
				if err != nil {
					return status.Errorf(
						codes.InvalidArgument,
						"ServiceActorServer.RunTrial: invalid actor class [%s] (%v)",
						actorClass,
						err,
					)
				}

				if err := actorConfig.UnmarshalProto(initInput.Config.Content); err != nil {
					return status.Errorf(
						codes.InvalidArgument,
						"ServiceActorServer.RunTrial: error while deserializing the actor config (%v)",
						err,
					)
				}

				err = s.actorManager.StartTrial(
					ctx,
					trialID,
					actorName,
					actorClass,
					initInput.ImplName,
					actorConfig,
				)
				if err != nil {
					return status.Errorf(
						codes.InvalidArgument,
						"ServiceActorServer.RunTrial: failed to initialize trial (%v)",
						err,
					)
				}

				trialInitialized = true

				err = stream.Send(&grpcapi.ActorRunTrialOutput{
					State: grpcapi.CommunicationState_NORMAL,
					Data:  &grpcapi.ActorRunTrialOutput_InitOutput{InitOutput: &grpcapi.ActorInitialOutput{}},
				})
				if err != nil {
					return err
				}
			} else if observation := in.GetObservation(); observation != nil {
				log.Trace("Receive observation")

				currentTickID = observation.TickId
				recvEvent := actor.RecvEvent{
					TickID:  currentTickID,
					Rewards: receivedRewards,
				}
				recvEvent.Observation, err = s.actorManager.Spec().NewObservation(actorClass)
				if err != nil {
					return status.Errorf(
						codes.Internal,
						"ServiceActorServer.RunTrial: invalid actor class [%s] (%v)",
						actorClass,
						err,
					)
				}

				if err := recvEvent.Observation.UnmarshalProto(observation.Content); err != nil {
					return status.Errorf(
						codes.InvalidArgument,
						"ServiceActorServer.RunTrial: error while deserializing the observation (%v)",
						err,
					)
				}

				sentEvent, err := s.actorManager.RequestAct(
					ctx,
					trialID,
					actorName,
					recvEvent,
				)
				if err != nil {
					return status.Errorf(
						codes.InvalidArgument,
						"ServiceActorServer.RunTrial: failed to process observation: %v",
						err,
					)
				}
				// Rewards sent properly, we can reset the list
				receivedRewards = []actor.RecvReward{}

				serializedAction, err := sentEvent.Action.MarshalProto()
				if err != nil {
					return status.Errorf(
						codes.InvalidArgument,
						"ServiceActorServer.RunTrial: error while serializing the action (%v)",
						err,
					)
				}

				action := grpcapi.Action{
					TickId:    int64(sentEvent.TickID),
					Timestamp: uint64(time.Now().UnixNano()),
					Content:   serializedAction,
				}

				err = stream.Send(&grpcapi.ActorRunTrialOutput{
					State: grpcapi.CommunicationState_NORMAL,
					Data:  &grpcapi.ActorRunTrialOutput_Action{Action: &action},
				})
				if err != nil {
					return err
				}

				for _, sentReward := range sentEvent.Rewards {
					log.WithFields(logrus.Fields{
						"tick_id":  sentReward.TickID,
						"receiver": sentReward.Receiver,
					}).Trace("Send reward")
					rewardSource := &grpcapi.RewardSource{
						SenderName: actorName,
						Value:      sentReward.Value,
						Confidence: sentReward.Confidence,
					}
					if sentReward.UserData != nil {
						userDataStructPb, err := structpb.NewValue(sentReward.UserData)
						if err != nil {
							return fmt.Errorf("Error while serializing sent reward user data: %w", err)
						}
						userDataAnyPb, err := anypb.New(userDataStructPb)
						if err != nil {
							return fmt.Errorf("Error while serializing sent reward user data: %w", err)
						}
						rewardSource.UserData = userDataAnyPb
					}
					reward := &grpcapi.Reward{
						TickId:       sentReward.TickID,
						ReceiverName: sentReward.Receiver,
						Sources:      []*grpcapi.RewardSource{rewardSource},
					}
					err = stream.Send(&grpcapi.ActorRunTrialOutput{
						State: grpcapi.CommunicationState_NORMAL,
						Data:  &grpcapi.ActorRunTrialOutput_Reward{Reward: reward},
					})
					if err != nil {
						return err
					}
				}
			} else if reward := in.GetReward(); reward != nil {
				receivedReward := actor.RecvReward{
					TickID: uint64(reward.TickId),
					Value:  reward.Value,
				}
				for _, rewardSource := range reward.Sources {
					log.WithFields(logrus.Fields{
						"tick_id": reward.TickId,
						"sender":  rewardSource.SenderName,
					}).Trace("Receive reward")
					receivedRewardSource := actor.RecvRewardSource{
						Sender:     rewardSource.SenderName,
						Value:      rewardSource.Value,
						Confidence: rewardSource.Confidence,
					}
					if rewardSource.UserData != nil {
						receivedRewardSource.UserData, err = s.actorManager.Spec().NewMessage(
							string(rewardSource.UserData.MessageName().Name()),
						)
						if err != nil {
							return fmt.Errorf("Error while deserializing received reward user data: %w", err)
						}

						err = rewardSource.UserData.UnmarshalTo(rewardSource.UserData)
						if err != nil {
							return fmt.Errorf("Error while deserializing received reward user data: %w", err)
						}
					}
					receivedReward.Sources = append(receivedReward.Sources, receivedRewardSource)
				}
				receivedRewards = append(receivedRewards, receivedReward)
			} else {
				log.Tracef("Got unexpected data [%#v]", in.Data)
			}
		}
	}
}
