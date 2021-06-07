// Copyright 2021 Artificial Intelligence Redefined <dev+cogment@ai-r.com>
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

uuids::uuid_system_generator Trial::id_generator_;

Trial::Trial(Orchestrator* orch, std::string user_id)
    : orchestrator_(orch),
      id_(id_generator_()),
      user_id_(std::move(user_id)),
      state_(InternalState::unknown),
      tick_id_(0),
      start_timestamp_(Timestamp()),
      end_timestamp_(0) {
  SPDLOG_TRACE("New Trial [{}]", to_string(id_));

  set_state(InternalState::initializing);
  refresh_activity();

  datalog_interface_ = orch->start_log(this);
}

Trial::~Trial() { SPDLOG_TRACE("Tearing down trial [{}]", to_string(id_)); }

const std::unique_ptr<Actor>& Trial::actor(const std::string& name) const {
  auto actor_index = actor_indexes_.at(name);
  return actors_[actor_index];
}

void Trial::advance_tick() {
  SPDLOG_TRACE("Tick [{}] is done for Trial [{}]", tick_id_, to_string(id_));

  tick_id_++;
  if (tick_id_ > MAX_TICK_ID) {
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

    if (new_tick_id < tick_id_) {
      throw MakeException("Environment repeated a tick id: [%llu]", new_tick_id);
    }

    if (new_tick_id > MAX_TICK_ID) {
      throw MakeException("Tick id from environment is too large");
    }

    if (new_tick_id > tick_id_) {
      throw MakeException("Environment skipped tick id: [%llu] vs [%llu]", new_tick_id, tick_id_);
    }
  }

  auto sample = get_last_sample();
  *(sample->mutable_observations()) = std::move(obs);
}

cogment::DatalogSample* Trial::get_last_sample() {
  const std::lock_guard<std::mutex> lg(sample_lock_);
  if (!step_data_.empty()) {
    return &(step_data_.back());
  }
  else {
    return nullptr;
  }
}

cogment::DatalogSample& Trial::make_new_sample() {
  const std::lock_guard<std::mutex> lg(sample_lock_);
  step_data_.emplace_back();
  auto& sample = step_data_.back();

  auto sample_actions = sample.mutable_actions();
  sample_actions->Reserve(actors_.size());
  for (size_t index = 0; index < actors_.size(); index++) {
    auto no_action = sample_actions->Add();
    no_action->set_tick_id(NO_DATA_TICK_ID);
  }
  gathered_actions_count_ = 0;

  auto trial_data = sample.mutable_trial_data();
  trial_data->set_tick_id(tick_id_);
  trial_data->set_timestamp(Timestamp());
  trial_data->set_state(get_trial_api_state(state_));

  return sample;
}

void Trial::flush_samples() {
  const std::lock_guard<std::mutex> lg(sample_lock_);

  for (auto& sample : step_data_) {
    datalog_interface_->add_sample(std::move(sample));
  }
  step_data_.clear();
}

void Trial::prepare_actors() {
  for (const auto& actor_info : params_.actors()) {
    auto url = actor_info.endpoint();
    const auto& actor_class = orchestrator_->get_trial_spec().get_actor_class(actor_info.actor_class());

    if (url == "client") {
      std::optional<std::string> config;
      if (actor_info.has_config()) {
        config = actor_info.config().content();
      }
      auto client_actor = std::make_unique<Client_actor>(this, actor_info.name(), &actor_class, config);
      actors_.push_back(std::move(client_actor));
    }
    else {
      std::optional<std::string> config;
      if (actor_info.has_config()) {
        config = actor_info.config().content();
      }
      auto stub_entry = orchestrator_->agent_pool()->get_stub(url);
      auto agent_actor = std::make_unique<Agent>(this, actor_info.name(), &actor_class, actor_info.implementation(),
                                                 stub_entry, config);
      actors_.push_back(std::move(agent_actor));
    }

    actor_indexes_.emplace(actor_info.name(), actors_.size() - 1);
  }
}

cogment::EnvStartRequest Trial::prepare_environment() {
  env_stub_ = orchestrator_->env_pool()->get_stub(params_.environment().endpoint());

  cogment::EnvStartRequest env_start_req;
  env_start_req.set_tick_id(tick_id_);
  env_start_req.set_impl_name(params_.environment().implementation());
  if (params_.environment().has_config()) {
    *env_start_req.mutable_config() = params_.environment().config();
  }

  for (const auto& actor_info : params_.actors()) {
    const auto& actor_class = orchestrator_->get_trial_spec().get_actor_class(actor_info.actor_class());
    auto actor_in_trial = env_start_req.add_actors_in_trial();
    actor_in_trial->set_actor_class(actor_class.name);
    actor_in_trial->set_name(actor_info.name());
  }

  return env_start_req;
}

