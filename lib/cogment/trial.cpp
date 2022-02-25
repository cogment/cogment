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

#include "cogment/trial.h"
#include "cogment/orchestrator.h"
#include "cogment/environment.h"
#include "cogment/actor.h"
#include "cogment/agent_actor.h"
#include "cogment/client_actor.h"
#include "cogment/datalog.h"
#include "cogment/stub_pool.h"

#include "spdlog/spdlog.h"

#include <limits>
#include <chrono>

namespace cogment {
const std::string DEFAULT_ENVIRONMENT_NAME("env");

// Directory property names
constexpr std::string_view ACTOR_CLASS_PROPERTY_NAME("actor_class");
constexpr std::string_view IMPLEMENTATION_PROPERTY_NAME("implementation");

constexpr int64_t AUTO_TICK_ID = -1;     // The actual tick ID will be determined by the Orchestrator
constexpr int64_t NO_DATA_TICK_ID = -2;  // When we have received no data (different from default/empty data)
constexpr uint64_t MAX_TICK_ID = static_cast<uint64_t>(std::numeric_limits<int64_t>::max());

const char* get_trial_state_string(Trial::InternalState state) {
  switch (state) {
  case Trial::InternalState::unknown:
    return "unknown";
  case Trial::InternalState::initializing:
    return "initializing";
  case Trial::InternalState::pending:
    return "pending";
  case Trial::InternalState::running:
    return "running";
  case Trial::InternalState::terminating:
    return "terminating";
  case Trial::InternalState::ended:
    return "ended";
  }

  throw MakeException<std::out_of_range>("Unknown trial state for string [{}]", static_cast<int>(state));
}

cogmentAPI::TrialState get_trial_api_state(Trial::InternalState state) {
  switch (state) {
  case Trial::InternalState::unknown:
    return cogmentAPI::UNKNOWN;
  case Trial::InternalState::initializing:
    return cogmentAPI::INITIALIZING;
  case Trial::InternalState::pending:
    return cogmentAPI::PENDING;
  case Trial::InternalState::running:
    return cogmentAPI::RUNNING;
  case Trial::InternalState::terminating:
    return cogmentAPI::TERMINATING;
  case Trial::InternalState::ended:
    return cogmentAPI::ENDED;
  }

  throw MakeException<std::out_of_range>("Unknown trial state for api: [{}]", static_cast<int>(state));
}

Trial::Trial(Orchestrator* orch, const std::string& user_id, const std::string& id, const Metrics& met) :
    m_id(id),
    m_user_id(user_id),
    m_start_timestamp(Timestamp()),
    m_end_timestamp(0),
    m_orchestrator(orch),
    m_metrics(met),
    m_state(InternalState::unknown),
    m_env_last_obs(false),
    m_end_requested(false),
    m_tick_id(0),
    m_tick_start_timestamp(0),
    m_nb_actors_acted(0),
    m_max_steps(std::numeric_limits<uint64_t>::max()),
    m_max_inactivity(std::numeric_limits<uint64_t>::max()) {
  SPDLOG_TRACE("Trial [{}] - Constructor", m_id);

  set_state(InternalState::initializing);
  refresh_activity();
}

Trial::~Trial() {
  SPDLOG_TRACE("Trial [{}] - Destructor", m_id);

  if (m_state != InternalState::unknown && m_state != InternalState::ended) {
    spdlog::error("Trial [{}] - Destroying trial before it is ended [{}]", m_id, get_trial_state_string(m_state));
  }

  // Destroy components while this trial instance still exists
  m_env.reset();
  m_actors.clear();
  m_datalog.reset();
}

ThreadPool& Trial::thread_pool() { return m_orchestrator->thread_pool(); }

const std::string& Trial::env_name() const {
  if (m_env != nullptr) {
    return m_env->name();
  }
  else {
    throw MakeException("Environment is not set to provide name");
  }
}

void Trial::advance_tick() {
  SPDLOG_TRACE("Trial [{}] - Tick [{}] is done", m_id, m_tick_id);

  if (m_tick_id < MAX_TICK_ID) {
    m_tick_id++;
  }
  else {
    throw MakeException("Tick id has reached the limit [{}]", m_tick_id);
  }
}

void Trial::new_obs(cogmentAPI::ObservationSet&& obs) {
  if (obs.tick_id() == AUTO_TICK_ID) {
    // do nothing
  }
  else if (obs.tick_id() < 0) {
    throw MakeException("Invalid negative tick id from environment");
  }
  else {
    const uint64_t new_tick_id = static_cast<uint64_t>(obs.tick_id());

    if (new_tick_id < m_tick_id) {
      throw MakeException("Environment repeated a tick id: [{}]", new_tick_id);
    }

    if (new_tick_id > MAX_TICK_ID) {
      throw MakeException("Tick id from environment is too large");
    }

    if (new_tick_id > m_tick_id) {
      throw MakeException("Environment skipped tick id: [{}] vs [{}]", new_tick_id, m_tick_id);
    }
  }

  auto sample = get_last_sample();
  if (sample != nullptr) {
    *(sample->mutable_observations()) = std::move(obs);
  }
  else {
    spdlog::debug("Trial [{}] - State [{}]. New observation lost", m_id, get_trial_state_string(m_state));
    return;
  }
}

void Trial::new_special_event(std::string_view desc) {
  auto sample = get_last_sample();
  if (sample != nullptr) {
    sample->mutable_info()->add_special_events(desc.data(), desc.size());
  }
  else {
    spdlog::debug("Trial [{}] - State [{}]. Special event lost [{}]", m_id, get_trial_state_string(m_state), desc);
  }
}

cogmentAPI::DatalogSample* Trial::get_last_sample() {
  const std::lock_guard lg(m_sample_lock);
  if (!m_step_data.empty()) {
    return &(m_step_data.back());
  }
  else {
    return nullptr;
  }
}

cogmentAPI::DatalogSample& Trial::make_new_sample() {
  const std::lock_guard lg(m_sample_lock);

  const uint64_t tick_start = Timestamp();

  if (!m_step_data.empty()) {
    m_step_data.back().mutable_info()->set_state(get_trial_api_state(m_state));
  }

  m_step_data.emplace_back();
  auto& sample = m_step_data.back();

  auto sample_actions = sample.mutable_actions();
  sample_actions->Reserve(m_actors.size());
  for (size_t index = 0; index < m_actors.size(); index++) {
    auto null_action = sample_actions->Add();
    null_action->set_tick_id(NO_DATA_TICK_ID);
  }
  m_nb_actors_acted = 0;

  auto info = sample.mutable_info();
  info->set_tick_id(m_tick_id);
  info->set_timestamp(tick_start);  // Changed later if first sample

  return sample;
}

void Trial::flush_samples() {
  SPDLOG_TRACE("Trial [{}] - Flushing last samples", m_id);
  const std::lock_guard lg(m_sample_lock);

  if (!m_step_data.empty()) {
    m_step_data.back().mutable_info()->set_state(get_trial_api_state(m_state));
  }

  if (m_datalog != nullptr) {
    for (auto& sample : m_step_data) {
      m_datalog->add_sample(std::move(sample));
    }
  }

  m_step_data.clear();
}

void Trial::prepare_actors() {
  if (m_params.actors().empty()) {
    throw MakeException("No Actor defined in parameters");
  }
  if (m_env == nullptr) {
    throw MakeException("Environment not ready for actors");
  }

  for (const auto& actor_info : m_params.actors()) {
    auto& endpoint = actor_info.endpoint();
    auto& name = actor_info.name();
    auto& actor_class = actor_info.actor_class();
    auto& implementation = actor_info.implementation();

    if (endpoint.empty() || name.empty() || actor_class.empty()) {
      throw MakeException("Actor [{}] not fully defined in parameters", name);
    }
    if (name == m_env->name()) {
      throw MakeException("Actor name cannot be the same as environment name [{}]", m_env->name());
    }

    static constexpr std::string_view DEPRECATED_CLIENT_ENDPOINT = "client";
    if (endpoint == DEPRECATED_CLIENT_ENDPOINT) {
      spdlog::warn("Client actor endpoint must be 'cogment://client' in the parameters [{}]", endpoint);

      auto client_actor = std::make_unique<ClientActor>(this, actor_info);
      m_actors.emplace_back(std::move(client_actor));
    }
    else {
      EndpointData data;
      try {
        parse_endpoint(endpoint, &data);
      }
      catch (const CogmentError& exc) {
        throw MakeException("Actor [{}] endpoint error: [{}]", name, exc.what());
      }
      auto& directory = m_orchestrator->directory();

      if (directory.is_context_endpoint(data)) {
        if (actor_class.find_first_of(INVALID_CHARACTERS) != name.npos) {
          throw MakeException("Actor class name contains invalid characters [{}]", actor_class);
        }
        if (implementation.find_first_of(INVALID_CHARACTERS) != implementation.npos) {
          throw MakeException("Actor implementation name contains invalid characters [{}]", implementation);
        }

        data.path = EndpointData::PathType::ACTOR;
        data.query.emplace_back(ACTOR_CLASS_PROPERTY_NAME, actor_class);
        if (!implementation.empty()) {
          data.query.emplace_back(IMPLEMENTATION_PROPERTY_NAME, implementation);
        }
      }

      std::string address;
      try {
        address = directory.get_address(data);
      }
      catch (const CogmentError& exc) {
        throw MakeException("Actor [{}] endpoint error: [{}]", name, exc.what());
      }

      if (address == CLIENT_ACTOR_ADDRESS) {
        auto client_actor = std::make_unique<ClientActor>(this, actor_info);
        m_actors.emplace_back(std::move(client_actor));
      }
      else {
        auto stub_entry = m_orchestrator->agent_pool()->get_stub_entry(address);
        auto agent_actor = std::make_unique<ServiceActor>(this, actor_info, stub_entry);
        m_actors.emplace_back(std::move(agent_actor));
      }
    }

    auto [itor, inserted] = m_actor_indexes.emplace(name, m_actors.size() - 1);
    if (!inserted) {
      throw MakeException("Actor name is not unique [{}]", name);
    }
  }
}

void Trial::prepare_environment() {
  auto& env_params = m_params.environment();
  auto& endpoint = env_params.endpoint();
  auto& implementation = env_params.implementation();
  auto& directory = m_orchestrator->directory();

  if (endpoint.empty()) {
    throw MakeException("No environment endpoint provided in parameters");
  }

  EndpointData data;
  try {
    parse_endpoint(endpoint, &data);
  }
  catch (const CogmentError& exc) {
    throw MakeException("Environment endpoint error: [{}]", exc.what());
  }

  if (directory.is_context_endpoint(data)) {
    if (implementation.find_first_of(INVALID_CHARACTERS) != implementation.npos) {
      throw MakeException("Environment implementation name contains invalid characters [{}]", implementation);
    }

    data.path = EndpointData::PathType::ENVIRONMENT;
    if (!implementation.empty()) {
      data.query.emplace_back(IMPLEMENTATION_PROPERTY_NAME, implementation);
    }
  }

  std::string address;
  try {
    address = directory.get_address(data);
  }
  catch (const CogmentError& exc) {
    throw MakeException("Environment endpoint error: [{}]", exc.what());
  }

  if (address == CLIENT_ACTOR_ADDRESS) {
    throw MakeException("Environment endpoint resolved to 'client'");
  }

  auto stub_entry = m_orchestrator->env_pool()->get_stub_entry(address);
  m_env = std::make_unique<Environment>(this, env_params, stub_entry);
}

void Trial::prepare_datalog() {
  if (!m_params.has_datalog()) {
    m_datalog = std::make_unique<DatalogServiceNull>();
  }
  else {
    auto& endpoint = m_params.datalog().endpoint();
    auto& directory = m_orchestrator->directory();

    if (endpoint.empty()) {
      throw MakeException("Parameter Datalog endpoint missing");
    }

    EndpointData data;
    try {
      parse_endpoint(endpoint, &data);
    }
    catch (const CogmentError& exc) {
      throw MakeException("Datalog endpoint error: [{}]", exc.what());
    }

    if (directory.is_context_endpoint(data)) {
      data.path = EndpointData::PathType::DATALOG;
    }

    std::string address;
    try {
      address = directory.get_address(data);
    }
    catch (const CogmentError& exc) {
      throw MakeException("Datalog endpoint error: [{}]", exc.what());
    }

    if (address == CLIENT_ACTOR_ADDRESS) {
      throw MakeException("Datalog endpoint resolved to 'client'");
    }

    auto stub_entry = m_orchestrator->log_pool()->get_stub_entry(address);
    m_datalog = std::make_unique<DatalogServiceImpl>(stub_entry);
  }

  m_datalog->start(this);
}

void Trial::start(cogmentAPI::TrialParams&& params) {
  SPDLOG_TRACE("Trial [{}] - Starting", m_id);

  if (m_state != InternalState::initializing) {
    throw MakeException("Trial is not in proper state to start: [{}]", get_trial_state_string(m_state));
  }

  m_params = std::move(params);
  SPDLOG_DEBUG("Trial [{}] - Configuring with parameters:\n {}", m_id, m_params.DebugString());

  if (m_params.environment().name().empty()) {
    spdlog::info("Trial [{}] - Environment name set to default [{}]", m_id, DEFAULT_ENVIRONMENT_NAME);
    m_params.mutable_environment()->set_name(DEFAULT_ENVIRONMENT_NAME);
  }

  if (m_params.max_steps() > 0) {
    m_max_steps = m_params.max_steps();
  }
  if (m_params.max_inactivity() > 0) {
    m_max_inactivity = m_params.max_inactivity() * NANOS;
  }

  prepare_datalog();
  prepare_environment();
  prepare_actors();

  make_new_sample();  // First sample

  set_state(InternalState::pending);

  auto self = shared_from_this();
  m_orchestrator->thread_pool().push("Trial starting", [self]() {
    try {
      std::vector<std::future<void>> actors_ready;
      for (const auto& actor : self->m_actors) {
        actors_ready.push_back(actor->init());
      }

      // TODO: Limit the time we wait for actor and environment init. Should have a limit per actor.
      for (size_t index = 0; index < actors_ready.size(); index++) {
        SPDLOG_TRACE("Trial [{}] - Waiting on actor [{}]...", self->m_id, self->m_actors[index]->actor_name());
        actors_ready[index].wait();
      }
      SPDLOG_TRACE("Trial [{}] - All actors started", self->m_id);

      // TODO: We could start the environment first (before the actors), then wait here.  But then we would
      //       have to synchronize everything, or hold the first observations until all actors are init.
      self->m_env->init().wait();

      spdlog::debug("Trial [{}] - Started", self->m_id);
    }
    catch (const std::exception& exc) {
      spdlog::error("Trial [{}] - Failed to start [{}]", self->m_id, exc.what());
    }
    catch (...) {
      spdlog::error("Trial [{}] - Failed to start for unknown reason", self->m_id);
    }
  });

  spdlog::debug("Trial [{}] - Configured", m_id);
}

void Trial::reward_received(const std::string& sender, cogmentAPI::Reward&& reward) {
  if (m_state < InternalState::pending) {
    spdlog::warn("Too early for trial [{}] to receive rewards.", m_id);
    return;
  }

  cogmentAPI::Reward* new_rew;
  auto sample = get_last_sample();
  if (sample != nullptr) {
    const std::lock_guard lg(m_reward_lock);
    new_rew = sample->add_rewards();
    *new_rew = reward;  // TODO: The reward may have a wildcard (or invalid) receiver, do we want that in the sample?
  }
  else {
    spdlog::debug("Trial [{}] - State [{}]. Reward from [{}] lost", m_id, get_trial_state_string(m_state), sender);
    return;
  }

  // TODO: Decide what to do with timed rewards (send anything present and past, hold future?)
  if (new_rew->tick_id() != AUTO_TICK_ID && new_rew->tick_id() != static_cast<int64_t>(m_tick_id)) {
    spdlog::error("Invalid reward tick from [{}]: [{}] (current tick id: [{}])", sender, new_rew->tick_id(), m_tick_id);
    return;
  }

  // Rewards are not dispatched as we receive them. They are accumulated, and sent once
  // per update.
  bool valid_name = for_actors(new_rew->receiver_name(), [this, new_rew, &sender](auto actor) {
    // Normally we should have only one source when receiving
    for (auto& src : *new_rew->mutable_sources()) {
      src.set_sender_name(sender);
      actor->add_reward_src(src, m_tick_id);
    }
  });

  if (!valid_name) {
    spdlog::error("Trial [{}] - Unknown receiver as reward destination [{}] from [{}]", m_id, new_rew->receiver_name(),
                  sender);
  }
}

void Trial::message_received(const std::string& sender, cogmentAPI::Message&& message) {
  if (m_state < InternalState::pending) {
    spdlog::warn("Too early for trial [{}] to receive messages.", m_id);
    return;
  }

  cogmentAPI::Message* new_msg;
  auto sample = get_last_sample();
  if (sample != nullptr) {
    const std::lock_guard lg(m_sample_message_lock);
    new_msg = sample->add_messages();
    *new_msg = std::move(message);
    new_msg->set_sender_name(sender);
  }
  else {
    spdlog::debug("Trial [{}] - State [{}]. Message from [{}] lost", m_id, get_trial_state_string(m_state), sender);
    return;
  }

  // TODO: Decide what to do with timed messages (send anything present and past, hold future?)
  if (new_msg->tick_id() != AUTO_TICK_ID && new_msg->tick_id() != static_cast<int64_t>(m_tick_id)) {
    spdlog::error("Invalid message tick from [{}]: [{}] (current tick id: [{}])", sender, new_msg->tick_id(),
                  m_tick_id);
    return;
  }

  if (new_msg->receiver_name() == m_env->name()) {
    m_env->send_message(*new_msg, m_tick_id);
  }
  else {
    bool valid_name = for_actors(new_msg->receiver_name(), [this, new_msg](auto actor) {
      actor->send_message(*new_msg, m_tick_id);
    });

    if (!valid_name) {
      spdlog::error("Trial [{}] - Unknown receiver as message destination [{}] from [{}]", m_id,
                    new_msg->receiver_name(), sender);
    }
  }
}

bool Trial::for_actors(const std::string& pattern, const std::function<void(Actor*)>& func) {
  // If exact name
  const auto actor_index_itor = m_actor_indexes.find(pattern);
  if (actor_index_itor != m_actor_indexes.end()) {
    const uint32_t index = actor_index_itor->second;
    func(m_actors[index].get());
    return true;
  }

  // If all actors
  if (pattern == "*") {
    for (const auto& actor : m_actors) {
      func(actor.get());
    }
    return true;
  }

  // If "class_name.actor_name" where actor_name can be "*"
  auto pos = pattern.find('.');
  if (pos != pattern.npos) {
    const auto class_name = pattern.substr(0, pos);
    const auto actor_name = pattern.substr(pos + 1);

    // If "class_name.*"
    if (actor_name == "*") {
      bool at_least_one = false;
      for (const auto& actor : m_actors) {
        if (actor->actor_class() == class_name) {
          at_least_one = true;
          func(actor.get());
        }
      }

      return at_least_one;
    }

    const auto actor_index_itor = m_actor_indexes.find(actor_name);
    if (actor_index_itor != m_actor_indexes.end()) {
      const uint32_t index = actor_index_itor->second;
      if (m_actors[index]->actor_class() == class_name) {
        func(m_actors[index].get());
        return true;
      }
    }
  }

  return false;
}

void Trial::dispatch_observations(bool last) {
  if (m_state == InternalState::ended) {
    return;
  }

  auto sample = get_last_sample();
  if (sample == nullptr) {
    spdlog::debug("Trial [{}] - State [{}]. Observations not sent", m_id, get_trial_state_string(m_state));
    return;
  }

  const auto& observations = sample->observations();

  std::uint32_t actor_index = 0;
  for (const auto& actor : m_actors) {
    auto obs_index = observations.actors_map(actor_index);
    cogmentAPI::Observation obs;
    obs.set_tick_id(m_tick_id);
    obs.set_timestamp(observations.timestamp());
    *obs.mutable_content() = observations.observations(obs_index);
    actor->dispatch_tick(std::move(obs), last);

    ++actor_index;
  }
}

void Trial::cycle_buffer() {
  const std::lock_guard lg(m_sample_lock);

  static constexpr uint64_t MIN_NB_BUFFERED_SAMPLES = 2;  // Because of the way we use the buffer
  static constexpr uint64_t NB_BUFFERED_SAMPLES = 5;      // Could be an external setting
  static constexpr uint64_t LOG_BATCH_SIZE = 1;           // Could be an external setting
  static constexpr uint64_t LOG_TRIGGER_SIZE = NB_BUFFERED_SAMPLES + LOG_BATCH_SIZE - 1;
  static_assert(NB_BUFFERED_SAMPLES >= MIN_NB_BUFFERED_SAMPLES);
  static_assert(LOG_BATCH_SIZE > 0);

  // Send overflow to log
  if (m_step_data.size() >= LOG_TRIGGER_SIZE) {
    while (m_step_data.size() >= NB_BUFFERED_SAMPLES) {
      m_datalog->add_sample(std::move(m_step_data.front()));
      m_step_data.pop_front();
    }
  }
}

cogmentAPI::ActionSet Trial::make_action_set() {
  cogmentAPI::ActionSet action_set;
  action_set.set_timestamp(Timestamp());

  action_set.set_tick_id(m_tick_id);

  const std::lock_guard lg(m_sample_lock);
  auto& sample = m_step_data.back();
  for (auto& act : sample.actions()) {
    if (act.tick_id() == AUTO_TICK_ID || act.tick_id() == static_cast<int64_t>(m_tick_id)) {
      action_set.add_actions(act.content());
    }
    else {
      // The registered action is not for this tick

      // TODO: Synchronize with `actor_acted` about past/future actions
      action_set.add_actions();  // Add default action
    }
  }

  return action_set;
}

bool Trial::finalize_env() {
  static constexpr auto timeout = std::chrono::seconds(60);

  SPDLOG_DEBUG("Trial [{}] - Waiting (max 60 sec) for environment [{}] to acknowledge 'LAST'", m_id, m_env->name());
  try {
    auto fut = m_env->last_ack();
    if (!fut.valid()) {
      throw std::future_error(std::future_errc::no_state);
    }

    auto status = fut.wait_for(timeout);
    switch (status) {
    case std::future_status::deferred:
      throw MakeException("Deferred last data");
    case std::future_status::ready:
      break;
    case std::future_status::timeout:
      throw MakeException("Last data wait timed out");
    }

    return true;
  }
  catch (const std::exception& exc) {
    spdlog::error("Trial [{}] - Environment [{}] last ack failed [{}]", m_id, m_env->name(), exc.what());
  }
  catch (...) {
    spdlog::error("Trial [{}] - Environment [{}] last ack failed", m_id, m_env->name());
  }

  return false;
}

void Trial::finalize_actors() {
  static constexpr auto timeout = std::chrono::seconds(30);

  SPDLOG_DEBUG("Trial [{}] - Waiting (max 30 sec per actor) for all actors to acknowledge 'LAST'", m_id);

  std::vector<std::future<void>> actors_last_ack;
  actors_last_ack.reserve(m_actors.size());
  for (auto& actor : m_actors) {
    actors_last_ack.emplace_back(actor->last_ack());
  }

  for (size_t index = 0; index < actors_last_ack.size(); index++) {
    auto& fut = actors_last_ack[index];

    try {
      if (!fut.valid()) {
        throw std::future_error(std::future_errc::no_state);
      }

      auto status = fut.wait_for(timeout);
      switch (status) {
      case std::future_status::deferred:
        throw MakeException("Deferred last data");
      case std::future_status::ready:
        break;
      case std::future_status::timeout:
        throw MakeException("Last data wait timed out");
      }
    }
    catch (const std::exception& exc) {
      spdlog::error("Trial [{}] - Actor [{}] last ack failed [{}]", m_id, m_actors[index]->actor_name(), exc.what());
    }
    catch (...) {
      spdlog::error("Trial [{}] - Actor [{}] last ack failed", m_id, m_actors[index]->actor_name());
    }
  }
}

// This should be called within a lock of m_terminating_lock
void Trial::finish() {
  SPDLOG_TRACE("Trial [{}] - finish()", m_id);

  if (m_state == InternalState::ended) {
    return;
  }
  set_state(InternalState::terminating);

  auto self = shared_from_this();
  m_orchestrator->thread_pool().push("Trial finishing", [self]() {
    SPDLOG_DEBUG("Trial [{}] - Finishing thread started", self->m_id);

    const bool env_finalize_success = self->finalize_env();
    if (env_finalize_success) {
      self->finalize_actors();
    }

    try {
      std::string details;
      if (!env_finalize_success) {
        spdlog::warn("Trial [{}] - Force ending all actors because environment failed to end properly", self->m_id);
        details = "Environment failed to end properly";
      }

      SPDLOG_DEBUG("Trial [{}] - Notifying all that the trial has ended", self->m_id);
      self->m_env->trial_ended(details);
      for (const auto& actor : self->m_actors) {
        actor->trial_ended(details);
      }
    }
    catch (const std::exception& exc) {
      spdlog::error("Trial [{}] - Component end failed [{}]", self->m_id, exc.what());
    }
    catch (...) {
      spdlog::error("Trial [{}] - component end failed", self->m_id);
    }

    self->set_state(InternalState::ended);
  });
}

void Trial::request_end() {
  SPDLOG_TRACE("Trial [{}] - End requested", m_id);
  new_special_event("End requested");
  m_end_requested = true;
}

void Trial::terminate(const std::string& details) {
  SPDLOG_TRACE("Trial [{}] - terminate()", m_id);
  const std::lock_guard lg(m_terminating_lock);

  if (m_state == InternalState::ended) {
    return;
  }
  try {
    set_state(InternalState::terminating);
  }
  catch (const CogmentError& exc) {
    spdlog::debug("Trial [{}] - Hard termination [{}] in state [{}]: {}", m_id, details,
                  get_trial_state_string(m_state), exc.what());
  }

  new_special_event("Forced termination: " + details);
  spdlog::info("Trial [{}] - Hard termination requested [{}]", m_id, details);

  if (m_env != nullptr) {
    try {
      m_env->trial_ended(details);
    }
    catch (const std::exception& exc) {
      spdlog::error("Trial [{}] - Environment [{}] termination failed [{}]", m_id, m_env->name(), exc.what());
    }
    catch (...) {
      spdlog::error("Trial [{}] - Environment [{}] termination failed", m_id, m_env->name());
    }
  }

  for (const auto& actor : m_actors) {
    try {
      actor->trial_ended(details);
    }
    catch (const std::exception& exc) {
      spdlog::error("Trial [{}] - Actor [{}] termination failed [{}]", m_id, actor->actor_name(), exc.what());
    }
    catch (...) {
      spdlog::error("Trial [{}] - Actor [{}] termination failed", m_id, actor->actor_name());
    }
  }

  set_state(InternalState::ended);
}

void Trial::env_observed(const std::string& env_name, cogmentAPI::ObservationSet&& obs, bool last) {
  SPDLOG_TRACE("Trial [{}] - New observations received from environment", m_id);

  const std::shared_lock lg(m_terminating_lock);
  refresh_activity();

  if (m_state < InternalState::pending) {
    spdlog::warn("Trial [{}] - Environemnt [{}] too early to receive observations.", m_id, env_name);
    return;
  }
  if (m_state == InternalState::ended) {
    spdlog::info("Trial [{}] - Environment [{}] observations arrived after end of trial. Data will be dropped [{}].",
                 m_id, env_name, last);
    return;
  }

  if (m_state >= InternalState::running) {
    advance_tick();
    make_new_sample();
  }
  else {
    // First observation
    set_state(InternalState::running);

    auto sample = get_last_sample();  // Actually first sample
    if (sample != nullptr) {
      auto info = sample->mutable_info();
      info->set_timestamp(Timestamp());
    }
  }
  new_obs(std::move(obs));

  if (!last) {
    dispatch_observations(false);
    cycle_buffer();
  }
  else {
    spdlog::info("Trial [{}] - Environment has ended the trial", m_id);
    new_special_event("Evironment ended trial");
    dispatch_observations(true);
    finish();
  }
}

void Trial::actor_acted(const std::string& actor_name, cogmentAPI::Action&& action) {
  const std::shared_lock lg(m_terminating_lock);
  refresh_activity();

  if (m_state < InternalState::pending) {
    spdlog::warn("Trial [{}] - Actor [{}] too early in trial to receive action.", m_id, actor_name);
    return;
  }
  if (m_state == InternalState::ended) {
    spdlog::info("Trial [{}] - Actor [{}] action arrived after end of trial. Data will be dropped.", m_id, actor_name);
    return;
  }

  const auto itor = m_actor_indexes.find(actor_name);
  if (itor == m_actor_indexes.end()) {
    spdlog::error("Trial [{}] - Unknown actor [{}] for action received.", m_id, actor_name);
    return;
  }
  const auto actor_index = itor->second;

  auto sample = get_last_sample();
  if (sample == nullptr) {
    spdlog::debug("Trial [{}] - State [{}]. Action from [{}] lost", m_id, get_trial_state_string(m_state), actor_name);
    return;
  }
  auto sample_action = sample->mutable_actions(actor_index);
  if (sample_action->tick_id() != NO_DATA_TICK_ID) {
    spdlog::warn("Trial [{}] - Actor [{}] multiple actions received for same step. Only the first one will be used.",
                 m_id, actor_name);
    return;
  }

  // TODO: Determine what we want to do in case of actions in the past or future
  if (action.tick_id() != AUTO_TICK_ID && action.tick_id() != static_cast<int64_t>(m_tick_id)) {
    spdlog::warn("Trial [{}] - Actor [{}] invalid action step: [{}] vs [{}]. Default action will be used.", m_id,
                 actor_name, action.tick_id(), m_tick_id);
  }

  SPDLOG_TRACE("Trial [{}] - Actor [{}] received action for tick [{}].", m_id, actor_name, m_tick_id);
  *sample_action = std::move(action);

  const auto new_count = ++m_nb_actors_acted;
  if (new_count == m_actors.size()) {
    SPDLOG_TRACE("Trial [{}] - All actions received for tick [{}]", m_id, m_tick_id);

    const bool last_actions = (m_tick_id >= m_max_steps || m_end_requested);

    if (!last_actions) {
      m_env->dispatch_actions(make_action_set(), false);

      // Here because we want this metric to be outside the first and last tick (i.e. overhead)
      if (m_metrics.tick_duration != nullptr) {
        if (m_tick_start_timestamp > 0) {
          const uint64_t end = Timestamp();
          m_metrics.tick_duration->Observe(static_cast<double>(end - m_tick_start_timestamp) * NANOS_INV);
          m_tick_start_timestamp = end;
        }
        else {
          m_tick_start_timestamp = Timestamp();
        }
      }
    }
    else {
      // To signal the end to the environment. The end will come with the "last" observations.
      set_state(InternalState::terminating);

      if (m_end_requested) {
        spdlog::info("Trial [{}] - Ending on request", m_id);
      }
      else {
        new_special_event("Maximum number of steps reached");
        spdlog::info("Trial [{}] - Ending on configured maximum number of steps [{}]", m_id, m_max_steps);
      }

      SPDLOG_DEBUG("Trial [{}] - Sending last actions to environment [{}]", m_id, m_env->name());
      m_env->dispatch_actions(make_action_set(), true);
    }
  }
}

ClientActor* Trial::get_join_candidate(const std::string& actor_name, const std::string& actor_class) const {
  if (m_state != InternalState::pending) {
    throw MakeException("Wrong trial state for actor to join [{}]", static_cast<int>(m_state));
  }

  ClientActor* candidate = nullptr;

  if (!actor_name.empty()) {
    auto actor_index_itor = m_actor_indexes.find(actor_name);
    if (actor_index_itor == m_actor_indexes.end()) {
      throw MakeException("Actor name unknown: [{}]", actor_name);
    }

    if (actor_index_itor->second >= m_actors.size()) {
      throw MakeException("Internal error: Bad actor index [{}] vs [{}]", actor_index_itor->second, m_actors.size());
    }
    auto& actor = m_actors[actor_index_itor->second];

    if (!actor_class.empty() && actor->actor_class() != actor_class) {
      throw MakeException("Actor does not match requested class: [{}] vs [{}]", actor_class, actor->actor_class());
    }

    candidate = dynamic_cast<ClientActor*>(actor.get());
    if (candidate == nullptr) {
      throw MakeException("Actor is not a 'client' type actor");
    }

    if (candidate->has_joined()) {
      throw MakeException("Actor has already joined");
    }
  }
  else if (!actor_class.empty()) {
    for (auto& actor : m_actors) {
      if (!actor->has_joined() && actor->actor_class() == actor_class) {
        auto available_actor = dynamic_cast<ClientActor*>(actor.get());
        if (available_actor != nullptr) {
          candidate = available_actor;
          break;
        }
      }
    }

    if (candidate == nullptr) {
      throw MakeException("Could not find actor of class [{}] to join", actor_class);
    }
  }
  else {
    throw MakeException("Must specify either actor name or actor class");
  }

  return candidate;
}

void Trial::set_state(InternalState new_state) {
  const std::lock_guard lg(m_state_lock);

  bool invalid_transition = false;
  switch (m_state) {
  case InternalState::unknown:
    invalid_transition = (new_state != InternalState::initializing);
    break;

  case InternalState::initializing:
    invalid_transition = (new_state != InternalState::pending);
    break;

  case InternalState::pending:
    invalid_transition = (new_state != InternalState::running);
    break;

  case InternalState::running:
    invalid_transition = (new_state != InternalState::terminating);
    break;

  case InternalState::terminating:
    invalid_transition = (new_state != InternalState::ended && new_state != InternalState::terminating);
    break;

  case InternalState::ended:
    if (new_state != InternalState::ended) {
      invalid_transition = true;
    }
    else {
      // Shouldn't happen, but acceptable
      spdlog::debug("Trial [{}] already ended: cannot end again", m_id);
    }
    break;
  }

  if (invalid_transition && new_state != InternalState::ended) {
    throw MakeException("Cannot switch trial state from [{}] to [{}]", get_trial_state_string(m_state),
                        get_trial_state_string(new_state));
  }

  if (m_state != new_state) {
    SPDLOG_TRACE("Trial [{}] - New state [{}] at tick [{}]", m_id, get_trial_state_string(new_state), m_tick_id);
    m_state = new_state;

    // TODO: Find a better way so we don't have to be locked when calling out
    //       Right now it is necessary to make sure we don't miss state and they are seen in-order
    m_orchestrator->notify_watchers(*this);

    if (new_state == InternalState::ended) {
      flush_samples();

      m_end_timestamp = Timestamp();

      if (m_metrics.trial_duration != nullptr) {
        m_metrics.trial_duration->Observe(static_cast<double>(m_end_timestamp - m_start_timestamp) * NANOS_INV);
      }
    }
  }
}

void Trial::refresh_activity() { m_last_activity = Timestamp(); }

bool Trial::is_stale() {
  if (m_state == InternalState::ended) {
    return false;
  }

  const auto inactivity_period = Timestamp() - m_last_activity;
  const bool stale = (inactivity_period > m_max_inactivity);

  if (stale) {
    new_special_event("Stale trial");
    if (m_state == InternalState::terminating) {
      spdlog::warn("Trial [{}] - Became stale while ending", m_id);
    }
  }

  return stale;
}

void Trial::set_info(cogmentAPI::TrialInfo* info, bool with_observations, bool with_actors) {
  if (info == nullptr) {
    spdlog::error("Trial [{}] request for info with no storage", m_id);
    return;
  }

  uint64_t end;
  if (m_end_timestamp == 0) {
    end = Timestamp();
  }
  else {
    end = m_end_timestamp;
  }
  info->set_trial_duration(end - m_start_timestamp);
  info->set_trial_id(m_id);

  // The state and tick may not be synchronized here, but it is better
  // to have the latest state (as opposed to the state of the sample).
  info->set_state(get_trial_api_state(m_state));

  auto sample = get_last_sample();
  if (sample == nullptr) {
    info->set_tick_id(m_tick_id);
    return;
  }

  if (m_env != nullptr) {
    info->set_env_name(m_env->name());
  }

  // We want to make sure the tick_id and observation are from the same tick
  const uint64_t tick = sample->info().tick_id();
  info->set_tick_id(tick);
  if (with_observations && sample->has_observations()) {
    info->mutable_latest_observation()->CopyFrom(sample->observations());
  }

  if (with_actors && m_state >= InternalState::pending) {
    for (auto& actor : m_actors) {
      auto trial_actor = info->add_actors_in_trial();
      trial_actor->set_actor_class(actor->actor_class());
      trial_actor->set_name(actor->actor_name());
    }
  }
}

}  // namespace cogment
