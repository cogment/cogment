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

#include "cogment/agent_actor.h"
#include "cogment/client_actor.h"
#include "cogment/utils.h"

#include "cogment/datalog/storage_interface.h"

#include "spdlog/spdlog.h"

#include <limits>

namespace cogment {

constexpr int64_t AUTO_TICK_ID = -1;     // The actual tick ID will be determined by the Orchestrator
constexpr int64_t NO_DATA_TICK_ID = -2;  // When we have received no data (different from default/empty data)
constexpr uint64_t MAX_TICK_ID = static_cast<uint64_t>(std::numeric_limits<int64_t>::max());
const std::string ENVIRONMENT_ACTOR_NAME("env");

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

cogment::TrialState get_trial_api_state(Trial::InternalState state) {
  switch (state) {
  case Trial::InternalState::unknown:
    return cogment::UNKNOWN;
  case Trial::InternalState::initializing:
    return cogment::INITIALIZING;
  case Trial::InternalState::pending:
    return cogment::PENDING;
  case Trial::InternalState::running:
    return cogment::RUNNING;
  case Trial::InternalState::terminating:
    return cogment::TERMINATING;
  case Trial::InternalState::ended:
    return cogment::ENDED;
  }

  throw MakeException<std::out_of_range>("Unknown trial state for api: [%d]", static_cast<int>(state));
}

uuids::uuid_system_generator Trial::m_id_generator;

Trial::Trial(Orchestrator* orch, std::string user_id)
    : m_orchestrator(orch),
      m_id(m_id_generator()),
      m_user_id(std::move(user_id)),
      m_state(InternalState::unknown),
      m_tick_id(0),
      m_start_timestamp(Timestamp()),
      m_end_timestamp(0) {
  SPDLOG_TRACE("New Trial [{}]", to_string(m_id));

  set_state(InternalState::initializing);
  refresh_activity();

  m_datalog_interface = orch->start_log(this);
}

Trial::~Trial() { SPDLOG_TRACE("Tearing down trial [{}]", to_string(m_id)); }

const std::unique_ptr<Actor>& Trial::actor(const std::string& name) const {
  auto actor_index = m_actor_indexes.at(name);
  return m_actors[actor_index];
}

void Trial::advance_tick() {
  SPDLOG_TRACE("Tick [{}] is done for Trial [{}]", m_tick_id, to_string(m_id));

  m_tick_id++;
  if (m_tick_id > MAX_TICK_ID) {
    throw MakeException("Tick id has reached the limit");
  }
}

void Trial::new_obs(ObservationSet&& obs) {
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
  *(sample->mutable_observations()) = std::move(obs);
}

cogment::DatalogSample* Trial::get_last_sample() {
  const std::lock_guard<std::mutex> lg(m_sample_lock);
  if (!m_step_data.empty()) {
    return &(m_step_data.back());
  }
  else {
    return nullptr;
  }
}

cogment::DatalogSample& Trial::make_new_sample() {
  const std::lock_guard<std::mutex> lg(m_sample_lock);

  if (!m_step_data.empty()) {
    m_step_data.back().mutable_trial_data()->set_state(get_trial_api_state(m_state));
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

  auto trial_data = sample.mutable_trial_data();
  trial_data->set_tick_id(m_tick_id);
  trial_data->set_timestamp(Timestamp());

  return sample;
}

void Trial::flush_samples() {
  const std::lock_guard<std::mutex> lg(m_sample_lock);

  if (!m_step_data.empty()) {
    m_step_data.back().mutable_trial_data()->set_state(get_trial_api_state(m_state));
  }

  for (auto& sample : m_step_data) {
    m_datalog_interface->add_sample(std::move(sample));
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
      auto client_actor = std::make_unique<Client_actor>(this, actor_info.name(), &actor_class, config);
      m_actors.push_back(std::move(client_actor));
    }
    else {
      std::optional<std::string> config;
      if (actor_info.has_config()) {
        config = actor_info.config().content();
      }
      auto stub_entry = m_orchestrator->agent_pool()->get_stub_entry(url);
      auto agent_actor = std::make_unique<Agent>(this, actor_info.name(), &actor_class, actor_info.implementation(),
                                                 stub_entry, config);
      m_actors.push_back(std::move(agent_actor));
    }

    m_actor_indexes.emplace(actor_info.name(), m_actors.size() - 1);
  }
}

cogment::EnvStartRequest Trial::prepare_environment() {
  m_env_entry = m_orchestrator->env_pool()->get_stub_entry(m_params.environment().endpoint());

  cogment::EnvStartRequest env_start_req;
  env_start_req.set_tick_id(m_tick_id);
  env_start_req.set_impl_name(m_params.environment().implementation());
  if (m_params.environment().has_config()) {
    *env_start_req.mutable_config() = m_params.environment().config();
  }

  for (const auto& actor_info : m_params.actors()) {
    const auto& actor_class = m_orchestrator->get_trial_spec().get_actor_class(actor_info.actor_class());
    auto actor_in_trial = env_start_req.add_actors_in_trial();
    actor_in_trial->set_actor_class(actor_class.name);
    actor_in_trial->set_name(actor_info.name());
  }

  return env_start_req;
}

void Trial::start(cogment::TrialParams params) {
  SPDLOG_TRACE("Starting Trial [{}]", to_string(m_id));

  if (m_state != InternalState::initializing) {
    throw MakeException("Trial [%s] is not in proper state to start: [%s]", to_string(m_id).c_str(),
                        get_trial_state_string(m_state));
  }

  m_params = std::move(params);
  SPDLOG_DEBUG("Configuring trial {} with parameters: {}", to_string(m_id), m_params.DebugString());

  grpc_metadata trial_header;
  trial_header.key = grpc_slice_from_static_string("trial-id");
  trial_header.value = grpc_slice_from_copied_string(to_string(m_id).c_str());
  m_headers.push_back(trial_header);
  m_call_options.headers = &m_headers;

  auto env_start_req = prepare_environment();

  prepare_actors();

  make_new_sample();  // First sample

  set_state(InternalState::pending);

  std::vector<aom::Future<void>> actors_ready;
  for (const auto& actor : m_actors) {
    actors_ready.push_back(actor->init());
  }

  auto env_ready = m_env_entry->get_stub().OnStart(std::move(env_start_req), m_call_options).then_expect([](auto rep) {
    if (!rep) {
      spdlog::error("failed to connect to environment");
      try {
        std::rethrow_exception(rep.error());
      } catch (std::exception& e) {
        spdlog::error("OnStart exception: {}", e.what());
      }
    }

    return rep;
  });

  auto self = shared_from_this();
  SPDLOG_TRACE("Trial [{}]: waiting for actors and environment...", to_string(m_id));
  join(env_ready, concat(actors_ready.begin(), actors_ready.end()))
      .then([this](auto env_rep) {
        new_obs(std::move(*env_rep.mutable_observation_set()));

        set_state(InternalState::running);

        run_environment();

        // Send the initial state
        dispatch_observations();
      })
      .finally([self](auto) { SPDLOG_TRACE("Trial [{}]: All components started.", to_string(self->m_id)); });

  spdlog::debug("Trial {} is configured", to_string(m_id));
}

void Trial::reward_received(const cogment::Reward& reward, const std::string& sender) {
  if (m_state < InternalState::pending) {
    spdlog::warn("Too early for trial [{}] to receive rewards.", to_string(m_id));
    return;
  }

  auto sample = get_last_sample();
  if (sample != nullptr) {
    const std::lock_guard<std::mutex> lg(m_reward_lock);
    auto new_rew = sample->add_rewards();
    *new_rew = reward;
  }
  else {
    return;
  }

  // TODO: Decide what to do with timed rewards (send anything present and past, hold future?)
  if (reward.tick_id() != AUTO_TICK_ID && reward.tick_id() != static_cast<int64_t>(m_tick_id)) {
    spdlog::error("Invalid reward tick from [{}]: [{}] (current tick id: [{}])", sender, reward.tick_id(), m_tick_id);
    return;
  }

  // Rewards are not dispatched as we receive them. They are accumulated, and sent once
  // per update.
  auto actor_index_itor = m_actor_indexes.find(reward.receiver_name());
  if (actor_index_itor != m_actor_indexes.end()) {
    // Normally we should have only one source when receiving
    for (const auto& src : reward.sources()) {
      m_actors[actor_index_itor->second]->add_immediate_reward_src(src, sender, m_tick_id);
    }
  }
  else {
    spdlog::error("Unknown actor name as reward destination [{}]", reward.receiver_name());
  }
}

void Trial::message_received(const cogment::Message& message, const std::string& sender) {
  if (m_state < InternalState::pending) {
    spdlog::warn("Too early for trial [{}] to receive messages.", to_string(m_id));
    return;
  }

  auto sample = get_last_sample();
  if (sample != nullptr) {
    const std::lock_guard<std::mutex> lg(m_sample_message_lock);
    auto new_msg = sample->add_messages();
    *new_msg = message;
  }
  else {
    return;
  }

  // TODO: Decide what to do with timed messages (send anything present and past, hold future?)
  if (message.tick_id() != AUTO_TICK_ID && message.tick_id() != static_cast<int64_t>(m_tick_id)) {
    spdlog::error("Invalid message tick from [{}]: [{}] (current tick id: [{}])", sender, message.tick_id(), m_tick_id);
    return;
  }

  // Message is not dispatched as we receive it. It is accumulated, and sent once
  // per update.
  auto actor_index_itor = m_actor_indexes.find(message.receiver_name());
  if (actor_index_itor != m_actor_indexes.end()) {
    m_actors[actor_index_itor->second]->add_immediate_message(message, sender, m_tick_id);
  }
  else if (message.receiver_name() == ENVIRONMENT_ACTOR_NAME) {
    const std::lock_guard<std::mutex> lg(m_env_message_lock);
    m_env_message_accumulator.emplace_back(message);
    m_env_message_accumulator.back().set_tick_id(m_tick_id);
    m_env_message_accumulator.back().set_sender_name(sender);
  }
  else {
    spdlog::error("Unknown receiver name as message destination [{}]", message.receiver_name());
  }
}

void Trial::next_step(EnvActionReply&& reply) {
  // Rewards and messages are for last step here
  for (auto&& rew : reply.rewards()) {
    reward_received(rew, ENVIRONMENT_ACTOR_NAME);
  }
  for (auto&& msg : reply.messages()) {
    message_received(msg, ENVIRONMENT_ACTOR_NAME);
  }

  advance_tick();

  make_new_sample();
  new_obs(std::move(*reply.mutable_observation_set()));
}

void Trial::dispatch_observations() {
  if (m_state == InternalState::ended) {
    return;
  }
  const bool ending = (m_state == InternalState::terminating);

  auto sample = get_last_sample();
  if (sample == nullptr) {
    return;
  }
  const auto& observations = sample->observations();

  std::uint32_t actor_index = 0;
  for (const auto& actor : m_actors) {
    auto obs_index = observations.actors_map(actor_index);
    cogment::Observation obs;
    obs.set_tick_id(m_tick_id);
    obs.set_timestamp(observations.timestamp());
    *obs.mutable_data() = observations.observations(obs_index);
    actor->dispatch_tick(std::move(obs), ending);

    ++actor_index;
  }

  if (ending) {
    set_state(InternalState::ended);
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
      m_datalog_interface->add_sample(std::move(m_step_data.front()));
      m_step_data.pop_front();
    }
  }
}

