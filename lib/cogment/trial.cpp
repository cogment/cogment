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
#include "cogment/utils.h"
#include "cogment/stub_pool.h"

#include "spdlog/spdlog.h"

#include <limits>

namespace cogment {

constexpr int64_t AUTO_TICK_ID = -1;     // The actual tick ID will be determined by the Orchestrator
constexpr int64_t NO_DATA_TICK_ID = -2;  // When we have received no data (different from default/empty data)
constexpr uint64_t MAX_TICK_ID = static_cast<uint64_t>(std::numeric_limits<int64_t>::max());
const std::string ENVIRONMENT_NAME("env");

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

  throw MakeException<std::out_of_range>("Unknown trial state for string [%d]", static_cast<int>(state));
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

  throw MakeException<std::out_of_range>("Unknown trial state for api: [%d]", static_cast<int>(state));
}

Trial::Trial(Orchestrator* orch, std::unique_ptr<DatalogService> log, const std::string& user_id, const std::string& id, const Metrics& met) :
    m_orchestrator(orch),
    m_metrics(met),
    m_tick_start_timestamp(0),
    m_id(id),
    m_user_id(user_id),
    m_state(InternalState::unknown),
    m_env_last_obs(false),
    m_end_requested(false),
    m_tick_id(0),
    m_start_timestamp(Timestamp()),
    m_end_timestamp(0),
    m_gathered_actions_count(0),
    m_datalog(std::move(log)) {
  SPDLOG_TRACE("Trial [{}] - Constructor", m_id);

  set_state(InternalState::initializing);
  refresh_activity();
}

Trial::~Trial() {
  SPDLOG_TRACE("Trial [{}] - Destructor", m_id);

  if (m_state != InternalState::unknown && m_state != InternalState::ended) {
    spdlog::error("Trial [{}] - Internal error: Trying to destroy trial before it is ended", m_id);
  }

  // Destroy components while this trial instance still exists
  m_env.reset();
  m_actors.clear();
}

const std::unique_ptr<Actor>& Trial::actor(const std::string& name) const {
  auto actor_index = m_actor_indexes.at(name);
  return m_actors[actor_index];
}

void Trial::advance_tick() {
  SPDLOG_TRACE("Trial [{}] - Tick [{}] is done", m_id, m_tick_id);

  m_tick_id++;
  if (m_tick_id > MAX_TICK_ID) {
    throw MakeException("Tick id has reached the limit");
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
      throw MakeException("Environment repeated a tick id: [%llu]", new_tick_id);
    }

    if (new_tick_id > MAX_TICK_ID) {
      throw MakeException("Tick id from environment is too large");
    }

    if (new_tick_id > m_tick_id) {
      throw MakeException("Environment skipped tick id: [%llu] vs [%llu]", new_tick_id, m_tick_id);
    }
  }

  auto sample = get_last_sample();
  if (sample != nullptr) { 
    *(sample->mutable_observations()) = std::move(obs);
  }
  else {
    spdlog::warn("Trial [{}] - No sample for new observation. Trial may have ended.", m_id);
    return;
  }
}

void Trial::new_special_event(std::string_view desc) {
  auto sample = get_last_sample();
  if (sample == nullptr) { 
    spdlog::debug("Trial [{}] - No sample for special event [{}]. Trial may have ended.", m_id, desc);
    return;
  }

  sample->mutable_info()->add_special_events(desc.data(), desc.size());
}

cogmentAPI::DatalogSample* Trial::get_last_sample() {
  const std::lock_guard<std::mutex> lg(m_sample_lock);
  if (!m_step_data.empty()) {
    return &(m_step_data.back());
  }
  else {
    return nullptr;
  }
}

cogmentAPI::DatalogSample& Trial::make_new_sample() {
  const std::lock_guard<std::mutex> lg(m_sample_lock);

  if (!m_step_data.empty()) {
    m_step_data.back().mutable_info()->set_state(get_trial_api_state(m_state));
  }

  m_step_data.emplace_back();
  auto& sample = m_step_data.back();

  auto sample_actions = sample.mutable_actions();
  sample_actions->Reserve(m_actors.size());
  for (size_t index = 0; index < m_actors.size(); index++) {
    auto no_action = sample_actions->Add();
    no_action->set_tick_id(NO_DATA_TICK_ID);
  }
  m_gathered_actions_count = 0;

  auto info = sample.mutable_info();
  info->set_tick_id(m_tick_id);
  info->set_timestamp(Timestamp());

  return sample;
}

