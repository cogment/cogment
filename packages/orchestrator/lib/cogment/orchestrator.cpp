// Copyright 2022 AI Redefined Inc. <dev+cogment@ai-r.com>
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
#include "cogment/uuid.h"
#include "cogment/versions.h"

#include "slt/settings.h"
#include "spdlog/spdlog.h"

namespace cogment {
Orchestrator::Orchestrator(cogmentAPI::TrialParams default_trial_params, uint32_t gc_frequency,
                           std::shared_ptr<grpc::ChannelCredentials> creds, prometheus::Registry* metrics_registry) :
    m_default_trial_params(std::move(default_trial_params)),
    m_gc_frequency(gc_frequency),
    m_channel_pool(creds),
    m_directory_stubs(&m_channel_pool),
    m_hook_stubs(&m_channel_pool),
    m_log_stubs(&m_channel_pool),
    m_env_stubs(&m_channel_pool),
    m_agent_stubs(&m_channel_pool),
    m_gc_countdown(gc_frequency),
    m_thread_pool(),
    m_watchdog(&m_thread_pool) {
  SPDLOG_TRACE("Orchestrator()");

  // TODO: Transform this into a "garbage collection" thread so the gc could
  //       also be triggered by time, in order to prevent old ended
  //       or stale trials from lingering if no new trials are started.
  m_delete_thread_fut = m_thread_pool.push("Orchestrator trial deletion", [this]() {
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
        spdlog::error("Problem deleting a trial: [{}]", exc.what());
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

  for (auto& rec : m_directory_registrations) {
    m_directory.deregister_service(rec);
  }

  {
    const std::lock_guard lg(m_notification_lock);
    for (auto& watcher : m_trial_watchers) {
      watcher.prom.set_value();
    }
  }

  m_trials_to_delete.push({});
  m_delete_thread_fut.wait();
}

std::shared_ptr<Trial> Orchestrator::start_trial(cogmentAPI::TrialParams&& params, const std::string& user_id,
                                                 std::string trial_id_req, bool final_params) {
  if (trial_id_req.empty()) {
    trial_id_req.assign(generate_uuid_v4());
  }
  else {
    // We pre-check the uniqueness to save some processing in most conflict cases.
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
    auto inserted = m_trials.emplace(new_trial->id(), new_trial).second;
    if (!inserted) {
      return nullptr;
    }
  }

  if (final_params) {
    spdlog::debug("Trial [{}] starting with parameters from the start command", new_trial->id());
    new_trial->start(std::move(params));
  }
  else {
    spdlog::debug("Trial [{}] starting with default parameters and pre-trial hooks", new_trial->id());
    auto hook_params = m_perform_pre_hooks(std::move(params), new_trial->id(), user_id);
    new_trial->start(std::move(hook_params));
  }

  spdlog::info("Trial [{}] successfully initialized", new_trial->id());

  return new_trial;
}

void Orchestrator::add_directory(std::string_view url, std::string_view auth_token) {
  EndpointData data;
  try {
    data.parse(url);
  }
  catch (const CogmentError& exc) {
    throw MakeException("Directory endpoint error: [{}]", exc.what());
  }

  if (data.scheme != EndpointData::SchemeType::GRPC) {
    throw MakeException("Only gRPC endpoints are valid for the directory service (grpc://) [{}]", url);
  }

  auto stub = m_directory_stubs.get_stub_entry(std::string(data.address));
  m_directory.add_stub(stub);
  m_directory.set_auth_token(auth_token);
}

void Orchestrator::register_to_directory(std::string_view host, uint16_t actor_port, uint16_t lifecycle_port,
                                         const std::string& props) {
  static const std::string COGMENT_VERSION_STR = COGMENT_VERSION;

  if (!m_directory.is_set()) {
    spdlog::debug("No directory set to register host [{}]", host);
    return;
  }

  constexpr char PROPERTY_SEPARATOR = ',';
  constexpr char VALUE_SEPARATOR = '=';
  auto properties = parse_properties(props, PROPERTY_SEPARATOR, VALUE_SEPARATOR);
  properties.emplace_back("__version", COGMENT_VERSION_STR);

  std::string host_address;
  std::string_view self_host;
  if (!host.empty()) {
    self_host = host;
    properties.emplace_back("__registration_source", "Self-Command_Line");
  }
  else {
    host_address = GetHostAddress();
    self_host = host_address;
    properties.emplace_back("__registration_source", "Self-Implicit");
  }
  spdlog::debug("Registering Orchestrator to directory as [{}]", self_host);

  if (actor_port > 0) {
    auto res = m_directory.register_host(self_host, actor_port, m_channel_pool.is_ssl(),
                                         Directory::ServiceType::CLIENT_ACTOR, properties);
    m_directory_registrations.emplace_back(std::move(res));
  }

  if (lifecycle_port > 0) {
    auto res = m_directory.register_host(self_host, lifecycle_port, m_channel_pool.is_ssl(),
                                         Directory::ServiceType::LIFE_CYCLE, properties);
    m_directory_registrations.emplace_back(std::move(res));
  }
}

void Orchestrator::add_prehook(const std::string& user_url) {
  EndpointData data;
  try {
    data.parse(user_url);
  }
  catch (const CogmentError& exc) {
    throw MakeException("Pre-trial hook endpoint error: [{}]", exc.what());
  }

  if (data.is_context_endpoint()) {
    data.path = EndpointData::PathType::PREHOOK;
  }

  Directory::InquiredAddress address;
  try {
    address = m_directory.inquire_address("pre-hook", data);
  }
  catch (const CogmentError& exc) {
    throw MakeException("Pre-trial hook endpoint error: [{}]", exc.what());
  }

  if (address.address == CLIENT_ACTOR_ADDRESS) {
    throw MakeException("Pre-trial hook endpoint resolved to 'client'");
  }

  if (address.ssl && !use_ssl()) {
    // TODO: We should pre-emptively search for the proper hook in the directory (to match ssl use)
    throw MakeException("Pre-trial hook requires a SSL connection");
  }

  m_prehooks.push_back(m_hook_stubs.get_stub_entry(address.address));
}

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
  if (m_gc_countdown > 0) {
    return;
  }
  m_gc_countdown.store(m_gc_frequency);

  SPDLOG_TRACE("Garbage collection starting");
  const auto start = Timestamp();

  std::vector<std::shared_ptr<Trial>> stale_trials;

  const std::lock_guard lg(m_trials_mutex);

  size_t nb_deleted_trials = 0;
  auto itor = m_trials.begin();
  while (itor != m_trials.end()) {
    auto& trial = itor->second;
    if (trial == nullptr) {
      throw MakeException("Null trial stored in list");
    }

    if (trial->state() == Trial::InternalState::ended) {
      m_trials_to_delete.push(std::move(trial));
      itor = m_trials.erase(itor);
      nb_deleted_trials++;
    }
    else if (trial->is_stale()) {
      stale_trials.emplace_back(trial);
      ++itor;
    }
    else {
      ++itor;
    }
  }

  // Terminate may be long, so we don't want to lock the list during that time.
  // We don't go as far as with the deleted trials (i.e. terminating in a different thread)
  // because stale trials should be rare.
  for (auto& trial : stale_trials) {
    trial->terminate("The trial was inactive for too long");
  }

  if (m_gc_metrics != nullptr) {
    const auto duration = Timestamp() - start;
    m_gc_metrics->Observe(static_cast<double>(duration) * NANOS_INV);
  }

  spdlog::debug("Garbage collection of newly ended [{}] ([{}] in process pipe) and stale [{}] trials",
                nb_deleted_trials, m_trials_to_delete.size(), stale_trials.size());
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
  ver->set_name("cogment");
  ver->set_version(COGMENT_VERSION);
}
}  // namespace cogment
