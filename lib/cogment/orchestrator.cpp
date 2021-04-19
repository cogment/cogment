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

#include "cogment/orchestrator.h"
#include "cogment/client_actor.h"
#include "cogment/utils.h"

#include "slt/settings.h"
#include "spdlog/spdlog.h"

namespace settings {

constexpr std::uint32_t default_garbage_collection_frequency = 10;

slt::Setting garbage_collection_frequency = slt::Setting_builder<std::uint32_t>()
                                                .with_default(default_garbage_collection_frequency)
                                                .with_description("Number of trials between garbage collection runs")
                                                .with_arg("gb_freq");
}  // namespace settings

namespace cogment {
Orchestrator::Orchestrator(Trial_spec trial_spec, cogment::TrialParams default_trial_params,
                           std::shared_ptr<easy_grpc::client::Credentials> creds)
    : trial_spec_(std::move(trial_spec)),
      default_trial_params_(std::move(default_trial_params)),
      channel_pool_(creds),
      env_stubs_(&channel_pool_, &client_queue_),
      agent_stubs_(&channel_pool_, &client_queue_),
      actor_service_(this),
      trial_lifecycle_service_(this) {
  garbage_collection_countdown_.store(settings::garbage_collection_frequency.get());
}

Orchestrator::~Orchestrator() {}

Future<std::shared_ptr<Trial>> Orchestrator::start_trial(cogment::TrialParams params, std::string user_id) {
  garbage_collection_countdown_--;
  if (garbage_collection_countdown_ <= 0) {
    garbage_collection_countdown_.store(settings::garbage_collection_frequency.get());
    perform_garbage_collection_();
  }

  auto new_trial = std::make_shared<Trial>(this, user_id);

  // Register the trial immediately.
  {
    std::lock_guard l(trials_mutex_);
    trials_[new_trial->id()] = new_trial;
  }

  cogment::PreTrialContext init_ctx;
  *init_ctx.mutable_params() = std::move(params);
  init_ctx.set_user_id(user_id);

  auto trial_id = to_string(new_trial->id());
  auto final_ctx_fut = perform_pre_hooks_(std::move(init_ctx), trial_id);

  return final_ctx_fut.then([new_trial, trial_id](auto final_ctx) {
    new_trial->start(std::move(*final_ctx.mutable_params()));
    spdlog::info("Trial [{}] successfully initialized", trial_id);
    return new_trial;
  });
}

TrialJoinReply Orchestrator::client_joined(TrialJoinRequest req) {
  Client_actor* joined_as_actor = nullptr;

  TrialJoinReply result;

  if (req.trial_id() != "") {
    std::lock_guard l(trials_mutex_);

    auto trial_uuid = uuids::uuid::from_string(req.trial_id());
    auto trial_itor = trials_.find(trial_uuid);
    if (trial_itor != trials_.end()) {
      joined_as_actor = trial_itor->second->get_join_candidate(req);
    }
  }
  else {
    std::lock_guard l(trials_mutex_);
    // We need to find a valid trial
    for (auto& candidate : trials_) {
      if (candidate.second->state() == Trial::InternalState::pending) {
        joined_as_actor = candidate.second->get_join_candidate(req);
        if (joined_as_actor) {
          break;
        }
      }
    }
  }

  if (joined_as_actor == nullptr) {
    throw MakeException("Could not join trial");
  }
  auto config_data = joined_as_actor->join();
  if (config_data) {
    result.mutable_config()->set_content(config_data.value());
  }
  result.set_actor_name(joined_as_actor->actor_name());
  result.set_trial_id(to_string(joined_as_actor->trial()->id()));

  for (const auto& actor : joined_as_actor->trial()->actors()) {
    auto actor_in_trial = result.add_actors_in_trial();
    actor_in_trial->set_actor_class(actor->actor_class()->name);
    actor_in_trial->set_name(actor->actor_name());
  }

  return result;
}

::easy_grpc::Stream_future<::cogment::TrialActionReply> Orchestrator::bind_client(
    const uuids::uuid& trial_id, std::string& actor_name,
    ::easy_grpc::Stream_future<::cogment::TrialActionRequest> actions) {
  auto trial = get_trial(trial_id);

  auto actor = dynamic_cast<Client_actor*>(trial->actor(actor_name).get());

  if (actor == nullptr) {
    throw MakeException("attempting to bind a service-driven actor");
  }

  return actor->bind(std::move(actions));
}

void Orchestrator::add_prehook(cogment::TrialHooks::Stub_interface* hook) { prehooks_.push_back(hook); }

Future<cogment::PreTrialContext> Orchestrator::perform_pre_hooks_(cogment::PreTrialContext ctx,
                                                                  const std::string& trial_id) {
  grpc_metadata trial_header;
  trial_header.key = grpc_slice_from_static_string("trial-id");
  trial_header.value = grpc_slice_from_copied_string(trial_id.c_str());

  // We need this set of headers to live through the pre-hook RPCS, and there's not great place to anchor them.
  // Since this is not performance critical, we'll just ref-count them on the ultimate result.
  auto headers = std::make_shared<std::vector<grpc_metadata>>(std::vector<grpc_metadata>{trial_header});

  easy_grpc::client::Call_options options;
  options.headers = headers.get();

  aom::Promise<cogment::PreTrialContext> prom;
  auto result = prom.get_future();
  prom.set_value(std::move(ctx));

  // Run prehooks.
  for (auto& hook : prehooks_) {
    result =
        result.then([hook, options, headers](auto context) { return hook->OnPreTrial(std::move(context), options); });
  }

  return result.then([headers](auto v) { return v; });
}

void Orchestrator::set_log_exporter(std::unique_ptr<DatalogStorageInterface> le) { log_exporter_ = std::move(le); }

// TODO: Add a timer to do garbage collection after 60 seconds (or whatever) since the last call
//       in order to prevent old, ended or stale trials from lingering if no new trials are started.
void Orchestrator::perform_garbage_collection_() {
  spdlog::debug("Performing garbage collection of ended and stale trials");

  std::vector<Trial*> stale_trials;
  {
    std::lock_guard l(trials_mutex_);

    auto itor = trials_.begin();
    while (itor != trials_.end()) {
      auto& trial = itor->second;

      if (trial->state() == Trial::InternalState::ended) {
        itor = trials_.erase(itor);
      }
      else if (trial->is_stale()) {
        stale_trials.emplace_back(trial.get());
        ++itor;
      }
      else {
        ++itor;
      }
    }
  }

  // terminate may be long, so we don't want to lock the list during that time
  for (auto trial : stale_trials) {
    spdlog::warn("Terminating trial [{}] because inactive for too long", to_string(trial->id()));
    trial->terminate();
  }
}

std::shared_ptr<Trial> Orchestrator::get_trial(const uuids::uuid& trial_id) const {
  std::lock_guard l(trials_mutex_);
  return trials_.at(trial_id);
}

std::vector<std::shared_ptr<Trial>> Orchestrator::all_trials() const {
  std::lock_guard l(trials_mutex_);

  std::vector<std::shared_ptr<Trial>> result;
  result.reserve(trials_.size());

  for (const auto& t : trials_) {
    result.push_back(t.second);
  }

  return result;
}

void Orchestrator::notify_watchers(const Trial& trial) {
  for (auto& handler : trial_watchers_) {
    handler(trial);
  }
}

}  // namespace cogment