void Trial::run_environment() {
  auto self = shared_from_this();

  // Launch the main update stream
  auto streams = m_env_entry->get_stub().OnAction(m_call_options);

  m_outgoing_actions = std::move(std::get<0>(streams));
  auto incoming_updates = std::move(std::get<1>(streams));

  incoming_updates
      .for_each([this](auto update) {
        SPDLOG_TRACE("Received new observation set for Trial [{}]", to_string(m_id));

        if (m_state == InternalState::ended) {
          return;
        }
        if (update.final_update()) {
          // TODO: There is a timing issue with the terminate() function which can really only
          //       be properly resolved with the gRPC API 2.0
          set_state(InternalState::terminating);
        }

        next_step(std::move(update));
        dispatch_observations();
        cycle_buffer();
      })
      .finally([self](auto) {
        // We are holding on to self until the rpc is over.
        // This is important because we we still need to finish
        // this call while the trial may be "deleted".
      });
}

cogment::EnvActionRequest Trial::make_action_request() {
  // TODO: Look into merging data formats so we don't have to copy all actions every time
  cogment::EnvActionRequest req;
  auto action_set = req.mutable_action_set();

  action_set->set_tick_id(m_tick_id);

  const std::lock_guard<std::mutex> lg(m_sample_lock);
  auto& sample = m_step_data.back();
  for (auto& act : sample.actions()) {
    if (act.tick_id() == AUTO_TICK_ID || act.tick_id() == static_cast<int64_t>(m_tick_id)) {
      action_set->add_actions(act.content());
    }
    else {
      // The registered action is not for this tick

      // TODO: Synchronize with `actor_acted` about past/future actions
      action_set->add_actions();
    }
  }

  return req;
}

