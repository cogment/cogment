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
#include "cogment/datalog.h"
#include "cogment/utils.h"

#include "spdlog/spdlog.h"

#include "uuid.h"

#include <limits>

namespace {
uuids::uuid_system_generator g_uuid_generator;
constexpr double NANOS_INV = 1.0 / 1'000'000'000;
}  // namespace

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

Trial::Trial(Orchestrator* orch, std::unique_ptr<DatalogService> log, const std::string& user_id, const Metrics& met) :
    m_orchestrator(orch),
    m_metrics(met),
    m_tick_start_timestamp(0),
    m_id(to_string(g_uuid_generator())),
    m_user_id(user_id),
    m_state(InternalState::unknown),
    m_tick_id(0),
    m_start_timestamp(Timestamp()),
    m_end_timestamp(0),
    m_gathered_actions_count(0),
    m_datalog(std::move(log)) {
  SPDLOG_TRACE("Trial [{}]: Constructor", m_id);

  m_env_stream_context.AddMetadata("trial-id", m_id);
  set_state(InternalState::initializing);
  refresh_activity();
}

Trial::~Trial() {
  SPDLOG_TRACE("Trial [{}]: Destructor", m_id);

  // Destroy actors while this trial instance still exists
  m_actors.clear();

  if (m_env_stream != nullptr) {
    m_env_stream->WritesDone();
    m_env_stream->Finish();
    m_env_incoming_thread.join();
  }
}

const std::unique_ptr<Actor>& Trial::actor(const std::string& name) const {
  auto actor_index = m_actor_indexes.at(name);
  return m_actors[actor_index];
}

