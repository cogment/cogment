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

#include "cogment/trial.h"
#include "cogment/orchestrator.h"

#include "cogment/agent_actor.h"
#include "cogment/client_actor.h"
#include "cogment/reward.h"

#include "spdlog/spdlog.h"

#include <limits>

#if COGMENT_DEBUG
  #define TRIAL_DEBUG_LOG(...) spdlog::debug(__VA_ARGS__)
#else
  #define TRIAL_DEBUG_LOG(...)
#endif
namespace cogment {

constexpr int64_t AUTO_TICK_ID = -1;
constexpr uint64_t MAX_TICK_ID = static_cast<uint64_t>(std::numeric_limits<int64_t>::max());
const std::string ENVIRONMENT_ACTOR_NAME("env");

const char* get_trial_state_string(Trial_state s) {
  switch (s) {
  case Trial_state::initializing:
    return "initializing";
  case Trial_state::pending:
    return "pending";
  case Trial_state::running:
    return "running";
  case Trial_state::terminating:
    return "terminating";
  case Trial_state::ended:
    return "ended";
  }

  throw std::out_of_range("unknown trial state");
}

cogment::TrialState get_trial_api_state(Trial_state s) {
  switch (s) {
  case Trial_state::initializing:
    return cogment::INITIALIZING;
  case Trial_state::pending:
    return cogment::PENDING;
  case Trial_state::running:
    return cogment::RUNNING;
  case Trial_state::terminating:
    return cogment::TERMINATING;
  case Trial_state::ended:
    return cogment::ENDED;
  }

  throw std::out_of_range("unknown trial state");
}

uuids::uuid_system_generator Trial::id_generator_;

Trial::Trial(Orchestrator* orch, std::string user_id)
    : orchestrator_(orch), id_(id_generator_()), user_id_(std::move(user_id)), tick_id_(0) {
  set_state(Trial_state::initializing);
  refresh_activity();
}

Trial::~Trial() { spdlog::debug("Tearing down trial {}", to_string(id_)); }

const std::unique_ptr<Actor>& Trial::actor(const std::string& name) const {
  auto actor_index = actor_indexes_.at(name);
  return actors_[actor_index];
}

void Trial::new_tick(ObservationSet&& new_obs) {
  if (new_obs.tick_id() == AUTO_TICK_ID) {
    tick_id_++;
    if (tick_id_ > MAX_TICK_ID) {
      throw std::runtime_error("Tick id has reached the limit");
    }
  }
  else if (new_obs.tick_id() < 0) {
    throw std::runtime_error("Invalid negative tick id from environment");
  }
  else {
    const uint64_t new_tick_id = static_cast<uint64_t>(new_obs.tick_id());

    if (new_tick_id <= tick_id_ && tick_id_ > 0) {
      throw std::runtime_error("Environment repeated a tick id");
    }

    // This condition could be revisited
    if (new_tick_id > tick_id_ + 1) {
      throw std::runtime_error("Environment skipped tick id");
    }

    if (new_tick_id > MAX_TICK_ID) {
      throw std::runtime_error("Tick id from environment is too large");
    }

    tick_id_ = new_tick_id;
  }

  observations_ = std::move(new_obs);
}

// TODO: Add protection so we don't "start" more than once.  We could also add protection in other
//       functions (performance permitting) to prevent them being called before the "start".
void Trial::start(cogment::TrialParams params) {
  params_ = std::move(params);
  TRIAL_DEBUG_LOG("Configuring trial {} with parameters: {}", to_string(id_), params_.DebugString());

  grpc_metadata trial_header;
  trial_header.key = grpc_slice_from_static_string("trial-id");
  trial_header.value = grpc_slice_from_copied_string(to_string(id_).c_str());
  headers_.push_back(trial_header);
  call_options_.headers = &headers_;

  env_stub_ = orchestrator_->env_pool()->get_stub(params_.environment().endpoint());

  ::cogment::EnvStartRequest env_start_req;
  env_start_req.set_tick_id(tick_id_);
  env_start_req.set_impl_name(params_.environment().implementation());
  if (params_.environment().has_config()) {
    *env_start_req.mutable_config() = params_.environment().config();
  }

  for (const auto& actor_info : params_.actors()) {
    auto url = actor_info.endpoint();
    const auto& actor_class = orchestrator_->get_trial_spec().get_actor_class(actor_info.actor_class());

    auto actor_in_trial = env_start_req.add_actors_in_trial();
    actor_in_trial->set_actor_class(actor_class.name);
    actor_in_trial->set_name(actor_info.name());

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

  actions_.resize(actors_.size());

  set_state(Trial_state::pending);

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

  join(env_ready, concat(actors_ready.begin(), actors_ready.end()))
      .then([this](auto env_rep) {
        auto obs_set = env_rep.mutable_observation_set();
        if (obs_set->tick_id() == AUTO_TICK_ID) {
          obs_set->set_tick_id(0);  // First observation is not for new/next tick but for first tick
        }
        new_tick(std::move(*obs_set));

        set_state(Trial_state::running);

        run_environment();
        // Send the initial state
        dispatch_observations(false);
      })
      .finally([](auto) {});

  spdlog::debug("Trial {} is configured", to_string(id_));
}

void Trial::reward_received(const cogment::Reward& reward, const std::string& sender) {
  // TODO: timed rewards (i.e. with a specific tick_id != AUTO_TICK_ID and != current tick_id_)

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
      actors_[actor_index_itor->second]->add_immediate_reward_src(src, sender);
    }
  }
  else {
    spdlog::error("Unknown actor name as reward destination [{}]", reward.receiver_name());
  }
}

