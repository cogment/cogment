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

#ifndef NDEBUG
  #define SPDLOG_ACTIVE_LEVEL SPDLOG_LEVEL_TRACE
#endif

#include "cogment/services/actor_service.h"

#include "cogment/orchestrator.h"
#include "cogment/versions.h"

#include "spdlog/spdlog.h"

#include <string>

namespace cogment {
ActorService::ActorService(Orchestrator* orch) : m_orchestrator(orch) {}

::cogmentAPI::TrialJoinReply ActorService::JoinTrial(::cogmentAPI::TrialJoinRequest req, easy_grpc::Context) {
  SPDLOG_TRACE("JoinTrial [{}]", req.trial_id());
  return m_orchestrator->client_joined(std::move(req));
}

::easy_grpc::Stream_future<cogmentAPI::TrialActionReply> ActorService::ActionStream(
    ::easy_grpc::Stream_future<cogmentAPI::TrialActionRequest> actions, easy_grpc::Context ctx) {
  std::string trial_id(ctx.get_client_header("trial-id"));
  std::string actor_name(ctx.get_client_header("actor-name"));

  SPDLOG_TRACE("ActionStream [{}] [{}]", trial_id, actor_name);
  return m_orchestrator->bind_client(trial_id, actor_name, std::move(actions));
}

::easy_grpc::Future<cogmentAPI::TrialHeartbeatReply> ActorService::Heartbeat(::cogmentAPI::TrialHeartbeatRequest,
                                                                            easy_grpc::Context) {
  return {};
}

::easy_grpc::Future<cogmentAPI::TrialRewardReply> ActorService::SendReward(::cogmentAPI::TrialRewardRequest reward,
                                                                          easy_grpc::Context ctx) {
  std::string trial_id(ctx.get_client_header("trial-id"));
  std::string actor_name(ctx.get_client_header("actor-name"));
  SPDLOG_TRACE("SendReward [{}] [{}]", trial_id, actor_name);

  auto trial = m_orchestrator->get_trial(trial_id);
  if (trial != nullptr) {
    for (auto& rew : reward.rewards()) {
      trial->reward_received(rew, actor_name);
    }
  }
  else {
    spdlog::error("Trial [{}] doesn't exist to send reward", trial_id);
  }

  ::easy_grpc::Promise<cogmentAPI::TrialRewardReply> prom;
  auto result = prom.get_future();
  ::cogmentAPI::TrialRewardReply reply;
  prom.set_value(std::move(reply));

  return result;
}

::easy_grpc::Future<cogmentAPI::TrialMessageReply> ActorService::SendMessage(::cogmentAPI::TrialMessageRequest message,
                                                                            easy_grpc::Context ctx) {
  std::string trial_id(ctx.get_client_header("trial-id"));
  std::string actor_name(ctx.get_client_header("actor-name"));
  SPDLOG_TRACE("SendMessage [{}] [{}]", trial_id, actor_name);

  auto trial = m_orchestrator->get_trial(trial_id);
  if (trial != nullptr) {
    for (auto& mess : message.messages()) {
      trial->message_received(mess, actor_name);
    }
  }
  else {
    spdlog::error("Trial [{}] doesn't exist to send message", trial_id);
  }

  ::easy_grpc::Promise<cogmentAPI::TrialMessageReply> prom;
  auto result = prom.get_future();
  ::cogmentAPI::TrialMessageReply reply;
  prom.set_value(std::move(reply));

  return result;
}

::cogmentAPI::VersionInfo ActorService::Version(::cogmentAPI::VersionRequest, easy_grpc::Context ctx) {
  (void)ctx;
  ::cogmentAPI::VersionInfo result;
  auto v = result.add_versions();

  v->set_name("orchestrator");
  v->set_version(COGMENT_ORCHESTRATOR_VERSION);

  v = result.add_versions();
  v->set_name("cogment-api");
  v->set_version(COGMENT_API_VERSION);

  return result;
}
}  // namespace cogment