void Trial::flush_samples() {
  const std::lock_guard<std::mutex> lg(m_sample_lock);
  SPDLOG_TRACE("Trial [{}] - Flushing last samples", m_id);

  if (!m_step_data.empty()) {
    m_step_data.back().mutable_info()->set_state(get_trial_api_state(m_state));
  }

  for (auto& sample : m_step_data) {
    m_datalog->add_sample(std::move(sample));
  }
  m_step_data.clear();
}

void Trial::prepare_actors() {
  for (const auto& actor_info : m_params.actors()) {
    auto url = actor_info.endpoint();
    const auto& actor_class = m_orchestrator->get_trial_spec().get_actor_class(actor_info.actor_class());

    if (url == "client") {
      std::optional<std::string> config;
      if (actor_info.has_config()) {
        config = actor_info.config().content();
      }
      auto client_actor = std::make_unique<ClientActor>(this, actor_info.name(), &actor_class, actor_info.implementation(), config);
      m_actors.push_back(std::move(client_actor));
    }
    else {
      std::optional<std::string> config;
      if (actor_info.has_config()) {
        config = actor_info.config().content();
      }
      auto stub_entry = m_orchestrator->agent_pool()->get_stub_entry(url);
      auto agent_actor = std::make_unique<ServiceActor>(this, actor_info.name(), &actor_class, actor_info.implementation(),
                                                 stub_entry, config);
      m_actors.push_back(std::move(agent_actor));
    }

    m_actor_indexes.emplace(actor_info.name(), m_actors.size() - 1);
  }
}

void Trial::prepare_environment() {
  auto stub_entry = m_orchestrator->env_pool()->get_stub_entry(m_params.environment().endpoint());

  std::optional<std::string> config;
  if (m_params.environment().has_config()) {
    config = m_params.environment().config().content();
  }

  m_env = std::make_unique<Environment>(this, ENVIRONMENT_NAME, m_params.environment().implementation(), stub_entry, config);
}

void Trial::start(cogmentAPI::TrialParams&& params) {
  SPDLOG_TRACE("Trial [{}] - Starting", m_id);

  if (m_state != InternalState::initializing) {
    throw MakeException("Trial [%s] is not in proper state to start: [%s]", m_id.c_str(),
                        get_trial_state_string(m_state));
  }

  m_params = std::move(params);
  SPDLOG_DEBUG("Trial [{}] - Configuring with parameters: {}", m_id, m_params.DebugString());

  m_datalog->start(m_id, m_user_id, m_params);

  prepare_actors();
  prepare_environment();

  make_new_sample();  // First sample

  set_state(InternalState::pending);

  auto self = shared_from_this();
  thread_pool().push("Trial starting", [self]() {
    try {
      auto env_ready = self->m_env->init();

      std::vector<std::future<void>> actors_ready;
      for (const auto& actor : self->m_actors) {
        actors_ready.push_back(actor->init());
      }

      for (size_t index = 0 ; index < actors_ready.size() ; index++) {
        SPDLOG_TRACE("Trial [{}] - Waiting on actor [{}]...", self->m_id, self->m_actors[index]->actor_name());
        actors_ready[index].wait();
      }
      SPDLOG_TRACE("Trial [{}] - All actors started", self->m_id);

      auto first_obs = env_ready.get();
      SPDLOG_TRACE("Trial [{}] - Environment [{}] started", self->m_id, self->m_env->name());

      self->new_obs(std::move(first_obs));
      self->set_state(InternalState::running);

      // Send the initial state
      self->dispatch_observations(false);
    }
    catch(const std::exception& exc) {
      spdlog::error("Trial [{}] - Failed to start [{}]", self->m_id, exc.what());
    }
    catch(...) {
      spdlog::error("Trial [{}] - Failed to start on unknown exception", self->m_id);
    }
  });

  spdlog::debug("Trial [{}] - Configured", m_id);
}