void Trial::message_received(const cogment::Message& message, const std::string& sender) {
  // TODO: timed messages (i.e. with a specific tick_id != AUTO_TICK_ID and != current tick_id_)

  if (message.tick_id() != AUTO_TICK_ID && message.tick_id() != static_cast<int64_t>(tick_id_)) {
    spdlog::error("Invalid message tick from [{}]: [{}] (current tick id: [{}])", sender, message.tick_id(), tick_id_);
    return;
  }

  // Message is not dispatched as we receive it. It is accumulated, and sent once
  // per update.
  auto actor_index_itor = actor_indexes_.find(message.receiver_name());
  if (actor_index_itor != actor_indexes_.end()) {
    actors_[actor_index_itor->second]->add_immediate_message(message, sender);
  }
  else {
    spdlog::error("Unknown actor name as message destination [{}]", message.receiver_name());
  }
}

void Trial::dispatch_observations(bool end_of_trial) {
  if (state_ == Trial_state::ended) {
    return;
  }

  std::uint32_t actor_index = 0;
  for (const auto& actor : actors_) {
    auto obs_index = observations_.actors_map(actor_index);

    cogment::Observation obs;
    obs.set_tick_id(tick_id_);
    obs.set_timestamp(observations_.timestamp());
    *obs.mutable_data() = observations_.observations(obs_index);
    actor->dispatch_observation(std::move(obs), end_of_trial);

    auto sources = actor->get_and_flush_immediate_reward_src();
    if (!sources.empty()) {
      auto reward = build_reward(sources);
      reward.set_tick_id(tick_id_);
      reward.set_receiver_name(actor->actor_name());
      actor->dispatch_reward(std::move(reward));
    }

    auto messages = actor->get_and_flush_immediate_message();
    if (!messages.empty()) {
      for (auto& message : messages) {
        message.set_tick_id(tick_id_);
        actor->dispatch_message(std::move(message));
      }
    }

    ++actor_index;
  }

  // This is not ideal and should get cleaned up as we (finally) revise the end-of-trial flow.
  if (end_of_trial && state_ != Trial_state::ended) {
    // Stop sending actions to the environment
    outgoing_actions_->complete();
    actors_.clear();
    actor_indexes_.clear();
    outgoing_actions_ = std::nullopt;
    set_state(Trial_state::ended);
  }
}

