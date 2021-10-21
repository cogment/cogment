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

grpc::Status ActorService::RunTrial(
    grpc::ServerContext* ctx,
    grpc::ServerReaderWriter<cogmentAPI::ActorRunTrialInput, cogmentAPI::ActorRunTrialOutput>* stream) {
  SPDLOG_TRACE("ActorService::RunTrial()");

  try {
    auto& metadata = ctx->client_metadata();
    std::string trial_id(OneFromMetadata(metadata, "trial-id"));
    SPDLOG_TRACE("ActorService::RunTrial [{}] [{}]", trial_id, ctx->peer());

    auto trial = m_orchestrator->get_trial(trial_id);
    if (trial == nullptr) {
      throw MakeException("Trial [%s] - Unknown trial for actor to join", trial_id.c_str());
    }

    std::weak_ptr<Trial> ptr(trial);
    trial.reset();
    return ClientActor::run_an_actor(ptr, stream);  // Blocking
  }
  catch (const std::exception& exc) {
    return MakeErrorStatus("ActorService::RunTrial failure on exception: %s", exc.what());
  }
  catch (...) {
    return MakeErrorStatus("ActorService::RunTrial failure on unknown exception");
  }
}

grpc::Status ActorService::Version(grpc::ServerContext*, const cogmentAPI::VersionRequest*,
                                   cogmentAPI::VersionInfo* out) {
  SPDLOG_TRACE("Version()");

  try {
    m_orchestrator->Version(out);
  }
  catch (const std::exception& exc) {
    return MakeErrorStatus("Version failure on exception [%s]", exc.what());
  }
  catch (...) {
    return MakeErrorStatus("Version failure on unknown exception");
  }

  return grpc::Status::OK;
}

}  // namespace cogment
