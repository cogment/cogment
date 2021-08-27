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

grpc::Status ActorService::JoinTrial(grpc::ServerContext*, const cogmentAPI::TrialJoinRequest* in, cogmentAPI::TrialJoinReply* out) {
  SPDLOG_TRACE("JoinTrial()");

  try {
    SPDLOG_TRACE("JoinTrial [{}]", in->DebugString());
    *out = m_orchestrator->client_joined(*in);
  }
  catch(const std::exception& exc) {
    return MakeErrorStatus("JoinTrial failure on exception [%s]", exc.what());
  }
  catch(...) {
    return MakeErrorStatus("JoinTrial failure on unknown exception");
  }

  return grpc::Status::OK;
}

grpc::Status ActorService::ActionStream(grpc::ServerContext* ctx, grpc::ServerReaderWriter<cogmentAPI::TrialActionReply, cogmentAPI::TrialActionRequest>* stream) {
  SPDLOG_TRACE("ActionStream()");

  try {
    auto& metadata = ctx->client_metadata();
    std::string trial_id(OneFromMetadata(metadata, "trial-id"));
    std::string actor_name(OneFromMetadata(metadata, "actor-name"));
    SPDLOG_TRACE("ActionStream [{}] [{}] [{}]", trial_id, actor_name, ctx->peer());

    return m_orchestrator->bind_client(trial_id, actor_name, stream);  // Blocking
  }
  catch(const std::exception& exc) {
    return MakeErrorStatus("ActionStream failure on exception [%s]", exc.what());
  }
  catch(...) {
    return MakeErrorStatus("ActionStream failure on unknown exception");
  }
}

grpc::Status ActorService::Heartbeat(grpc::ServerContext* ctx, const cogmentAPI::TrialHeartbeatRequest* in, cogmentAPI::TrialHeartbeatReply* out) {
  return grpc::Status(grpc::StatusCode::UNIMPLEMENTED, "Heartbeat()");
}

grpc::Status ActorService::SendReward(grpc::ServerContext* ctx, const cogmentAPI::TrialRewardRequest* in, cogmentAPI::TrialRewardReply* out) {
  SPDLOG_TRACE("SendReward()");

  try {
    auto& metadata = ctx->client_metadata();
    std::string trial_id(OneFromMetadata(metadata, "trial-id"));
    std::string actor_name(OneFromMetadata(metadata, "actor-name"));
    SPDLOG_TRACE("SendReward [{}] [{}]", trial_id, actor_name);

    auto trial = m_orchestrator->get_trial(trial_id);
    if (trial != nullptr) {
      for (auto& rew : in->rewards()) {
        trial->reward_received(rew, actor_name);
      }
    }
    else {
      return MakeErrorStatus("SendReward could not find trial [%s]", trial_id.c_str());
    }

    out->Clear();
  }
  catch(const std::exception& exc) {
    return MakeErrorStatus("SendReward failure on exception [%s]", exc.what());
  }
  catch(...) {
    return MakeErrorStatus("SendReward failure on unknown exception");
  }

  return grpc::Status::OK;
}

grpc::Status ActorService::SendMessage(grpc::ServerContext* ctx, const cogmentAPI::TrialMessageRequest* in, cogmentAPI::TrialMessageReply* out) {
  SPDLOG_TRACE("SendMessage()");

  try {
    auto& metadata = ctx->client_metadata();
    std::string trial_id(OneFromMetadata(metadata, "trial-id"));
    std::string actor_name(OneFromMetadata(metadata, "actor-name"));
    SPDLOG_TRACE("SendMessage [{}] [{}]", trial_id, actor_name);

    auto trial = m_orchestrator->get_trial(trial_id);
    if (trial != nullptr) {
      for (auto& mess : in->messages()) {
        trial->message_received(mess, actor_name);
      }
    }
    else {
      return MakeErrorStatus("SendMessage could not find trial [%s]", trial_id.c_str());
    }

    out->Clear();
  }
  catch(const std::exception& exc) {
    return MakeErrorStatus("SendMessage failure on exception [%s]", exc.what());
  }
  catch(...) {
    return MakeErrorStatus("SendMessage failure on unknown exception");
  }

  return grpc::Status::OK;
}

grpc::Status ActorService::Version(grpc::ServerContext*, const cogmentAPI::VersionRequest*, cogmentAPI::VersionInfo* out) {
  SPDLOG_TRACE("Version()");

  try {
    auto ver = out->add_versions();
    ver->set_name("orchestrator");
    ver->set_version(COGMENT_ORCHESTRATOR_VERSION);

    ver = out->add_versions();
    ver->set_name("cogment-api");
    ver->set_version(COGMENT_API_VERSION);
  }
  catch(const std::exception& exc) {
    return MakeErrorStatus("Version failure on exception [%s]", exc.what());
  }
  catch(...) {
    return MakeErrorStatus("Version failure on unknown exception");
  }

  return grpc::Status::OK;
}

}  // namespace cogment