void Trial::run_environment() {
  auto self = shared_from_this();

  // Bootstrap the log with the initial observation
  step_data_.emplace_back();

  auto& sample = step_data_.back();
  sample.mutable_observations()->CopyFrom(observations_);

  // Launch the main update stream
  auto streams = (*env_stub_)->OnAction(call_options_);

  outgoing_actions_ = std::move(std::get<0>(streams));
  auto incoming_updates = std::move(std::get<1>(streams));

  // Whenever we get an update, advance the datalog table.
  incoming_updates
      .for_each([this](auto update) {
        TRIAL_DEBUG_LOG("update: {}", update.DebugString());

        new_tick(std::move(*update.mutable_observation_set()));

        step_data_.emplace_back();
        auto& sample = step_data_.back();
        sample.mutable_observations()->CopyFrom(observations_);

        for (const auto& rew : update.rewards()) {
          reward_received(rew, ENVIRONMENT_ACTOR_NAME);
        }

        for (const auto& message : update.messages()) {
          message_received(message, ENVIRONMENT_ACTOR_NAME);
        }

        if (update.final_update() && state_ != Trial_state::ended) {
          set_state(Trial_state::terminating);
        }
        dispatch_observations(update.final_update());
      })
      .finally([self](auto) {
        // We are holding on to self until the rpc is over.
        // This is important because we we still need to finish
        // this call while the trial is "deleted".
      });
}

cogment::EnvActionRequest Trial::make_action_request() {
  cogment::EnvActionRequest req;
  auto action_set = req.mutable_action_set();
  action_set->set_tick_id(tick_id_);

  for (auto& act : actions_) {
    // TODO: Synchronize properly with actor_acted()
    if (act) {
      action_set->add_actions(act->content());
    }
    else {
      action_set->add_actions("");
    }

    act = std::nullopt;
    gathered_actions_count_--;
  }

  return req;
}

void Trial::terminate() {
  auto self = shared_from_this();

  if (state_ != Trial_state::ended) {
    set_state(Trial_state::terminating);
  }

  // Send the actions we have so far (partial set)
  // TODO: Should we instead send all empty actions?
  auto req = make_action_request();

  // TODO: Add fail safe in case communication is broken or environment is down
  (*env_stub_)
      ->OnEnd(req, call_options_)
      .then([this](auto rep) {
        new_tick(std::move(*rep.mutable_observation_set()));

        step_data_.emplace_back();
        auto& sample = step_data_.back();
        sample.mutable_observations()->CopyFrom(observations_);

        dispatch_observations(true);
      })
      .finally([self](auto) {
        // We are holding on to self until the rpc is over.
        // This is important because we still need to finish
        // this call while the trial is "deleted".
      });
}

void Trial::actor_acted(const std::string& actor_name, const cogment::Action& action) {
  if (!outgoing_actions_) {
    return;
  }

  if (action.tick_id() != AUTO_TICK_ID && action.tick_id() != static_cast<int64_t>(tick_id_)) {
    spdlog::error("Invalid action tick from [{}]: [{}] (current tick id: [{}])", actor_name, action.tick_id(),
                  tick_id_);
    return;
  }

  // TODO: Do we want to manage the exception if the name is not found?
  auto actor_index = actor_indexes_.at(actor_name);

  if (actions_[actor_index] == std::nullopt) {
    ++gathered_actions_count_;
  }
  actions_[actor_index] = action;
  actions_[actor_index]->set_tick_id(static_cast<int64_t>(tick_id_));

  if (gathered_actions_count_ == actions_.size()) {
    auto req = make_action_request();
    outgoing_actions_->push(std::move(req));
  }
}

Client_actor* Trial::get_join_candidate(const TrialJoinRequest& req) {
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
    throw std::invalid_argument("Must specify either actor_name or actor_class");
  }

  if (result != nullptr && dynamic_cast<Client_actor*>(result) == nullptr) {
    throw std::invalid_argument("Actor name or class is not a client actor");
  }

  return static_cast<Client_actor*>(result);
}

void Trial::set_state(Trial_state state) {
  const std::lock_guard<std::mutex> lock(state_lock_);
  state_ = state;
  orchestrator_->notify_watchers(*this);
}

void Trial::refresh_activity() { last_activity_ = std::chrono::steady_clock::now(); }

bool Trial::is_stale() const {
  bool stale = std::chrono::steady_clock::now() - last_activity_ > std::chrono::seconds(params_.max_inactivity());
  return params_.max_inactivity() > 0 && stale;
}
}  // namespace cogment