void Trial::reward_received(const cogmentAPI::Reward& reward, const std::string& sender) {
  if (m_state < InternalState::pending) {
    spdlog::warn("Too early for trial [{}] to receive rewards.", m_id);
    return;
  }

  auto sample = get_last_sample();
  if (sample != nullptr) {
    const std::lock_guard<std::mutex> lg(m_reward_lock);
    auto new_rew = sample->add_rewards();
    *new_rew = reward;  // TODO: The reward may have a wildcard (or invalid) receiver, do we want that in the sample?
  }
  else {
    spdlog::warn("Trial [{}] - No sample for reward received from [{}]. Trial may have ended.", m_id, sender);
    return;
  }

  // TODO: Decide what to do with timed rewards (send anything present and past, hold future?)
  if (reward.tick_id() != AUTO_TICK_ID && reward.tick_id() != static_cast<int64_t>(m_tick_id)) {
    spdlog::error("Invalid reward tick from [{}]: [{}] (current tick id: [{}])", sender, reward.tick_id(), m_tick_id);
    return;
  }

  // Rewards are not dispatched as we receive them. They are accumulated, and sent once
  // per update.
  bool valid_name = for_actors(reward.receiver_name(), [this, &reward, &sender](auto actor) {
      // Normally we should have only one source when receiving
      for (const auto& src : reward.sources()) {
        actor->add_immediate_reward_src(src, sender, m_tick_id);
      }
  });

  if (!valid_name) {
    spdlog::error("Trial [{}] - Unknown receiver as reward destination [{}]", m_id, reward.receiver_name());
  }
}