void Trial::terminate() {
  {
    const std::lock_guard<std::shared_mutex> lg(m_terminating_lock);

    if (m_state >= InternalState::terminating) {
      return;
    }
    if (m_state != InternalState::running) {
      spdlog::error("Trial [{}] cannot terminate in current state: [{}]", to_string(m_id).c_str(),
                    get_trial_state_string(m_state));
      return;
    }

    set_state(InternalState::terminating);
  }

  dispatch_env_messages();

  auto self = shared_from_this();

  // Send the actions we have so far (may be a partial set)
  auto req = make_action_request();

  // Fail safe timed call because this function is also called to end abnormal/stale trials.
  auto timed_options = m_call_options;
  timed_options.deadline.tv_sec = 60;
  timed_options.deadline.tv_nsec = 0;
  timed_options.deadline.clock_type = GPR_TIMESPAN;

  m_env_entry->get_stub()
      .OnEnd(req, timed_options)
      .then([this](auto rep) {
        next_step(std::move(rep));
        dispatch_observations();
      })
      .finally([self](auto) {
        // We are holding on to self to be safe.

        if (self->m_state != InternalState::ended) {
          // One possible reason is if the environment took longer than the deadline to respond
          spdlog::error("Trial [{}] did not end normally.", to_string(self->m_id));

          // TODO: Send "ended trial" to all components?  So they don't get stuck waiting. Or cut comms?

          self->m_state = InternalState::ended;  // To enable garbage collection on this trial
        }
      });
}

