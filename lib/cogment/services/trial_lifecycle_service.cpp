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

TrialLifecycleService::TrialLifecycleService(Orchestrator* orch) : m_orchestrator(orch) {}

::easy_grpc::Future<cogmentAPI::TrialStartReply> TrialLifecycleService::StartTrial(::cogmentAPI::TrialStartRequest req,
                                                                                  easy_grpc::Context) {
  SPDLOG_TRACE("StartTrial [{}]", req.user_id());

  auto params = m_orchestrator->default_trial_params();

  // Apply config override if provided
  if (req.has_config()) {
    params.mutable_trial_config()->set_content(req.config().content());
  }

  auto trial_fut = m_orchestrator->start_trial(std::move(params), req.user_id());

  return trial_fut.then([](std::shared_ptr<Trial> trial) {
    ::cogmentAPI::TrialStartReply reply;

    reply.set_trial_id(trial->id());

    for (const auto& actor : trial->actors()) {
      auto actor_in_trial = reply.add_actors_in_trial();
      actor_in_trial->set_actor_class(actor->actor_class()->name);
      actor_in_trial->set_name(actor->actor_name());
    }

    return reply;
  });
}

::cogmentAPI::TerminateTrialReply TrialLifecycleService::TerminateTrial(::cogmentAPI::TerminateTrialRequest,
                                                                     easy_grpc::Context ctx) {
  std::string trial_id(ctx.get_client_header("trial-id"));
  SPDLOG_TRACE("TerminateTrial [{}]", trial_id);

  auto trial = m_orchestrator->get_trial(trial_id);
  if (trial != nullptr) {
    trial->terminate();
  }
  else {
    spdlog::error("Trial [{}] doesn't exist for termination", trial_id);
  }

  return {};
}

::cogmentAPI::TrialInfoReply TrialLifecycleService::GetTrialInfo(::cogmentAPI::TrialInfoRequest req, easy_grpc::Context ctx) {
  std::string_view trial_id;
  try {
    // TODO: Find a way to test for header values instead of relying on exceptions
    trial_id = ctx.get_client_header("trial-id");
  }
  catch (const std::out_of_range&) {
    // There was no "trial-id" header
  }
  SPDLOG_TRACE("GetTrialInfo [{}]", trial_id);

  cogmentAPI::TrialInfoReply result;
  if (!trial_id.empty()) {
    auto trial = m_orchestrator->get_trial(std::string(trial_id));
    if (trial != nullptr) {
      auto trial_info = result.add_trial();
      trial->set_info(trial_info, req.get_latest_observation(), req.get_actor_list());
    }
  }
  else {
    // The user is asking for ALL trials
    auto trials = m_orchestrator->all_trials();
    for (auto& trial : trials) {
      auto trial_info = result.add_trial();
      trial->set_info(trial_info, req.get_latest_observation(), req.get_actor_list());
    }
  }

  return result;
}

::easy_grpc::Stream_future<cogmentAPI::TrialListEntry> TrialLifecycleService::WatchTrials(
    ::cogmentAPI::TrialListRequest req, easy_grpc::Context) {
  SPDLOG_TRACE("WatchTrials");

  // Shared and not a unique ptr because we want the lambda copy constructible
  auto promise = std::make_shared<Trial_promise>();
  Trial_future future(promise->get_future());

  // Build a bitmask for testing wether or not a trial should be reported.
  std::bitset<cogmentAPI::TrialState_MAX + 1> state_mask;
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
      cogmentAPI::TrialListEntry msg;
      msg.set_trial_id(trial.id());
      msg.set_state(state);
      promise->push(std::move(msg));
    }
  };

  m_orchestrator->watch_trials(std::move(handler));

  return future;
}

::cogmentAPI::VersionInfo TrialLifecycleService::Version(::cogmentAPI::VersionRequest, easy_grpc::Context) {
  ::cogmentAPI::VersionInfo result;

  auto ver = result.add_versions();
  ver->set_name("orchestrator");
  ver->set_version(COGMENT_ORCHESTRATOR_VERSION);

  ver = result.add_versions();
  ver->set_name("cogment-api");
  ver->set_version(COGMENT_API_VERSION);

  return result;
}
}  // namespace cogment