void Trial::message_received(const cogmentAPI::Message& message, const std::string& sender) {
  if (m_state < InternalState::pending) {
    spdlog::warn("Too early for trial [{}] to receive messages.", m_id);
    return;
  }

  auto sample = get_last_sample();
  if (sample != nullptr) {
    const std::lock_guard<std::mutex> lg(m_sample_message_lock);
    auto new_msg = sample->add_messages();
    *new_msg = message;
  }
  else {
    spdlog::warn("Trial [{}] - No sample for message received from [{}]. Trial may have ended.", m_id, sender);
    return;
  }

  // TODO: Decide what to do with timed messages (send anything present and past, hold future?)
  if (message.tick_id() != AUTO_TICK_ID && message.tick_id() != static_cast<int64_t>(m_tick_id)) {
    spdlog::error("Invalid message tick from [{}]: [{}] (current tick id: [{}])", sender, message.tick_id(), m_tick_id);
    return;
  }

  // Message is not dispatched as we receive it. It is accumulated, and sent once
  // per update.
  if (message.receiver_name() == ENVIRONMENT_NAME) {
    m_env->dispatch_message(message, sender, m_tick_id);
  }
  else {
    bool valid_name = for_actors(message.receiver_name(), [this, &message, &sender](auto actor) {
        actor->add_immediate_message(message, sender, m_tick_id);
    });

    if (!valid_name) {
      spdlog::error("Trial [{}] - Unknown receiver as message destination [{}]", m_id, message.receiver_name());
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
        if (actor->actor_class()->name == class_name) {
          at_least_one = true;
          func(actor.get());
        }
      }

      return at_least_one;
    }

    const auto actor_index_itor = m_actor_indexes.find(actor_name);
    if (actor_index_itor != m_actor_indexes.end()) {
      const uint32_t index = actor_index_itor->second;
      if (m_actors[index]->actor_class()->name == class_name) {
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
    spdlog::warn("Trial [{}] - No sample to dispatch an observation. Trial may have ended.", m_id);
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
  static constexpr uint64_t MIN_NB_BUFFERED_SAMPLES = 2;  // Because of the way we use the buffer
  static constexpr uint64_t NB_BUFFERED_SAMPLES = 5;      // Could be an external setting
  static constexpr uint64_t LOG_BATCH_SIZE = 1;           // Could be an external setting
  static constexpr uint64_t LOG_TRIGGER_SIZE = NB_BUFFERED_SAMPLES + LOG_BATCH_SIZE - 1;
  static_assert(NB_BUFFERED_SAMPLES >= MIN_NB_BUFFERED_SAMPLES);
  static_assert(LOG_BATCH_SIZE > 0);

  const std::lock_guard<std::mutex> lg(m_sample_lock);

  // Send overflow to log
  if (m_step_data.size() >= LOG_TRIGGER_SIZE) {
    while (m_step_data.size() >= NB_BUFFERED_SAMPLES) {
      m_datalog->add_sample(std::move(m_step_data.front()));
      m_step_data.pop_front();
    }
  }
}

cogmentAPI::ActionSet Trial::make_action_set() {
  // TODO: Look into merging data formats so we don't have to copy all actions every time
  cogmentAPI::ActionSet action_set;
  action_set.set_timestamp(Timestamp());

  action_set.set_tick_id(m_tick_id);

  const std::lock_guard<std::mutex> lg(m_sample_lock);
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

  try {
    SPDLOG_DEBUG("Trial [{}] - Sending last actions to environment [{}]", m_id, m_env->name());

    // Send the actions we have so far (may be a partial set) if not received 'LAST'
    m_env->dispatch_actions(make_action_set(), true);

    SPDLOG_DEBUG("Trial [{}] - Waiting (max 60 sec) for environment [{}] to acknowledge 'LAST'", m_id, m_env->name());
    auto fut = m_env->last_ack();
    if (!fut.valid()) {
      throw std::future_error(std::future_errc::no_state);
    }

    auto status = fut.wait_for(timeout);
    switch(status) {
    case std::future_status::deferred:
      throw MakeException("Deferred last data"); 
    case std::future_status::ready:
      break;
    case std::future_status::timeout:
      throw MakeException("Last data wait timed out"); 
    }

    dispatch_observations(true);
    return true;
  }
  catch(const std::exception& exc) {
    spdlog::error("Trial [{}] - Environment [{}] last ack failed [{}]", m_id, m_env->name(), exc.what());
  }
  catch(...) {
    spdlog::error("Trial [{}] - Environment [{}] last ack failed", m_id, m_env->name());
  }

  return false;
}

void Trial::finalize_actors() {
  SPDLOG_DEBUG("Trial [{}] - Waiting (max 30 sec per actor) for all actors to acknowledge 'LAST'", m_id);
  static constexpr auto timeout = std::chrono::seconds(30);

  for (const auto& actor : m_actors) {
    try {  // We don't want one actor to prevent the end to be sent to all other actors
      auto fut = actor->last_ack();
      if (!fut.valid()) {
        throw std::future_error(std::future_errc::no_state);
      }

      auto status = fut.wait_for(timeout);
      switch(status) {
      case std::future_status::deferred:
        throw MakeException("Deferred last data"); 
      case std::future_status::ready:
        break;
      case std::future_status::timeout:
        throw MakeException("Last data wait timed out"); 
      }
    }
    catch(const std::exception& exc) {
      spdlog::error("Trial [{}] - Actor [{}] last ack failed [{}]", m_id, actor->actor_name(), exc.what());
    }
    catch(...) {
      spdlog::error("Trial [{}] - Actor [{}] last ack failed", m_id, actor->actor_name());
    }
  }
}

void Trial::finish() {
  SPDLOG_TRACE("Trial [{}] - finish()", m_id);

  const std::lock_guard<std::shared_mutex> lg(m_terminating_lock);

  if (m_state == InternalState::running) {
    set_state(InternalState::terminating);
  }
  else if (m_state >= InternalState::terminating) {
    SPDLOG_DEBUG("Trial [{}] - Finish called but already terminating", m_id);
    return;
  }
  else {
    spdlog::error("Trial [{}] cannot finish in current state: [{}]", m_id, get_trial_state_string(m_state));
    return;
  }

  auto self = shared_from_this();
  thread_pool().push("Trial finishing", [self]() {
    bool env_finalize_success = self->finalize_env();
    if (env_finalize_success) {
      self->finalize_actors();
    }

    try {
      std::string details;
      if (!env_finalize_success) {
        spdlog::warn("Trial [{}] - Force ending all actors", self->m_id);
        details = "Environment failed to end properly";
      }

      SPDLOG_DEBUG("Trial [{}] - Notifying all that the trial has ended", self->m_id);
      self->m_env->trial_ended(details);
      for (const auto& actor : self->m_actors) {
        actor->trial_ended(details);
      }
    }
    catch(const std::exception& exc) {
      spdlog::error("Trial [{}] - Component end failed [{}]", self->m_id, exc.what());
    }
    catch(...) {
      spdlog::error("Trial [{}] - component end failed", self->m_id);
    }

    self->set_state(InternalState::ended);
  });
}

void Trial::request_end() {
  spdlog::info("Trial [{}] - End requested", m_id);
  new_special_event("End requested");
  m_end_requested = true;
}

void Trial::terminate(const std::string& details) {
  SPDLOG_TRACE("Trial [{}] - terminate()", m_id);

  const std::lock_guard<std::shared_mutex> lg(m_terminating_lock);

  // Debatable if we want to let an ongoing ending trial finish, or we want to force it to end still.
  // The use case of lots of unresponsive actors could be a reason to force a quick end and not wait.
  if (m_state >= InternalState::terminating) {
    if (m_state == InternalState::terminating) {
      SPDLOG_DEBUG("Trial [{}] - Will not force termination: already ongoing [{}]", m_id, details);
    }
    return;
  }

  new_special_event("Forced termination: " + details);

  try {
    set_state(InternalState::terminating);
    spdlog::warn("Trial [{}] - Being forced to end [{}]", m_id, details);

    try {
      m_env->trial_ended(details);
    }
    catch(...) {
      spdlog::error("Trial [{}] - Environment [{}] termination failed", m_id, m_env->name());
    }

    for (const auto& actor : m_actors) {
      try {
        actor->trial_ended(details);
      }
      catch(...) {
        spdlog::error("Trial [{}] - Actor [{}] termination failed", m_id, actor->actor_name());
      }
    }

    set_state(InternalState::ended);
  }
  catch(...) {
    spdlog::error("Trial [{}] - Termination error", m_id);
  }
}

void Trial::env_observed(const std::string& env_name, cogmentAPI::ObservationSet&& obs, bool last) {
  std::shared_lock<std::shared_mutex> terminating_guard(m_terminating_lock);
  refresh_activity();

  if (m_state < InternalState::pending) {
    spdlog::warn("Trial [{}] - Environemnt [{}] too early to receive observations.", m_id, env_name);
    return;
  }
  if (m_state == InternalState::ended) {
    spdlog::info("Trial [{}] - Environment [{}] observations arrived after end of trial. Data will be dropped [{}].", m_id, env_name, last);
    return;
  }

  advance_tick();
  make_new_sample();
  new_obs(std::move(obs));

  if (!last) {
    dispatch_observations(false);
    cycle_buffer();
  }
  else if (m_state != InternalState::terminating) {
    terminating_guard.unlock();
    spdlog::info("Trial [{}] - Environment has ended the trial", m_id);
    new_special_event("Evironment ended trial");
    finish();
  }
}

void Trial::actor_acted(const std::string& actor_name, const cogmentAPI::Action& action) {
  std::shared_lock<std::shared_mutex> terminating_guard(m_terminating_lock);
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
    spdlog::warn("Trial [{}] - No sample for action received from [{}]. Trial may have ended.", m_id, actor_name);
    return;
  }
  auto sample_action = sample->mutable_actions(actor_index);
  if (sample_action->tick_id() != NO_DATA_TICK_ID) {
    spdlog::warn("Trial [{}] - Actor [{}] multiple actions received for same step. Only the first one will be used.", m_id, actor_name);
    return;
  }

  // TODO: Determine what we want to do in case of actions in the past or future
  if (action.tick_id() != AUTO_TICK_ID && action.tick_id() != static_cast<int64_t>(m_tick_id)) {
    spdlog::warn("Trial [{}] - Actor [{}] invalid action step: [{}] vs [{}]. Default will be used.", m_id, actor_name, action.tick_id(), m_tick_id);
  }

  SPDLOG_TRACE("Trial [{}] - Actor [{}] received action for tick [{}].", m_id, actor_name, m_tick_id);
  *sample_action = action;

  const auto new_count = ++m_gathered_actions_count;
  if (new_count == m_actors.size()) {
    SPDLOG_TRACE("Trial [{}] - All actions received for tick [{}]", m_id, m_tick_id);

    const auto max_steps = m_params.max_steps();
    if ((max_steps == 0 || m_tick_id < max_steps) && !m_end_requested) {
      if (m_env->started()) {
        m_env->dispatch_actions(make_action_set(), false);

        // We put this here to try to be outside the first and last tick (i.e. overhead)
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
        spdlog::error("Trial [{}] - Environment [{}] not ready for fist action set.", m_id, m_env->name());
      }
    }
    else if (m_state != InternalState::terminating) {
      if (m_end_requested) {
        spdlog::info("Trial [{}] - Ending on request", m_id);
      }
      else {
        new_special_event("Maximum number of steps reached");
        spdlog::info("Trial [{}] - Ending on configured maximum number of steps [{}]", m_id, max_steps);
      }
      terminating_guard.unlock();
      finish();
    }
  }
}

ClientActor* Trial::get_join_candidate(const std::string& actor_name, const std::string& actor_class) const {
  if (m_state != InternalState::pending) {
    throw MakeException("Trial [%s] - Wrong state to be joined", m_id.c_str());
  }

  ClientActor* candidate = nullptr;

  if (!actor_name.empty()) {
    auto actor_index = m_actor_indexes.at(actor_name);
    auto& actor = m_actors.at(actor_index);

    if (!actor_class.empty() && actor->actor_class()->name != actor_class) {
      throw MakeException("Trial [%s] - Actor [%s] does not match requested class: [%s] vs [%s]", 
                          m_id.c_str(), actor_name.c_str(), actor_class.c_str(), actor->actor_class()->name.c_str());
    }

    candidate = dynamic_cast<ClientActor*>(actor.get());
    if (candidate == nullptr) {
      throw MakeException("Trial [%s] - Actor [%s] is not a 'client' type actor", m_id.c_str(), actor_name.c_str());
    }

    if (candidate->is_active()) {
      throw MakeException("Trial [%s] - Actor [%s] has already joined", m_id.c_str(), actor_name.c_str());
    }
  }
  else if (!actor_class.empty()) {
    for (auto& actor : m_actors) {
      if (!actor->is_active() && actor->actor_class()->name == actor_class) {
        auto available_actor = dynamic_cast<ClientActor*>(actor.get());
        if (available_actor != nullptr) {
          candidate = available_actor;
          break;
        }
      }
    }

    if (candidate == nullptr) {
      throw MakeException("Trial [%s] - Could not find actor of class [%s] to join", m_id.c_str(), actor_class.c_str());
    }
  }
  else {
    throw MakeException<std::invalid_argument>("Trial [%s] - Must specify either actor name or actor class", m_id.c_str());
  }

  return candidate;
}

void Trial::set_state(InternalState new_state) {
  const std::lock_guard<std::mutex> lg(m_state_lock);

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

  if (invalid_transition) {
    throw MakeException("Cannot switch trial state from [%s] to [%s]", get_trial_state_string(m_state),
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

void Trial::refresh_activity() { m_last_activity = std::chrono::steady_clock::now(); }

ThreadPool& Trial::thread_pool() { return m_orchestrator->thread_pool(); }

bool Trial::is_stale() {
  if (m_state == InternalState::ended) {
    return false;
  }

  const auto inactivity_period = std::chrono::steady_clock::now() - m_last_activity;
  const auto& max_inactivity = m_params.max_inactivity();

  const bool stale = (inactivity_period > std::chrono::seconds(max_inactivity));
  const bool result = (max_inactivity > 0 && stale);

  if (result) {
    new_special_event("Stale trial");
    if (m_state == InternalState::terminating) {
      spdlog::warn("Trial [{}] - Became stale while ending", m_id);
    }
  }

  return result;
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

  // We want to make sure the tick_id and observation are from the same tick
  const uint64_t tick = sample->info().tick_id();
  info->set_tick_id(tick);
  if (with_observations && sample->has_observations()) {
    info->mutable_latest_observation()->CopyFrom(sample->observations());
  }

  if (with_actors && m_state >= InternalState::pending) {
    for (auto& actor : m_actors) {
      auto trial_actor = info->add_actors_in_trial();
      trial_actor->set_actor_class(actor->actor_class()->name);
      trial_actor->set_name(actor->actor_name());
    }
  }
}

}  // namespace cogment