void Trial::start(cogment::TrialParams params) {
  SPDLOG_TRACE("Starting Trial [{}]", to_string(id_));

  if (state_ != InternalState::initializing) {
    throw MakeException("Trial [%s] is not in proper state to start: [%s]", to_string(id_).c_str(),
                        get_trial_state_string(state_));
  }

  params_ = std::move(params);
  SPDLOG_DEBUG("Configuring trial {} with parameters: {}", to_string(id_), params_.DebugString());

  grpc_metadata trial_header;
  trial_header.key = grpc_slice_from_static_string("trial-id");
  trial_header.value = grpc_slice_from_copied_string(to_string(id_).c_str());
  headers_.push_back(trial_header);
  call_options_.headers = &headers_;

  auto env_start_req = prepare_environment();

  prepare_actors();

  make_new_sample();  // First sample

  set_state(InternalState::pending);

  std::vector<aom::Future<void>> actors_ready;
  for (const auto& actor : actors_) {
    actors_ready.push_back(actor->init());
  }

  auto env_ready = (*env_stub_)->OnStart(std::move(env_start_req), call_options_).then_expect([](auto rep) {
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
  SPDLOG_TRACE("Trial [{}]: waiting for actors and environment...", to_string(id_));
  join(env_ready, concat(actors_ready.begin(), actors_ready.end()))
      .then([this](auto env_rep) {
        new_obs(std::move(*env_rep.mutable_observation_set()));

        set_state(InternalState::running);

        run_environment();

        // Send the initial state
        dispatch_observations();
      })
      .finally([self](auto) { SPDLOG_TRACE("Trial [{}]: All components started.", to_string(self->id_)); });

  spdlog::debug("Trial {} is configured", to_string(id_));
}

void Trial::reward_received(const cogment::Reward& reward, const std::string& sender) {
  if (state_ < InternalState::pending) {
    spdlog::warn("Too early for trial [{}] to receive rewards.", to_string(id_));
    return;
  }

  auto sample = get_last_sample();
  if (sample != nullptr) {
    const std::lock_guard<std::mutex> lg(reward_lock_);
    auto new_rew = sample->add_rewards();
    *new_rew = reward;
  }
  else {
    return;
  }

  // TODO: Decide what to do with timed rewards (send anything present and past, hold future?)
  if (reward.tick_id() != AUTO_TICK_ID && reward.tick_id() != static_cast<int64_t>(tick_id_)) {
    spdlog::error("Invalid reward tick from [{}]: [{}] (current tick id: [{}])", sender, reward.tick_id(), tick_id_);
    return;
  }

  // Rewards are not dispatched as we receive them. They are accumulated, and sent once
  // per update.
  auto actor_index_itor = actor_indexes_.find(reward.receiver_name());
  if (actor_index_itor != actor_indexes_.end()) {
    // Normally we should have only one source when receiving
    for (const auto& src : reward.sources()) {
      actors_[actor_index_itor->second]->add_immediate_reward_src(src, sender, tick_id_);
    }
  }
  else {
    spdlog::error("Unknown actor name as reward destination [{}]", reward.receiver_name());
  }
}

void Trial::message_received(const cogment::Message& message, const std::string& sender) {
  if (state_ < InternalState::pending) {
    spdlog::warn("Too early for trial [{}] to receive messages.", to_string(id_));
    return;
  }

  auto sample = get_last_sample();
  if (sample != nullptr) {
    const std::lock_guard<std::mutex> lg(sample_message_lock_);
    auto new_msg = sample->add_messages();
    *new_msg = message;
  }
  else {
    return;
  }

  // TODO: Decide what to do with timed messages (send anything present and past, hold future?)
  if (message.tick_id() != AUTO_TICK_ID && message.tick_id() != static_cast<int64_t>(tick_id_)) {
    spdlog::error("Invalid message tick from [{}]: [{}] (current tick id: [{}])", sender, message.tick_id(), tick_id_);
    return;
  }

  // Message is not dispatched as we receive it. It is accumulated, and sent once
  // per update.
  auto actor_index_itor = actor_indexes_.find(message.receiver_name());
  if (actor_index_itor != actor_indexes_.end()) {
    actors_[actor_index_itor->second]->add_immediate_message(message, sender, tick_id_);
  }
  else if (message.receiver_name() == ENVIRONMENT_ACTOR_NAME) {
    const std::lock_guard<std::mutex> lg(env_message_lock_);
    env_message_accumulator_.emplace_back(message);
    env_message_accumulator_.back().set_tick_id(tick_id_);
    env_message_accumulator_.back().set_sender_name(sender);
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
  if (state_ == InternalState::ended) {
    return;
  }
  const bool ending = (state_ == InternalState::terminating);

  auto sample = get_last_sample();
  if (sample == nullptr) {
    return;
  }
  const auto& observations = sample->observations();

  std::uint32_t actor_index = 0;
  for (const auto& actor : actors_) {
    auto obs_index = observations.actors_map(actor_index);
    cogment::Observation obs;
    obs.set_tick_id(tick_id_);
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

  const std::lock_guard<std::mutex> lg(sample_lock_);

  // Send overflow to log
  if (step_data_.size() >= LOG_TRIGGER_SIZE) {
    while (step_data_.size() >= NB_BUFFERED_SAMPLES) {
      datalog_interface_->add_sample(std::move(step_data_.front()));
      step_data_.pop_front();
    }
  }
}

void Trial::run_environment() {
  auto self = shared_from_this();

  // Launch the main update stream
  auto streams = (*env_stub_)->OnAction(call_options_);

  outgoing_actions_ = std::move(std::get<0>(streams));
  auto incoming_updates = std::move(std::get<1>(streams));

  incoming_updates
      .for_each([this](auto update) {
        SPDLOG_TRACE("Received new observation set for Trial [{}]", to_string(id_));

        if (state_ == InternalState::ended) {
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

  action_set->set_tick_id(tick_id_);

  const std::lock_guard<std::mutex> lg(sample_lock_);
  auto& sample = step_data_.back();
  for (auto& act : sample.actions()) {
    if (act.tick_id() == AUTO_TICK_ID || act.tick_id() == static_cast<int64_t>(tick_id_)) {
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
    const std::lock_guard<std::shared_mutex> lg(terminating_lock_);

    if (state_ >= InternalState::terminating) {
      return;
    }
    if (state_ != InternalState::running) {
      spdlog::error("Trial [{}] cannot terminate in current state: [{}]", to_string(id_).c_str(),
                    get_trial_state_string(state_));
      return;
    }

    set_state(InternalState::terminating);
  }

  dispatch_env_messages();

  auto self = shared_from_this();

  // Send the actions we have so far (may be a partial set)
  auto req = make_action_request();

  // Fail safe timed call because this function is also called to end abnormal/stale trials.
  auto timed_options = call_options_;
  timed_options.deadline.tv_sec = 60;
  timed_options.deadline.tv_nsec = 0;
  timed_options.deadline.clock_type = GPR_TIMESPAN;

  (*env_stub_)
      ->OnEnd(req, timed_options)
      .then([this](auto rep) {
        next_step(std::move(rep));
        dispatch_observations();
      })
      .finally([self](auto) {
        // We are holding on to self to be safe.

        if (self->state_ != InternalState::ended) {
          // One possible reason is if the environment took longer than the deadline to respond
          spdlog::error("Trial [{}] did not end normally.", to_string(self->id_));

          // TODO: Send "ended trial" to all components?  So they don't get stuck waiting. Or cut comms?

          self->state_ = InternalState::ended;  // To enable garbage collection on this trial
        }
      });
}

void Trial::actor_acted(const std::string& actor_name, const cogment::Action& action) {
  std::shared_lock<std::shared_mutex> terminating_guard(terminating_lock_);

  if (state_ < InternalState::pending) {
    spdlog::warn("Too early for trial [{}] to receive actions from [{}].", to_string(id_), actor_name);
    return;
  }
  if (state_ >= InternalState::terminating) {
    spdlog::info("An action from [{}] arrived after end of trial [{}].  Action will be dropped.", actor_name,
                 to_string(id_));
    return;
  }

  const auto itor = actor_indexes_.find(actor_name);
  if (itor == actor_indexes_.end()) {
    spdlog::error("Unknown actor [{}] for action received in trial [{}].", actor_name, to_string(id_));
    return;
  }
  const auto actor_index = itor->second;

  auto sample = get_last_sample();
  if (sample == nullptr) {
    return;
  }
  auto sample_action = sample->mutable_actions(actor_index);
  if (sample_action->tick_id() != NO_DATA_TICK_ID) {
    spdlog::warn("Multiple actions from [{}] for same step: only the first one will be used.", actor_name);
    return;
  }

  // TODO: Determine what we want to do in case of actions in the past or future
  if (action.tick_id() != AUTO_TICK_ID && action.tick_id() != static_cast<int64_t>(tick_id_)) {
    spdlog::warn("Invalid action tick from [{}]: [{}] vs [{}].  Default action will be used.", actor_name,
                 action.tick_id(), tick_id_);
  }
  *sample_action = action;

  bool all_actions_received = false;
  {
    const std::lock_guard<std::mutex> lg(actor_lock_);
    gathered_actions_count_++;
    all_actions_received = (gathered_actions_count_ == actors_.size());
  }

  if (all_actions_received) {
    SPDLOG_TRACE("All actions received for Trial [{}]", to_string(id_));

    const auto max_steps = params_.max_steps();
    if (max_steps == 0 || tick_id_ < max_steps) {
      if (outgoing_actions_) {
        dispatch_env_messages();
        outgoing_actions_->push(make_action_request());
      }
      else {
        spdlog::error("Environment for trial [{}] not ready for first action set", to_string(id_));
      }
    }
    else {
      spdlog::info("Trial [{}] reached its maximum number of steps: [{}]", to_string(id_), max_steps);
      terminating_guard.unlock();
      terminate();
    }
  }
}

Client_actor* Trial::get_join_candidate(const TrialJoinRequest& req) {
  if (state_ != InternalState::pending) {
    return nullptr;
  }

  Actor* result = nullptr;

  switch (req.slot_selection_case()) {
  case TrialJoinRequest::kActorName: {
    auto actor_index = actor_indexes_.at(req.actor_name());
    auto& actor = actors_.at(actor_index);
    if (actor->is_active()) {
      result = actor.get();
    }
  } break;

  case TrialJoinRequest::kActorClass:
    for (auto& actor : actors_) {
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
  const std::lock_guard<std::mutex> lg(state_lock_);

  bool invalid_transition = false;
  switch (state_) {
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
      spdlog::debug("Trial [{}] already ended: cannot end again", to_string(id_));
    }
    break;
  }

  if (invalid_transition) {
    throw MakeException("Cannot switch trial state from [%s] to [%s]", get_trial_state_string(state_),
                        get_trial_state_string(new_state));
  }

  if (state_ != new_state) {
    state_ = new_state;

    if (new_state == InternalState::ended) {
      end_timestamp_ = Timestamp();
    }

    // TODO: Find a better way so we don't have to be locked when calling out
    //       Right now it is necessary to make sure we don't miss state and they are seen in-order
    orchestrator_->notify_watchers(*this);

    if (new_state == InternalState::ended) {
      flush_samples();
    }
  }
}

void Trial::refresh_activity() { last_activity_ = std::chrono::steady_clock::now(); }

bool Trial::is_stale() const {
  const auto inactivity_period = std::chrono::steady_clock::now() - last_activity_;
  const auto& max_inactivity = params_.max_inactivity();

  const bool stale = (inactivity_period > std::chrono::seconds(max_inactivity));
  return (max_inactivity > 0 && stale);
}

void Trial::set_info(cogment::TrialInfo* info, bool with_observations, bool with_actors) {
  if (info == nullptr) {
    spdlog::error("Trial [{}] request for info with no storage", to_string(id_));
    return;
  }

  uint64_t end;
  if (end_timestamp_ == 0) {
    end = Timestamp();
  }
  else {
    end = end_timestamp_;
  }
  info->set_trial_duration(end - start_timestamp_);
  info->set_trial_id(to_string(id_));

  // The state and tick may not be synchronized here, but it is better
  // to have the latest state (as opposed to the state of the sample).
  info->set_state(get_trial_api_state(state_));

  auto sample = get_last_sample();
  if (sample == nullptr) {
    info->set_tick_id(tick_id_);
    return;
  }

  // We want to make sure the tick_id and observation are from the same tick
  const uint64_t tick = sample->trial_data().tick_id();
  info->set_tick_id(tick);
  if (with_observations && sample->has_observations()) {
    info->mutable_latest_observation()->CopyFrom(sample->observations());
  }

  if (with_actors && state_ >= InternalState::pending) {
    for (auto& actor : actors_) {
      auto trial_actor = info->add_actors_in_trial();
      trial_actor->set_actor_class(actor->actor_class()->name);
      trial_actor->set_name(actor->actor_name());
    }
  }
}

void Trial::dispatch_env_messages() {
  const std::lock_guard<std::mutex> lg(env_message_lock_);

  cogment::EnvMessageRequest req;
  for (auto& message : env_message_accumulator_) {
    auto msg = req.add_messages();
    *msg = std::move(message);
  }

  if (!env_message_accumulator_.empty()) {
    (*env_stub_)->OnMessage(req, call_options_).finally([](auto) {});
  }

  env_message_accumulator_.clear();
}

}  // namespace cogment
