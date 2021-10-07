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

#include "cogment/services/trial_lifecycle_service.h"

#include "cogment/orchestrator.h"
#include "cogment/utils.h"

#include <bitset>

namespace cogment {

TrialLifecycleService::TrialLifecycleService(Orchestrator* orch) : m_orchestrator(orch) {}

grpc::Status TrialLifecycleService::StartTrial(grpc::ServerContext*, const cogmentAPI::TrialStartRequest* in, cogmentAPI::TrialStartReply* out) {
  SPDLOG_TRACE("StartTrial()");

  try {
    SPDLOG_TRACE("StartTrial from [{}]", in->user_id());
    auto params = m_orchestrator->default_trial_params();

    // Apply config override if provided
    if (in->has_config()) {
      params.mutable_trial_config()->set_content(in->config().content());
    }

    auto trial = m_orchestrator->start_trial(std::move(params), in->user_id());

    out->set_trial_id(trial->id());
  }
  catch(const std::exception& exc) {
    return MakeErrorStatus("StartTrial failure on exception [%s]", exc.what());
  }
  catch(...) {
    return MakeErrorStatus("StartTrial failure on unknown exception");
  }

  return grpc::Status::OK;
}

grpc::Status TrialLifecycleService::TerminateTrial(grpc::ServerContext* ctx, const cogmentAPI::TerminateTrialRequest* req, cogmentAPI::TerminateTrialReply* out) {
  SPDLOG_TRACE("TerminateTrial()");

  try {
    auto& metadata = ctx->client_metadata();
    std::string trial_id(OneFromMetadata(metadata, "trial-id"));
    SPDLOG_TRACE("TerminateTrial [{}]", trial_id);

    auto trial = m_orchestrator->get_trial(trial_id);
    if (trial != nullptr) {
      if (req->hard_termination()) {
        trial->terminate("Externally requested");
      }
      else {
        trial->request_end();
      }
    }
    else {
      return MakeErrorStatus("Trial [%s] does not exist", trial_id.c_str());
    }

    out->Clear();
  }
  catch(const std::exception& exc) {
    return MakeErrorStatus("TerminateTrial failure on exception [%s]", exc.what());
  }
  catch(...) {
    return MakeErrorStatus("TerminateTrial failure on unknown exception");
  }

  return grpc::Status::OK;
}

grpc::Status TrialLifecycleService::GetTrialInfo(grpc::ServerContext* ctx, const cogmentAPI::TrialInfoRequest* in, cogmentAPI::TrialInfoReply* out) {
  SPDLOG_TRACE("GetTrialInfo()");

  try {
    auto& metadata = ctx->client_metadata();
    auto trial_ids = FromMetadata(metadata, "trial-id");
    SPDLOG_TRACE("GetTrialInfo for [{}] trials", trial_ids.size());

    if (!trial_ids.empty()) {
      for (auto& trial_id : trial_ids) {
        auto trial = m_orchestrator->get_trial(std::string(trial_id));
        if (trial != nullptr) {
          auto trial_info = out->add_trial();
          trial->set_info(trial_info, in->get_latest_observation(), in->get_actor_list());
        }
      }
    }
    else {
      // The user is asking for ALL trials
      auto trials = m_orchestrator->all_trials();
      for (auto& trial : trials) {
        auto trial_info = out->add_trial();
        trial->set_info(trial_info, in->get_latest_observation(), in->get_actor_list());
      }
    }
  }
  catch(const std::exception& exc) {
    return MakeErrorStatus("GetTrialInfo failure on exception [%s]", exc.what());
  }
  catch(...) {
    return MakeErrorStatus("GetTrialInfo failure on unknown exception");
  }

  return grpc::Status::OK;
}

grpc::Status TrialLifecycleService::WatchTrials(grpc::ServerContext*, const cogmentAPI::TrialListRequest* in, grpc::ServerWriter<cogmentAPI::TrialListEntry>* out) {
  SPDLOG_TRACE("WatchTrials()");

  try {
    // Build a bitmask for testing wether or not a trial should be reported.
    std::bitset<cogmentAPI::TrialState_MAX + 1> state_mask;
    if (in->filter_size() == 0) {
      // If filter is empty, we report everything
      state_mask.set();
    }
    else {
      for (auto state : in->filter()) {
        state_mask.set(static_cast<std::size_t>(state));
      }
    }

    // This will get invoked on each reported trial
    auto handler = [state_mask, out](const Trial& trial) {
      auto state = get_trial_api_state(trial.state());

      if (state_mask.test(static_cast<std::size_t>(state))) {
        cogmentAPI::TrialListEntry msg;
        msg.set_trial_id(trial.id());
        msg.set_state(state);
        out->Write(msg);
      }
    };

    auto fut = m_orchestrator->watch_trials(std::move(handler));
    fut.wait();
  }
  catch(const std::exception& exc) {
    return MakeErrorStatus("WatchTrials failure on exception [%s]", exc.what());
  }
  catch(...) {
    return MakeErrorStatus("WatchTrials failure on unknown exception");
  }

  return grpc::Status::OK;
}

grpc::Status TrialLifecycleService::Version(grpc::ServerContext*, const cogmentAPI::VersionRequest*, cogmentAPI::VersionInfo* out) {
  SPDLOG_TRACE("Version()");

  try {
    m_orchestrator->Version(out);
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
