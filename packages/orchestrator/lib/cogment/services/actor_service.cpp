// Copyright 2022 AI Redefined Inc. <dev+cogment@ai-r.com>
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

#include "spdlog/spdlog.h"

#include <string>

namespace cogment {

ActorService::ActorService(Orchestrator* orch) : m_orchestrator(orch) {}

grpc::Status ActorService::RunTrial(grpc::ServerContext* ctx, ServerStream::StreamType* stream) {
  SPDLOG_TRACE("ActorService::RunTrial()");

  try {
    auto& metadata = ctx->client_metadata();
    std::string trial_id(OneFromMetadata(metadata, "trial-id"));
    SPDLOG_TRACE("ActorService::RunTrial [{}] [{}]", trial_id, ctx->peer());

    auto trial = m_orchestrator->get_trial(trial_id);
    if (trial == nullptr) {
      throw MakeException("Unknown trial for actor to join");
    }

    ClientActor::run_an_actor(std::move(trial), stream);  // Blocking
  }
  catch (const std::exception& exc) {
    return MakeErrorStatus("ClientActorSP::RunTrial failure: {}", exc.what());
  }
  catch (...) {
    return MakeErrorStatus("ClientActorSP::RunTrial failure");
  }

  return grpc::Status::OK;
}

grpc::Status ActorService::Version(grpc::ServerContext*, const cogmentAPI::VersionRequest*,
                                   cogmentAPI::VersionInfo* out) {
  SPDLOG_TRACE("ActorService::Version()");

  try {
    m_orchestrator->Version(out);
  }
  catch (const std::exception& exc) {
    return MakeErrorStatus("ClientActorSP::Version failure: {}", exc.what());
  }
  catch (...) {
    return MakeErrorStatus("ClientActorSP::Version failure");
  }

  return grpc::Status::OK;
}

}  // namespace cogment
