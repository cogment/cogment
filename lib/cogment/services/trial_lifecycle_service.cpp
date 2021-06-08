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
#include "cogment/versions.h"

namespace cogment {

TrialLifecycleService::TrialLifecycleService(Orchestrator* orch) : orchestrator_(orch) {}

::easy_grpc::Future<::cogment::TrialStartReply> TrialLifecycleService::StartTrial(::cogment::TrialStartRequest req,
                                                                                  easy_grpc::Context) {
  SPDLOG_TRACE("StartTrial [{}]", req.user_id());

  auto params = orchestrator_->default_trial_params();

  // Apply config override if provided
  if (req.has_config()) {
    params.mutable_trial_config()->set_content(req.config().content());
  }

  auto trial_fut = orchestrator_->start_trial(std::move(params), req.user_id());

  return trial_fut.then([](std::shared_ptr<Trial> trial) {
    ::cogment::TrialStartReply reply;

    reply.set_trial_id(to_string(trial->id()));

    for (const auto& actor : trial->actors()) {
      auto actor_in_trial = reply.add_actors_in_trial();
      actor_in_trial->set_actor_class(actor->actor_class()->name);
      actor_in_trial->set_name(actor->actor_name());
    }

    return reply;
  });
}

::cogment::TerminateTrialReply TrialLifecycleService::TerminateTrial(::cogment::TerminateTrialRequest,
                                                                     easy_grpc::Context ctx) {
  const auto trial_id_strv = ctx.get_client_header("trial-id");
  SPDLOG_TRACE("TerminateTrial [{}]", trial_id_strv);

  auto trial = orchestrator_->get_trial(uuids::uuid::from_string(trial_id_strv));
  if (trial != nullptr) {
    trial->terminate();
  }
  else {
    spdlog::error("Trial [{}] doesn't exist for termination", std::string(trial_id_strv));
  }

  return {};
}

::cogment::TrialInfoReply TrialLifecycleService::GetTrialInfo(::cogment::TrialInfoRequest req, easy_grpc::Context ctx) {
  std::string_view trial_id_strv;
  try {
    trial_id_strv = ctx.get_client_header("trial-id");
  } catch (const std::out_of_range&) {
    // There was no "trial-id" header
  }
  SPDLOG_TRACE("GetTrialInfo [{}]", trial_id_strv);

  cogment::TrialInfoReply result;
  if (!trial_id_strv.empty()) {
    const auto trial_id = uuids::uuid::from_string(trial_id_strv);
    auto trial = orchestrator_->get_trial(trial_id);
    if (trial != nullptr) {
      auto trial_info = result.add_trial();
      trial->set_info(trial_info, req.get_latest_observation(), req.get_actor_list());
    }
  }
  else {
    // The user is asking for ALL trials
    auto trials = orchestrator_->all_trials();
    for (auto& trial : trials) {
      auto trial_info = result.add_trial();
      trial->set_info(trial_info, req.get_latest_observation(), req.get_actor_list());
    }
  }

  return result;
}

::easy_grpc::Stream_future<::cogment::TrialListEntry> TrialLifecycleService::WatchTrials(
    ::cogment::TrialListRequest req, easy_grpc::Context) {
  SPDLOG_TRACE("WatchTrials");

  // Shared and not a unique ptr because we want the lambda copy constructible
  auto promise = std::make_shared<Trial_promise>();
  Trial_future future(promise->get_future());

  // Build a bitmask for testing wether or not a trial should be reported.
  std::bitset<cogment::TrialState_MAX + 1> state_mask;
  if (req.filter_size() == 0) {
    // If filter is empty, we report everything
    state_mask.set();
  }
  else {
    for (auto state : req.filter()) {
      state_mask.set(static_cast<std::size_t>(state));
    }
  }

  // This will get invoked on each reported trial
  auto handler = [state_mask, promise](const Trial& trial) {
    auto state = get_trial_api_state(trial.state());

    if (state_mask.test(static_cast<std::size_t>(state))) {
      cogment::TrialListEntry msg;
      msg.set_trial_id(to_string(trial.id()));
      msg.set_state(state);
      promise->push(std::move(msg));
    }
  };

  orchestrator_->watch_trials(std::move(handler));

  return future;
}

::cogment::VersionInfo TrialLifecycleService::Version(::cogment::VersionRequest, easy_grpc::Context) {
  ::cogment::VersionInfo result;

  auto ver = result.add_versions();
  ver->set_name("orchestrator");
  ver->set_version(COGMENT_ORCHESTRATOR_VERSION);

  ver = result.add_versions();
  ver->set_name("cogment-api");
  ver->set_version(COGMENT_API_VERSION);

  return result;
}
}  // namespace cogment