void Trial::advance_tick() {
  SPDLOG_TRACE("Trial [{}]: Tick [{}] is done", m_id, m_tick_id);

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
  *(sample->mutable_observations()) = std::move(obs);
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
  SPDLOG_TRACE("Trial [{}]: Flushing last samples", m_id);

  if (!m_step_data.empty()) {
    m_step_data.back().mutable_trial_data()->set_state(get_trial_api_state(m_state));
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

cogmentAPI::EnvStartRequest Trial::prepare_environment() {
  cogmentAPI::EnvStartRequest env_start_req;
  env_start_req.set_tick_id(m_tick_id);
  env_start_req.set_impl_name(m_params.environment().implementation());
  if (m_params.environment().has_config()) {
    *env_start_req.mutable_config() = m_params.environment().config();
  }

  for (const auto& actor_info : m_params.actors()) {
    auto& actor_class = m_orchestrator->get_trial_spec().get_actor_class(actor_info.actor_class());
    auto actor_in_trial = env_start_req.add_actors_in_trial();
    actor_in_trial->set_actor_class(actor_class.name);
    actor_in_trial->set_name(actor_info.name());
  }

  SPDLOG_DEBUG("Trial [{}]: Start environment request: {}", m_id, env_start_req.DebugString());
  return env_start_req;
}

void Trial::start(cogmentAPI::TrialParams params) {
  SPDLOG_TRACE("Trial [{}]: Starting", m_id);

  if (m_state != InternalState::initializing) {
    throw MakeException("Trial [%s] is not in proper state to start: [%s]", m_id.c_str(),
                        get_trial_state_string(m_state));
  }

  m_params = std::move(params);
  SPDLOG_DEBUG("Trial [{}]: Configuring with parameters: {}", m_id, m_params.DebugString());

  m_datalog->start(m_id, m_params);

  m_env_entry = m_orchestrator->env_pool()->get_stub_entry(m_params.environment().endpoint());

  prepare_actors();

  make_new_sample();  // First sample

  set_state(InternalState::pending);

  std::vector<std::future<void>> actors_ready;
  for (const auto& actor : m_actors) {
    actors_ready.push_back(actor->init());
  }

  auto self = shared_from_this();

  std::promise<cogmentAPI::EnvStartReply> env_ready_promise;
  auto env_ready = env_ready_promise.get_future();
  auto env_thr = std::thread([self, prom=std::move(env_ready_promise)]() mutable {
    try {
      auto request = self->prepare_environment();
      cogmentAPI::EnvStartReply response;
      grpc::ClientContext context;
      context.AddMetadata("trial-id", self->m_id);
      auto status = self->m_env_entry->get_stub().OnStart(&context, request, &response);
      if (!status.ok()) {
        prom.set_exception(std::make_exception_ptr(MakeException("Failed to start environment for trial [%s]", 
                                                                 self->m_id.c_str())));
      } else {
        prom.set_value(std::move(response));
      }
    }
    catch (...) {
      prom.set_exception(std::current_exception());
    }
  });
  env_thr.detach();

  auto start_thr = std::thread([self, actors=std::move(actors_ready), env=std::move(env_ready)]() mutable {
    try {
      for (size_t index = 0 ; index < actors.size() ; index++) {
        SPDLOG_TRACE("Trial [{}]: Waiting on actor [{}]...", self->m_id, self->m_actors[index]->actor_name());
        actors[index].wait();
      }
      SPDLOG_TRACE("Trial [{}]: All actors started", self->m_id);

      auto env_rep = env.get();
      SPDLOG_TRACE("Trial [{}]: Environment started", self->m_id);

      self->new_obs(std::move(*env_rep.mutable_observation_set()));
      self->set_state(InternalState::running);
      self->run_environment();

      // Send the initial state
      self->dispatch_observations();
    }
    catch(const std::exception& exc) {
      spdlog::error("Trial [{}] - Failed to start [{}]", self->m_id, exc.what());
    }
    catch(...) {
      spdlog::error("Trial [{}] - Failed to start on unknown exception", self->m_id);
    }
  });
  start_thr.detach();

  spdlog::debug("Trial [{}]: Configured", m_id);
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
    return;
  }

  // TODO: Decide what to do with timed messages (send anything present and past, hold future?)
  if (message.tick_id() != AUTO_TICK_ID && message.tick_id() != static_cast<int64_t>(m_tick_id)) {
    spdlog::error("Invalid message tick from [{}]: [{}] (current tick id: [{}])", sender, message.tick_id(), m_tick_id);
    return;
  }

  // Message is not dispatched as we receive it. It is accumulated, and sent once
  // per update.
  if (message.receiver_name() == ENVIRONMENT_ACTOR_NAME) {
    const std::lock_guard<std::mutex> lg(m_env_message_lock);
    m_env_message_accumulator.emplace_back(message);
    m_env_message_accumulator.back().set_tick_id(m_tick_id);
    m_env_message_accumulator.back().set_sender_name(sender);
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

void Trial::next_step(cogmentAPI::EnvActionReply&& reply) {
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
    cogmentAPI::Observation obs;
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
      m_datalog->add_sample(std::move(m_step_data.front()));
      m_step_data.pop_front();
    }
  }
}

void Trial::run_environment() {
  auto self = shared_from_this();

  m_env_stream = m_env_entry->get_stub().OnAction(&m_env_stream_context);

  // Launch the main update stream
  m_env_incoming_thread = std::thread([this]() {
    try {
      cogmentAPI::EnvActionReply data;
      while(m_env_stream->Read(&data)) {
        SPDLOG_TRACE("Trial [{}]: Received new observation set", m_id);
        
        if (m_state == InternalState::ended) {
          return;
        }
        if (data.final_update()) {
          spdlog::info("Trial [{}] - Environment has ended the trial", m_id);
          // TODO: There is a timing issue with the terminate() function which can really only
          //       be properly resolved with the gRPC API 2.0 for the environment
          set_state(InternalState::terminating);
        }

        next_step(std::move(data));
        dispatch_observations();
        cycle_buffer();
      }
      SPDLOG_TRACE("Trial [{}]: Finished reading environment stream", m_id);
    }
    catch(const std::exception& exc) {
      spdlog::error("Trial [{}] - Environment: Error reading stream [{}]", m_id, exc.what());
    }
    catch(...) {
      spdlog::error("Trial [{}] - Environment: Unknown exception reading stream", m_id);
    }
  });
}

cogmentAPI::EnvActionRequest Trial::make_action_request() {
  // TODO: Look into merging data formats so we don't have to copy all actions every time
  cogmentAPI::EnvActionRequest req;
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
  SPDLOG_TRACE("Trial [{}]: terminate()", m_id);
  {
    const std::lock_guard<std::shared_mutex> lg(m_terminating_lock);

    if (m_state >= InternalState::terminating) {
      return;
    }
    if (m_state != InternalState::running) {
      spdlog::error("Trial [{}] cannot terminate in current state: [{}]", m_id.c_str(),
                    get_trial_state_string(m_state));
      return;
    }

    set_state(InternalState::terminating);
  }

  dispatch_env_messages();

  auto self = shared_from_this();

  // Send the actions we have so far (may be a partial set)
  auto req = make_action_request();

  auto env_thr = std::thread([self, data=std::move(req)]() {
    static constexpr auto timeout = std::chrono::seconds(60);

    try {
      grpc::ClientContext context;
      context.AddMetadata("trial-id", self->m_id);
      context.set_deadline(std::chrono::system_clock::now() + timeout);
      cogmentAPI::EnvActionReply response;
      auto status = self->m_env_entry->get_stub().OnEnd(&context, data, &response);
      if (status.ok()) {
        self->next_step(std::move(response));
        self->dispatch_observations();
      } 
      else {
        spdlog::error("Trial [{}] environment did not end normally [{}]", self->m_id, status.error_message());
        self->set_state(InternalState::ended);  // To enable garbage collection on this trial
      }
    }
    catch(const std::exception& exc) {
      spdlog::error("Trial [{}] - Environment end failed [{}]", self->m_id, exc.what());
      self->set_state(InternalState::ended);  // To enable garbage collection on this trial
    }
    catch(...) {
      spdlog::error("Trial [{}] - Environment end failed with unknown exception", self->m_id);
      self->set_state(InternalState::ended);  // To enable garbage collection on this trial
    }

    // TODO: handle case when the actors may not have received 'LAST' because of errors above
    try {
      SPDLOG_DEBUG("Waiting (max 60 sec per actor) for all actors to acknowledge the last data");
      for (const auto& actor : self->m_actors) {
        try {  // We don't want one actor to prevent the end to be sent to all other actors
          auto fut = actor->last_ack();
          if (!fut.valid()) {
            throw std::future_error(std::future_errc::no_state);
          }

          auto status = fut.wait_for(timeout);
          switch(status) {
          case std::future_status::deferred:
            throw MakeException("Defered last data"); 
          case std::future_status::ready:
            break;
          case std::future_status::timeout:
            throw MakeException("Last data wait timed out"); 
          }
        }
        catch(const std::exception& exc) {
          spdlog::error("Trial [{}] - Actor [{}] last ack failed [{}]", self->m_id, actor->actor_name(), exc.what());
        }
        catch(...) {
          spdlog::error("Trial [{}] - Actor [{}] last ack failed", self->m_id, actor->actor_name());
        }
      }

      SPDLOG_DEBUG("Notifying all actors that the trial has ended");
      for (const auto& actor : self->m_actors) {
        actor->trial_ended("");
      }
    }
    catch(const std::exception& exc) {
      spdlog::error("Trial [{}] - Actors end failed [{}]", self->m_id, exc.what());
    }
    catch(...) {
      spdlog::error("Trial [{}] - Actors end failed", self->m_id);
    }
  });
  env_thr.detach();
}

void Trial::actor_acted(const std::string& actor_name, const cogmentAPI::Action& action) {
  std::shared_lock<std::shared_mutex> terminating_guard(m_terminating_lock);

  if (m_state < InternalState::pending) {
    spdlog::warn("Too early for trial [{}] to receive actions from [{}].", m_id, actor_name);
    return;
  }
  if (m_state >= InternalState::terminating) {
    spdlog::info("An action from [{}] arrived after end of trial [{}].  Action will be dropped.", actor_name, m_id);
    return;
  }

  const auto itor = m_actor_indexes.find(actor_name);
  if (itor == m_actor_indexes.end()) {
    spdlog::error("Unknown actor [{}] for action received in trial [{}].", actor_name, m_id);
    return;
  }
  const auto actor_index = itor->second;

  auto sample = get_last_sample();
  if (sample == nullptr) {
    spdlog::warn("No sample to accept action from [{}] in trial [{}]. Trial may be finished.", actor_name, m_id);
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

  SPDLOG_TRACE("Trial [{}]: Received action from actor [{}] for tick [{}].", m_id, actor_name, m_tick_id);
  *sample_action = action;

  bool all_actions_received = false;
  {
    const std::lock_guard<std::mutex> lg(m_actor_lock);
    m_gathered_actions_count++;
    all_actions_received = (m_gathered_actions_count == m_actors.size());
  }

  if (all_actions_received) {
    SPDLOG_TRACE("Trial [{}]: All actions received for tick [{}]", m_id, m_tick_id);

    const auto max_steps = m_params.max_steps();
    if (max_steps == 0 || m_tick_id < max_steps) {
      if (m_env_stream) {
        dispatch_env_messages();

        m_env_stream->Write(make_action_request());

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
        spdlog::error("Environment for trial [{}] not ready for first action set", m_id);
      }
    }
    else {
      spdlog::info("Trial [{}]: Terminating on configured maximum number of steps [{}]", m_id, max_steps);
      terminating_guard.unlock();
      terminate();
    }
  }
}

Client_actor* Trial::get_join_candidate(const cogmentAPI::TrialJoinRequest& req) {
  if (m_state != InternalState::pending) {
    throw MakeException("Trial in wrong state to be joined");
  }

  Client_actor* candidate = nullptr;

  switch (req.slot_selection_case()) {
  case cogmentAPI::TrialJoinRequest::kActorName: {
    auto actor_index = m_actor_indexes.at(req.actor_name());
    auto& actor = m_actors.at(actor_index);

    candidate = dynamic_cast<Client_actor*>(actor.get());
    if (candidate == nullptr) {
      throw MakeException("Actor [%s] is not a client actor", req.actor_name().c_str());
    }

    if (candidate->is_active()) {
      throw MakeException("Actor [%s] has already joined", req.actor_name().c_str());
    }
    break;
  }

  case cogmentAPI::TrialJoinRequest::kActorClass: {
    for (auto& actor : m_actors) {
      if (!actor->is_active() && actor->actor_class()->name == req.actor_class()) {
        auto available_actor = dynamic_cast<Client_actor*>(actor.get());
        if (available_actor != nullptr) {
          candidate = available_actor;
          break;
        }
      }
    }

    if (candidate == nullptr) {
      throw MakeException("Could not find matching actor of class [%s] to join", req.actor_class().c_str());
    }
    break;
  }

  case cogmentAPI::TrialJoinRequest::SLOT_SELECTION_NOT_SET:
  default:
    throw MakeException<std::invalid_argument>("Must specify either actor_name or actor_class");
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
    SPDLOG_TRACE("Trial [{}]: New state [{}] at tick [{}]", m_id, get_trial_state_string(new_state), m_tick_id);
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

bool Trial::is_stale() const {
  if (m_state == InternalState::ended) {
    return false;
  }

  const auto inactivity_period = std::chrono::steady_clock::now() - m_last_activity;
  const auto& max_inactivity = m_params.max_inactivity();

  const bool stale = (inactivity_period > std::chrono::seconds(max_inactivity));
  return (max_inactivity > 0 && stale);
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

  cogmentAPI::EnvMessageRequest req;
  for (auto& message : m_env_message_accumulator) {
    auto msg = req.add_messages();
    *msg = std::move(message);
  }

  if (!m_env_message_accumulator.empty()) {
    grpc::ClientContext context;
    context.AddMetadata("trial-id", m_id);
    cogmentAPI::EnvMessageReply response;
    auto status = m_env_entry->get_stub().OnMessage(&context, req, &response);
    if (!status.ok()) {
      spdlog::error("Trial [{}] - Failed to send a message to environment", m_id);
    }
  }

  m_env_message_accumulator.clear();
}

}  // namespace cogment