void Trial::actor_acted(const std::string& actor_name, const cogment::Action& action) {
  std::shared_lock<std::shared_mutex> terminating_guard(m_terminating_lock);

  if (m_state < InternalState::pending) {
    spdlog::warn("Too early for trial [{}] to receive actions from [{}].", to_string(m_id), actor_name);
    return;
  }
  if (m_state >= InternalState::terminating) {
    spdlog::info("An action from [{}] arrived after end of trial [{}].  Action will be dropped.", actor_name,
                 to_string(m_id));
    return;
  }

  const auto itor = m_actor_indexes.find(actor_name);
  if (itor == m_actor_indexes.end()) {
    spdlog::error("Unknown actor [{}] for action received in trial [{}].", actor_name, to_string(m_id));
    return;
  }
  const auto actor_index = itor->second;

  auto sample = get_last_sample();
  if (sample == nullptr) {
    spdlog::warn("No sample to accept action from [{}] in trial [{}]. Trial may be finished.", actor_name,
                 to_string(m_id));
    return;
  }
  auto sample_action = sample->mutable_actions(actor_index);
  if (sample_action->tick_id() != NO_DATA_TICK_ID) {
    spdlog::warn("Multiple actions from [{}] for same step: only the first one will be used.", actor_name);
    return;
  }

  // TODO: Determine what we want to do in case of actions in the past or future
  if (action.tick_id() != AUTO_TICK_ID && action.tick_id() != static_cast<int64_t>(m_tick_id)) {
    spdlog::warn("Invalid action tick from [{}]: [{}] vs [{}].  Default action will be used.", actor_name,
                 action.tick_id(), m_tick_id);
  }

  SPDLOG_TRACE("Received action from actor [{}] in trial [{}] for tick [{}].", actor_name, to_string(m_id), m_tick_id);
  *sample_action = action;

  bool all_actions_received = false;
  {
    const std::lock_guard<std::mutex> lg(m_actor_lock);
    m_gathered_actions_count++;
    all_actions_received = (m_gathered_actions_count == m_actors.size());
  }

  if (all_actions_received) {
    SPDLOG_TRACE("All actions received in Trial [{}] for tick [{}]", to_string(m_id), m_tick_id);

    const auto max_steps = m_params.max_steps();
    if (max_steps == 0 || m_tick_id < max_steps) {
      if (m_outgoing_actions) {
        dispatch_env_messages();

        // We serialize the actions to prevent long lags where
        // messages arrive much later and cause errors.
        m_env_entry->serialize([this]() { m_outgoing_actions->push(make_action_request()); });
      }
      else {
        spdlog::error("Environment for trial [{}] not ready for first action set", to_string(m_id));
      }
    }
    else {
      spdlog::info("Trial [{}] reached its maximum number of steps: [{}]", to_string(m_id), max_steps);
      terminating_guard.unlock();
      terminate();
    }
  }
}

