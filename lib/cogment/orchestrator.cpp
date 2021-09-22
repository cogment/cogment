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

#include <exception>

namespace {
constexpr double NANOS_INV = 1.0 / 1'000'000'000;
}

namespace settings {

constexpr std::uint32_t default_garbage_collection_frequency = 10;

slt::Setting garbage_collection_frequency = slt::Setting_builder<std::uint32_t>()
                                                .with_default(default_garbage_collection_frequency)
                                                .with_description("Number of trials between garbage collection runs")
                                                .with_arg("gb_freq");
}  // namespace settings

namespace cogment {
Orchestrator::Orchestrator(Trial_spec trial_spec, cogmentAPI::TrialParams default_trial_params,
                           std::shared_ptr<grpc::ChannelCredentials> creds,
                           prometheus::Registry* metrics_registry) :
    m_trial_spec(std::move(trial_spec)),
    m_default_trial_params(std::move(default_trial_params)),
    m_channel_pool(creds),
    m_env_stubs(&m_channel_pool),
    m_agent_stubs(&m_channel_pool),
    m_log_stubs(&m_channel_pool),
    m_actor_service(this),
    m_trial_lifecycle_service(this),
    m_watchtrial_fut(m_watchtrial_prom.get_future()) {
  SPDLOG_TRACE("Orchestrator()");
  m_garbage_collection_countdown.store(settings::garbage_collection_frequency.get());

  m_trial_deletion_thread = std::thread([this]() {
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

  m_watchtrial_prom.set_value();
  m_trials_to_delete.push({});
  m_trial_deletion_thread.join();
}

std::shared_ptr<Trial> Orchestrator::start_trial(cogmentAPI::TrialParams params, const std::string& user_id) {
  spdlog::info("New trial requested by user [{}]", user_id);

  m_garbage_collection_countdown--;
  if (m_garbage_collection_countdown == 0) {
    const auto start = Timestamp();
    m_garbage_collection_countdown.store(settings::garbage_collection_frequency.get());
    m_perform_garbage_collection();

    if (m_gc_metrics != nullptr) {
      const auto duration = Timestamp() - start;
      m_gc_metrics->Observe(static_cast<double>(duration) * NANOS_INV);
    }
  }

  // TODO: The whole datalog management needs to be refactored
  std::unique_ptr<DatalogService> datalog;
  if (m_log_url.empty()) {
    datalog.reset(new DatalogServiceNull());
  }
  else {
    auto stub_entry = m_log_stubs.get_stub_entry(m_log_url);
    datalog.reset(new DatalogServiceImpl(stub_entry));
  }
  auto new_trial = std::make_shared<Trial>(this, std::move(datalog), user_id, Trial::Metrics {m_trials_metrics, m_ticks_metrics});

  // Register the trial immediately.
  {
    const std::lock_guard lg(m_trials_mutex);
    m_trials[new_trial->id()] = new_trial;
  }

  cogmentAPI::PreTrialContext init_param;
  *init_param.mutable_params() = std::move(params);
  init_param.set_user_id(user_id);

  auto final_param = m_perform_pre_hooks(std::move(init_param), new_trial->id());

  new_trial->start(std::move(*final_param.mutable_params()));
  spdlog::info("Trial [{}] successfully initialized", new_trial->id());

  return new_trial;
}

void Orchestrator::add_prehook(const HookEntryType& hook) { m_prehooks.push_back(hook); }

cogmentAPI::PreTrialContext Orchestrator::m_perform_pre_hooks(cogmentAPI::PreTrialContext&& data, const std::string& trial_id) {
  for (auto& hook : m_prehooks) {
    grpc::ClientContext hook_context;
    hook_context.AddMetadata("trial-id", trial_id);

    auto status = hook->get_stub().OnPreTrial(&hook_context, data, &data);
    if (!status.ok()) {
      throw MakeException("Trial [%s] - Prehook failure [%s]", trial_id.c_str(), status.error_message().c_str());
    }
  }

  return std::move(data);
}

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
        m_trials_to_delete.push(std::move(trial));
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

  // Terminate may be long, so we don't want to lock the list during that time.
  // We don't go as far as with the deleted trials because stale trials should be rare.
  for (auto trial : stale_trials) {
    spdlog::warn("Trial [{}] - Terminating because inactive for too long", trial->id());
    trial->finish();
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

std::shared_future<void> Orchestrator::watch_trials(HandlerFunction func) {
  SPDLOG_TRACE("Adding new notification function");

  const std::lock_guard lg(m_notification_lock);

  // Report current trial states
  for (const auto& trial : all_trials()) {
    func(*trial.get());
  }

  m_trial_watchers.emplace_back(std::move(func));

  return m_watchtrial_fut;
}

void Orchestrator::notify_watchers(const Trial& trial) {
  const std::lock_guard lg(m_notification_lock);

  for (auto& handler : m_trial_watchers) {
    handler(trial);
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