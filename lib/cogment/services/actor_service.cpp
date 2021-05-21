// Copyright 2021 Artificial Intelligence Redefined <dev+cogment@ai-r.com>
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

#ifndef NDEBUG
  #define SPDLOG_ACTIVE_LEVEL SPDLOG_LEVEL_TRACE
#endif

#include "cogment/services/actor_service.h"

#include "cogment/orchestrator.h"
#include "cogment/versions.h"

#include "spdlog/spdlog.h"

#include <string>

namespace cogment {
ActorService::ActorService(Orchestrator* orch) : orchestrator_(orch) {}

::cogment::TrialJoinReply ActorService::JoinTrial(::cogment::TrialJoinRequest req, easy_grpc::Context) {
  SPDLOG_TRACE("JoinTrial [{}]", req.trial_id());
  return orchestrator_->client_joined(std::move(req));
}

::easy_grpc::Stream_future<::cogment::TrialActionReply> ActorService::ActionStream(
    ::easy_grpc::Stream_future<::cogment::TrialActionRequest> actions, easy_grpc::Context ctx) {
  auto trial_id = uuids::uuid::from_string(ctx.get_client_header("trial-id"));
  std::string actor_name(ctx.get_client_header("actor-name"));

  SPDLOG_TRACE("ActionStream [{}] [{}]", ctx.get_client_header("trial-id"), actor_name);
  return orchestrator_->bind_client(trial_id, actor_name, std::move(actions));
}

::easy_grpc::Future<::cogment::TrialHeartbeatReply> ActorService::Heartbeat(::cogment::TrialHeartbeatRequest,
                                                                            easy_grpc::Context) {
  return {};
}

::easy_grpc::Future<::cogment::TrialRewardReply> ActorService::SendReward(::cogment::TrialRewardRequest reward,
                                                                          easy_grpc::Context ctx) {
  auto trial_id = uuids::uuid::from_string(ctx.get_client_header("trial-id"));
  std::string actor_name(ctx.get_client_header("actor-name"));
  SPDLOG_TRACE("SendReward [{}] [{}]", ctx.get_client_header("trial-id"), actor_name);

  auto trial = orchestrator_->get_trial(trial_id);
  if (trial != nullptr) {
    for (auto& rew : reward.rewards()) {
      trial->reward_received(rew, actor_name);
    }
  }
  else {
    spdlog::error("Trial [{}] doesn't exist to send reward", to_string(trial_id));
  }

  ::easy_grpc::Promise<::cogment::TrialRewardReply> prom;
  auto result = prom.get_future();
  ::cogment::TrialRewardReply reply;
  prom.set_value(std::move(reply));

  return result;
}

::easy_grpc::Future<::cogment::TrialMessageReply> ActorService::SendMessage(::cogment::TrialMessageRequest message,
                                                                            easy_grpc::Context ctx) {
  auto trial_id = uuids::uuid::from_string(ctx.get_client_header("trial-id"));
  std::string actor_name(ctx.get_client_header("actor-name"));
  SPDLOG_TRACE("SendMessage [{}] [{}]", ctx.get_client_header("trial-id"), actor_name);

  auto trial = orchestrator_->get_trial(trial_id);
  if (trial != nullptr) {
    for (auto& mess : message.messages()) {
      trial->message_received(mess, actor_name);
    }
  }
  else {
    spdlog::error("Trial [{}] doesn't exist to send message", to_string(trial_id));
  }

  ::easy_grpc::Promise<::cogment::TrialMessageReply> prom;
  auto result = prom.get_future();
  ::cogment::TrialMessageReply reply;
  prom.set_value(std::move(reply));

  return result;
}

::cogment::VersionInfo ActorService::Version(::cogment::VersionRequest, easy_grpc::Context ctx) {
  (void)ctx;
  ::cogment::VersionInfo result;
  auto v = result.add_versions();

  v->set_name("orchestrator");
  v->set_version(COGMENT_ORCHESTRATOR_VERSION);

  v = result.add_versions();
  v->set_name("cogment-api");
  v->set_version(COGMENT_API_VERSION);

  return result;
}
}  // namespace cogment