Client_actor* Trial::get_join_candidate(const TrialJoinRequest& req) {
  if (m_state != InternalState::pending) {
    return nullptr;
  }

  Actor* result = nullptr;

  switch (req.slot_selection_case()) {
  case TrialJoinRequest::kActorName: {
    auto actor_index = m_actor_indexes.at(req.actor_name());
    auto& actor = m_actors.at(actor_index);
    if (actor->is_active()) {
      result = actor.get();
    }
  } break;

  case TrialJoinRequest::kActorClass:
    for (auto& actor : m_actors) {
      if (!actor->is_active() && actor->actor_class()->name == req.actor_class()) {
        result = actor.get();
        break;
      }
    }
    break;

  case TrialJoinRequest::SLOT_SELECTION_NOT_SET:
  default:
    throw MakeException<std::invalid_argument>("Must specify either actor_name or actor_class");
  }

  if (result != nullptr && dynamic_cast<Client_actor*>(result) == nullptr) {
    throw MakeException<std::invalid_argument>("Actor name or class is not a client actor");
  }

  return static_cast<Client_actor*>(result);
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
      spdlog::debug("Trial [{}] already ended: cannot end again", to_string(m_id));
    }
    break;
  }

  if (invalid_transition) {
    throw MakeException("Cannot switch trial state from [%s] to [%s]", get_trial_state_string(m_state),
                        get_trial_state_string(new_state));
  }

  if (m_state != new_state) {
    m_state = new_state;

    if (new_state == InternalState::ended) {
      m_end_timestamp = Timestamp();
    }

    // TODO: Find a better way so we don't have to be locked when calling out
    //       Right now it is necessary to make sure we don't miss state and they are seen in-order
    m_orchestrator->notify_watchers(*this);

    if (new_state == InternalState::ended) {
      flush_samples();
    }
  }
}

void Trial::refresh_activity() { m_last_activity = std::chrono::steady_clock::now(); }

bool Trial::is_stale() const {
  const auto inactivity_period = std::chrono::steady_clock::now() - m_last_activity;
  const auto& max_inactivity = m_params.max_inactivity();

  const bool stale = (inactivity_period > std::chrono::seconds(max_inactivity));
  return (max_inactivity > 0 && stale);
}

void Trial::set_info(cogment::TrialInfo* info, bool with_observations, bool with_actors) {
  if (info == nullptr) {
    spdlog::error("Trial [{}] request for info with no storage", to_string(m_id));
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
  info->set_trial_id(to_string(m_id));

  // The state and tick may not be synchronized here, but it is better
  // to have the latest state (as opposed to the state of the sample).
  info->set_state(get_trial_api_state(m_state));

  auto sample = get_last_sample();
  if (sample == nullptr) {
    info->set_tick_id(m_tick_id);
    return;
  }

  // We want to make sure the tick_id and observation are from the same tick
  const uint64_t tick = sample->trial_data().tick_id();
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

void Trial::dispatch_env_messages() {
  const std::lock_guard<std::mutex> lg(m_env_message_lock);

  cogment::EnvMessageRequest req;
  for (auto& message : m_env_message_accumulator) {
    auto msg = req.add_messages();
    *msg = std::move(message);
  }

  if (!m_env_message_accumulator.empty()) {
    m_env_entry->serialize(
        [this, request = std::move(req)]() { m_env_entry->get_stub().OnMessage(request, m_call_options).get(); });
  }

  m_env_message_accumulator.clear();
}

}  // namespace cogment
