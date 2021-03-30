// Copyright 2020 Artificial Intelligence Redefined <dev+cogment@ai-r.com>
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

#include "cogment/services/trial_lifecycle_service.h"

#include "cogment/orchestrator.h"
#include "cogment/versions.h"

namespace cogment {

TrialLifecycleService::TrialLifecycleService(Orchestrator* orch) : orchestrator_(orch) {}

::easy_grpc::Future<::cogment::TrialStartReply> TrialLifecycleService::StartTrial(::cogment::TrialStartRequest req,
                                                                                  easy_grpc::Context) {
  auto params = orchestrator_->default_trial_params();

  // Apply config override if provided
  if (req.has_config()) {
    params.mutable_trial_config()->set_content(req.config().content());
  }

  return orchestrator_->start_trial(std::move(params), req.user_id()).then([](std::shared_ptr<Trial> trial) {
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
  auto trial = orchestrator_->get_trial(uuids::uuid::from_string(ctx.get_client_header("trial-id")));
  trial->terminate();
  return {};
}

// TODO: Implement this function
#define GET_TRIAL_INFO_IMPLEMENTED_FOR_NEW_API 0
#if GET_TRIAL_INFO_IMPLEMENTED_FOR_NEW_API
::cogment::TrialInfoReply TrialLifecycleService::GetTrialInfo(::cogment::TrialInfoRequest req, easy_grpc::Context ctx) {
  ::cogment::TrialInfoReply result;
  auto add_trial = [&](Trial* trial) {
    auto trial_info = result.add_trial();

    auto trial_lock = trial->lock();
    trial_info->set_trial_id(to_string(trial->id()));
    trial_info->set_state(get_trial_api_state(trial->state()));
  };

  auto trial_id_str = ctx.get_client_header("trial-id");

  if (trial_id_str == "all_trials") {
    auto trials = orchestrator_->all_trials();
    for (auto& trial : trials) {
      add_trial(trial.get());
    }
  }
  else {
    auto trial_id = uuids::uuid::from_string(trial_id_str);
    auto trial = orchestrator_->get_trial(trial_id);
    add_trial(trial.get());
  }

  return result;
}
#else
::cogment::TrialInfoReply TrialLifecycleService::GetTrialInfo(::cogment::TrialInfoRequest, easy_grpc::Context) {
  throw std::runtime_error("GetTrialInfo is not implemented");
  return {};
}
#endif

::easy_grpc::Stream_future<::cogment::TrialListEntry> TrialLifecycleService::WatchTrials(
    ::cogment::TrialListRequest req, easy_grpc::Context) {
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

  // Report current trials.
  for (const auto& trial : orchestrator_->all_trials()) {
    handler(*trial.get());
  }

  // TODO: Between the initial report just above and the handler registration just bellow some trials could change
  // state. Those states won't be reported

  // Request that the orchestrator invokes the handler whenever trials change.
  orchestrator_->watch_trials(std::move(handler));

  return future;
}

::cogment::VersionInfo TrialLifecycleService::Version(::cogment::VersionRequest, easy_grpc::Context) {
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
