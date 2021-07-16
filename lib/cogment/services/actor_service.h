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

#ifndef COGMENT_ORCHESTRATOR_ACTOR_SERVICE_H
#define COGMENT_ORCHESTRATOR_ACTOR_SERVICE_H

#include "cogment/api/orchestrator.egrpc.pb.h"

namespace cogment {
class Orchestrator;
class ActorService {
  Orchestrator* m_orchestrator;

public:
  using service_type = cogmentAPI::ClientActorSP;

  ActorService(Orchestrator* orch);

  ::cogmentAPI::TrialJoinReply JoinTrial(::cogmentAPI::TrialJoinRequest, easy_grpc::Context ctx);

  ::easy_grpc::Stream_future<cogmentAPI::TrialActionReply> ActionStream(
      ::easy_grpc::Stream_future<cogmentAPI::TrialActionRequest>, easy_grpc::Context ctx);
  ::easy_grpc::Future<cogmentAPI::TrialHeartbeatReply> Heartbeat(::cogmentAPI::TrialHeartbeatRequest,
                                                                easy_grpc::Context ctx);
  ::easy_grpc::Future<cogmentAPI::TrialRewardReply> SendReward(::cogmentAPI::TrialRewardRequest, easy_grpc::Context ctx);
  ::easy_grpc::Future<cogmentAPI::TrialMessageReply> SendMessage(::cogmentAPI::TrialMessageRequest, easy_grpc::Context ctx);

  ::cogmentAPI::VersionInfo Version(::cogmentAPI::VersionRequest, easy_grpc::Context ctx);
};
}  // namespace cogment

#endif