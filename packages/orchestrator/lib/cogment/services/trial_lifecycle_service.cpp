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

#ifndef NDEBUG
  #define SPDLOG_ACTIVE_LEVEL SPDLOG_LEVEL_TRACE
#endif

#include "cogment/services/trial_lifecycle_service.h"

#include "cogment/orchestrator.h"
#include "cogment/utils.h"

#include <bitset>

namespace cogment {

TrialLifecycleService::TrialLifecycleService(Orchestrator* orch) : m_orchestrator(orch) {}

grpc::Status TrialLifecycleService::StartTrial(grpc::ServerContext*, const cogmentAPI::TrialStartRequest* in,
                                               cogmentAPI::TrialStartReply* out) {
  SPDLOG_TRACE("TrialLifecycleService::StartTrial()");

  try {
    SPDLOG_TRACE("StartTrial from [{}] with trial id [{}]", in->user_id(), in->trial_id_requested());

    std::shared_ptr<Trial> trial;

    if (in->has_params()) {
      if (in->has_config()) {
        throw MakeException("Only config or params is allowed, not both");
      }
      cogmentAPI::TrialParams params(in->params());
      trial = m_orchestrator->start_trial(std::move(params), in->user_id(), in->trial_id_requested(), true);
    }
    else {
      auto params = m_orchestrator->default_trial_params();
      if (in->has_config()) {
        params.mutable_trial_config()->set_content(in->config().content());
      }
      trial = m_orchestrator->start_trial(std::move(params), in->user_id(), in->trial_id_requested(), false);
    }

    if (trial != nullptr) {
      out->set_trial_id(trial->id());
    }
    else {
      spdlog::warn("Start of trial [{}] by user [{}] was refused (was the trial id unique?)", in->trial_id_requested(),
                   in->user_id());
      out->Clear();
    }
  }
  catch (const std::exception& exc) {
    return MakeErrorStatus("TrialLifecycleSP/StartTrial failure: [{}]", exc.what());
  }
  catch (...) {
    return MakeErrorStatus("TrialLifecycleSP/StartTrial failure");
  }

  return grpc::Status::OK;
}

grpc::Status TrialLifecycleService::TerminateTrial(grpc::ServerContext* ctx,
                                                   const cogmentAPI::TerminateTrialRequest* req,
                                                   cogmentAPI::TerminateTrialReply* out) {
  SPDLOG_TRACE("TrialLifecycleService::TerminateTrial()");

  try {
    auto& metadata = ctx->client_metadata();
    auto trial_ids = FromMetadata(metadata, "trial-id");
    if (trial_ids.empty()) {
      throw MakeException("No 'trial-id' key in metadata");
    }
    SPDLOG_TRACE("TerminateTrial for [{}] trials", trial_ids.size());

    for (auto& trial_id : trial_ids) {
      auto trial = m_orchestrator->get_trial(std::string(trial_id));
      if (trial != nullptr) {
        if (req->hard_termination()) {
          trial->terminate("Externally requested");
        }
        else {
          trial->request_end();
        }
      }
    }

    out->Clear();
  }
  catch (const std::exception& exc) {
    return MakeErrorStatus("TrialLifecycleSP/TerminateTrial failure: [{}]", exc.what());
  }
  catch (...) {
    return MakeErrorStatus("TrialLifecycleSP/TerminateTrial failure");
  }

  return grpc::Status::OK;
}

grpc::Status TrialLifecycleService::GetTrialInfo(grpc::ServerContext* ctx, const cogmentAPI::TrialInfoRequest* in,
                                                 cogmentAPI::TrialInfoReply* out) {
  SPDLOG_TRACE("TrialLifecycleService::GetTrialInfo()");

  try {
    auto& metadata = ctx->client_metadata();
    auto trial_ids = FromMetadata(metadata, "trial-id");
    SPDLOG_TRACE("GetTrialInfo for [{}] trials (0 == all)", trial_ids.size());

    if (!trial_ids.empty()) {
      for (auto& trial_id : trial_ids) {
        auto trial = m_orchestrator->get_trial(std::string(trial_id));
        if (trial != nullptr) {
          SPDLOG_TRACE("GetTrialInfo for [{}]", trial_id);
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
  catch (const std::exception& exc) {
    return MakeErrorStatus("TrialLifecycleSP::GetTrialInfo failure: [{}]", exc.what());
  }
  catch (...) {
    return MakeErrorStatus("TrialLifecycleSP::GetTrialInfo failure");
  }

  return grpc::Status::OK;
}

grpc::Status TrialLifecycleService::WatchTrials(grpc::ServerContext*, const cogmentAPI::TrialListRequest* in,
                                                grpc::ServerWriter<cogmentAPI::TrialListEntry>* out) {
  SPDLOG_TRACE("TrialLifecycleService::WatchTrials()");

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

    const bool full_info = in->full_info();

    // This will get invoked on each state change of a trial
    auto handler = [state_mask, full_info, out](const Trial& trial) -> bool {
      auto state = get_trial_api_state(trial.state());

      if (state_mask.test(static_cast<std::size_t>(state))) {
        cogmentAPI::TrialListEntry msg;
        if (!full_info) {
          msg.set_trial_id(trial.id());
          msg.set_state(state);
        }
        else {
          auto info = msg.mutable_info();
          trial.set_info(info, false, false);
        }

        return out->Write(msg);
      }
      return true;
    };

    auto fut = m_orchestrator->watch_trials(std::move(handler));
    fut.wait();
  }
  catch (const std::exception& exc) {
    return MakeErrorStatus("TrialLifecycleSP::WatchTrials failure: [{}]", exc.what());
  }
  catch (...) {
    return MakeErrorStatus("TrialLifecycleSP::WatchTrials failure");
  }

  return grpc::Status::OK;
}

grpc::Status TrialLifecycleService::Version(grpc::ServerContext*, const cogmentAPI::VersionRequest*,
                                            cogmentAPI::VersionInfo* out) {
  SPDLOG_TRACE("TrialLifecycleService::Version()");

  try {
    m_orchestrator->Version(out);
  }
  catch (const std::exception& exc) {
    return MakeErrorStatus("TrialLifecycleSP::Version failure: [{}]", exc.what());
  }
  catch (...) {
    return MakeErrorStatus("TrialLifecycleSP::Version failure");
  }

  return grpc::Status::OK;
}

grpc::Status TrialLifecycleService::Status(grpc::ServerContext*, const cogmentAPI::StatusRequest* request,
                                           cogmentAPI::StatusReply* reply) {
  SPDLOG_TRACE("TrialLifecycleService::Status()");

  try {
    if (request->names().size() > 0) {
      // We purposefully don't scan for "*" ahead of time to allow explicit values before.
      bool all = false;
      for (auto name : request->names()) {
        if (name == "*") {
          all = true;
        }
        if (all || name == "overall_load") {
          (*reply->mutable_statuses())["overall_load"] = "0";
        }
        if (all) {
          break;
        }
      }
    }
  }
  catch (const std::exception& exc) {
    return MakeErrorStatus("TrialLifecycleService::Status failure: [{}]", exc.what());
  }
  catch (...) {
    return MakeErrorStatus("TrialLifecycleService::Status failure");
  }

  return grpc::Status::OK;
}

}  // namespace cogment
