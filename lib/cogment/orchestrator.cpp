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
    : m_trial_spec(std::move(trial_spec)),
      m_default_trial_params(std::move(default_trial_params)),
      m_channel_pool(creds),
      m_env_stubs(&m_channel_pool, &m_client_queue),
      m_agent_stubs(&m_channel_pool, &m_client_queue),
      m_actor_service(this),
      m_trial_lifecycle_service(this) {
  SPDLOG_TRACE("Orchestrator()");
  m_garbage_collection_countdown.store(settings::garbage_collection_frequency.get());
}

Orchestrator::~Orchestrator() { SPDLOG_TRACE("~Orchestrator()"); }

aom::Future<std::shared_ptr<Trial>> Orchestrator::start_trial(cogment::TrialParams params, std::string user_id) {
  m_garbage_collection_countdown--;
  if (m_garbage_collection_countdown <= 0) {
    m_garbage_collection_countdown.store(settings::garbage_collection_frequency.get());
    m_perform_garbage_collection();
  }

  auto new_trial = std::make_shared<Trial>(this, user_id);

  // Register the trial immediately.
  {
    const std::lock_guard lg(m_trials_mutex);
    m_trials[new_trial->id()] = new_trial;
  }

  cogment::PreTrialContext init_ctx;
  *init_ctx.mutable_params() = std::move(params);
  init_ctx.set_user_id(user_id);

  auto trial_id = to_string(new_trial->id());
  auto final_ctx_fut = m_perform_pre_hooks(std::move(init_ctx), trial_id);

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
    const std::lock_guard lg(m_trials_mutex);

    auto trial_uuid = uuids::uuid::from_string(req.trial_id());
    auto trial_itor = m_trials.find(trial_uuid);
    if (trial_itor != m_trials.end()) {
      joined_as_actor = trial_itor->second->get_join_candidate(req);
    }
  }
  else {
    const std::lock_guard lg(m_trials_mutex);
    // We need to find a valid trial
    for (auto& candidate_trial : m_trials) {
      joined_as_actor = candidate_trial.second->get_join_candidate(req);
      if (joined_as_actor) {
        break;
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
  if (trial == nullptr) {
    throw MakeException("Unknown trial to bind [%s]", to_string(trial_id).c_str());
  }

  auto actor = dynamic_cast<Client_actor*>(trial->actor(actor_name).get());

  if (actor == nullptr) {
    throw MakeException("Attempting to bind a service-driven actor");
  }

  return actor->bind(std::move(actions));
}

void Orchestrator::add_prehook(const HookEntryType& hook) { m_prehooks.push_back(hook); }

aom::Future<cogment::PreTrialContext> Orchestrator::m_perform_pre_hooks(cogment::PreTrialContext ctx,
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

  for (auto& hook : m_prehooks) {
    result = result.then([hook, options, headers](auto context) {
      spdlog::debug("Calling a pre-hook on trial parameters");
      return hook->get_stub().OnPreTrial(std::move(context), options);
    });
  }

  return result.then([headers](auto v) { return v; });
}

void Orchestrator::set_log_exporter(std::unique_ptr<DatalogStorageInterface> le) { m_log_exporter = std::move(le); }

// TODO: Add a timer to do garbage collection after 60 seconds (or whatever) since the last call
//       in order to prevent old, ended or stale trials from lingering if no new trials are started.
void Orchestrator::m_perform_garbage_collection() {
  spdlog::debug("Performing garbage collection of ended and stale trials");

  std::vector<Trial*> stale_trials;
  {
    const std::lock_guard lg(m_trials_mutex);

    auto itor = m_trials.begin();
    while (itor != m_trials.end()) {
      auto& trial = itor->second;

      if (trial->state() == Trial::InternalState::ended) {
        itor = m_trials.erase(itor);
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

  SPDLOG_TRACE("Garbage collection done");
}

std::shared_ptr<Trial> Orchestrator::get_trial(const uuids::uuid& trial_id) const {
  const std::lock_guard lg(m_trials_mutex);
  auto itor = m_trials.find(trial_id);
  if (itor != m_trials.end()) {
    return itor->second;
  }
  else {
    return {};
  }
}

std::vector<std::shared_ptr<Trial>> Orchestrator::all_trials() const {
  const std::lock_guard lg(m_trials_mutex);

  std::vector<std::shared_ptr<Trial>> result;
  result.reserve(m_trials.size());

  for (const auto& t : m_trials) {
    result.push_back(t.second);
  }

  return result;
}

void Orchestrator::watch_trials(HandlerFunction func) {
  SPDLOG_TRACE("Adding new notification function");

  const std::lock_guard lg(m_notification_lock);

  // Report current trial states
  for (const auto& trial : all_trials()) {
    func(*trial.get());
  }

  m_trial_watchers.emplace_back(std::move(func));
}

void Orchestrator::notify_watchers(const Trial& trial) {
  SPDLOG_TRACE("Trial [{}] changed state to [{}] at tick [{}]", to_string(trial.id()),
               get_trial_state_string(trial.state()), trial.tick_id());

  const std::lock_guard lg(m_notification_lock);

  for (auto& handler : m_trial_watchers) {
    handler(trial);
  }
}

}  // namespace cogment