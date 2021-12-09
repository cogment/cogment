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
#include "cogment/utils.h"
#include "cogment/versions.h"

#include "slt/settings.h"
#include "spdlog/spdlog.h"
#include "uuid.h"

namespace {
uuids::uuid_system_generator g_uuid_generator;
}  // namespace

namespace cogment {
Orchestrator::Orchestrator(cogmentAPI::TrialParams default_trial_params, uint32_t gc_frequency,
                           std::shared_ptr<grpc::ChannelCredentials> creds, prometheus::Registry* metrics_registry) :
    m_default_trial_params(std::move(default_trial_params)),
    m_gc_frequency(gc_frequency),
    m_channel_pool(creds),
    m_hook_stubs(&m_channel_pool),
    m_log_stubs(&m_channel_pool),
    m_env_stubs(&m_channel_pool),
    m_agent_stubs(&m_channel_pool),
    m_gc_countdown(gc_frequency) {
  SPDLOG_TRACE("Orchestrator()");

  // TODO: Transform this into a "garbage collection" thread so the gc could
  //       also be triggered by time, in order to prevent old ended
  //       or stale trials from lingering if no new trials are started.
  m_delete_thread_fut = thread_pool().push("Orchestrator trial deletion", [this]() {
    while (true) {
      try {  // overkill
        auto trial = m_trials_to_delete.pop();
        if (trial == nullptr) {
          return;
        }
        SPDLOG_TRACE("Trial [{}] - deleting...", trial->id());
        trial.reset();  // Not really necessary, but clarifies intent
      }
      catch (const std::exception& exc) {
        spdlog::error("Problem deleting a trial: {}", exc.what());
      }
      catch (...) {
        spdlog::error("Unknown problem deleting a trial");
      }
    }
  });

  if (metrics_registry != nullptr) {
    auto& trial_family = prometheus::BuildSummary()
                             .Name("orchestrator_trial_duration_seconds")
                             .Help("Duration (in seconds) of a trial")
                             .Register(*metrics_registry);
    m_trials_metrics = &(trial_family.Add({}, prometheus::Summary::Quantiles()));

    auto& tick_family = prometheus::BuildSummary()
                            .Name("orchestrator_tick_duration_seconds")
                            .Help("Duration (in seconds) of a normals step (not the first or last step)")
                            .Register(*metrics_registry);
    m_ticks_metrics = &(tick_family.Add({}, prometheus::Summary::Quantiles()));

    auto& gc_family = prometheus::BuildSummary()
                          .Name("orchestrator_garbage_collection_duration_seconds")
                          .Help("Duration (in seconds) of a trial garbage collection call")
                          .Register(*metrics_registry);
    m_gc_metrics = &(gc_family.Add({}, prometheus::Summary::Quantiles()));
  }
  else {
    m_trials_metrics = nullptr;
    m_ticks_metrics = nullptr;
    m_gc_metrics = nullptr;
  }
}

Orchestrator::~Orchestrator() {
  SPDLOG_TRACE("~Orchestrator()");

  {
    const std::lock_guard lg(m_notification_lock);
    for (auto& watcher : m_trial_watchers) {
      watcher.prom.set_value();
    }
  }

  m_trials_to_delete.push({});
  m_delete_thread_fut.wait();
}

std::shared_ptr<Trial> Orchestrator::start_trial(cogmentAPI::TrialParams params, const std::string& user_id,
                                                 std::string trial_id_req) {
  if (trial_id_req.empty()) {
    trial_id_req.assign(to_string(g_uuid_generator()));
  }
  else {
    // We pre-check the uniqueness to save some processing.
    const std::lock_guard lg(m_trials_mutex);
    if (m_trials.find(trial_id_req) != m_trials.end()) {
      return nullptr;
    }
  }

  try {
    m_perform_trial_gc();
  }
  catch (const std::exception& exc) {
    spdlog::error("Failure to perform garbage collection of trials [{}]", exc.what());
  }
  catch (...) {
    spdlog::error("Failure to perform garbage collection of trials");
  }

  auto new_trial = Trial::make(this, user_id, trial_id_req, Trial::Metrics {m_trials_metrics, m_ticks_metrics});

  // Register the trial
  {
    const std::lock_guard lg(m_trials_mutex);
    auto [itor, inserted] = m_trials.emplace(new_trial->id(), new_trial);
    if (!inserted) {
      return nullptr;
    }
  }

  auto final_param = m_perform_pre_hooks(std::move(params), new_trial->id(), user_id);

  new_trial->start(std::move(final_param));
  spdlog::info("Trial [{}] successfully initialized", new_trial->id());

  return new_trial;
}

void Orchestrator::add_prehook(const std::string& url) { m_prehooks.push_back(m_hook_stubs.get_stub_entry(url)); }

cogmentAPI::TrialParams Orchestrator::m_perform_pre_hooks(cogmentAPI::TrialParams&& params, const std::string& trial_id,
                                                          const std::string& user_id) {
  cogmentAPI::PreTrialParams hook_param;

  *hook_param.mutable_params() = std::move(params);
  for (auto& hook : m_prehooks) {
    grpc::ClientContext hook_context;
    hook_context.AddMetadata("trial-id", trial_id);
    hook_context.AddMetadata("user-id", user_id);

    auto status = hook->get_stub().OnPreTrial(&hook_context, hook_param, &hook_param);
    if (!status.ok()) {
      throw MakeException("Pre-trial hook failure [{}]", status.error_message());
    }
  }

  return std::move(*hook_param.mutable_params());
}

void Orchestrator::m_perform_trial_gc() {
  m_gc_countdown--;
  if (m_gc_countdown != 0) {
    return;
  }
  m_gc_countdown.store(m_gc_frequency);

  const auto start = Timestamp();

  spdlog::debug("Performing garbage collection of ended and stale trials");
  std::vector<std::shared_ptr<Trial>> stale_trials;

  // Extra exception catching in this function to find throwing code of non-std exception
  try {
    const std::lock_guard lg(m_trials_mutex);

    auto itor = m_trials.begin();
    while (itor != m_trials.end()) {
      auto& trial = itor->second;
      if (trial == nullptr) {
        throw MakeException("Null trial stored in list");
      }

      if (trial->state() == Trial::InternalState::ended) {
        m_trials_to_delete.push(std::move(trial));
        itor = m_trials.erase(itor);
      }
      else {
        try {
          if (trial->is_stale()) {
            stale_trials.emplace_back(trial);
            ++itor;
          }
          else {
            ++itor;
          }
        }
        catch (...) {
          spdlog::error("Problem performing garbage collection (1.1)");
          throw;
        }
      }
    }
  }
  catch (...) {
    spdlog::error("Problem performing garbage collection (1)");
    throw;
  }

  try {
    // Terminate may be long, so we don't want to lock the list during that time.
    // We don't go as far as with the deleted trials (i.e. terminating in a different thread)
    // because stale trials should be rare.
    for (auto& trial : stale_trials) {
      trial->terminate("The trial was inactive for too long");
    }
  }
  catch (...) {
    spdlog::error("Problem performing garbage collection (2)");
    throw;
  }

  if (m_gc_metrics != nullptr) {
    const auto duration = Timestamp() - start;
    m_gc_metrics->Observe(static_cast<double>(duration) * NANOS_INV);
  }
  SPDLOG_TRACE("Garbage collection done");
}

std::shared_ptr<Trial> Orchestrator::get_trial(const std::string& trial_id) const {
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

std::future<void> Orchestrator::watch_trials(HandlerFunction func) {
  SPDLOG_TRACE("Adding new notification function");

  // We have to lock here to make sure the new handler does not miss a state
  const std::lock_guard lg(m_notification_lock);

  // Report current trial states
  for (const auto& trial : all_trials()) {
    if (!func(*trial.get())) {
      std::promise<void> prom;
      prom.set_value();
      return prom.get_future();
    }
  }

  auto& new_watcher = m_trial_watchers.emplace_back(std::move(func));

  return new_watcher.prom.get_future();
}

void Orchestrator::notify_watchers(const Trial& trial) {
  const std::lock_guard lg(m_notification_lock);

  auto itor = m_trial_watchers.begin();
  while (itor != m_trial_watchers.end()) {
    if (itor->handler(trial)) {
      ++itor;
    }
    else {
      itor->prom.set_value();
      itor = m_trial_watchers.erase(itor);
    }
  }
}

void Orchestrator::Version(cogmentAPI::VersionInfo* out) {
  SPDLOG_TRACE("Orchestrator::Version()");

  auto ver = out->add_versions();
  ver->set_name("orchestrator");
  ver->set_version(COGMENT_ORCHESTRATOR_VERSION);

  ver = out->add_versions();
  ver->set_name("cogment-api");
  ver->set_version(COGMENT_API_VERSION);
}
}  // namespace cogment