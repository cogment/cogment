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

#if COGMENT_DEBUG
  #define TRIAL_DEBUG_LOG(...) spdlog::debug(__VA_ARGS__)
#else
  #define TRIAL_DEBUG_LOG(...)
#endif
namespace cogment {

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

cogment::TrialState get_trial_state_proto(Trial_state s) {
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
    : orchestrator_(orch), id_(id_generator_()), user_id_(std::move(user_id)), state_(Trial_state::initializing) {
  refresh_activity();
}

Trial::~Trial() { spdlog::debug("Tearing down trial {}", to_string(id_)); }

std::lock_guard<std::mutex> Trial::lock() { return std::lock_guard(lock_); }

const uuids::uuid& Trial::id() const { return id_; }

const std::string& Trial::user_id() const { return user_id_; }

const std::vector<std::unique_ptr<Actor>>& Trial::actors() const { return actors_; }

const std::unique_ptr<Actor>& Trial::actor(const std::string& name) const {
  auto actor_index = actor_indexes_.at(name);
  return actors_[actor_index];
}

void Trial::configure(cogment::TrialParams params) {
  params_ = std::move(params);

  grpc_metadata trial_header;
  trial_header.key = grpc_slice_from_static_string("trial-id");
  trial_header.value = grpc_slice_from_copied_string(to_string(id_).c_str());
  headers_.push_back(trial_header);
  call_options_.headers = &headers_;

  env_stub_ = orchestrator_->env_pool()->get_stub(params_.environment().endpoint());

  ::cogment::EnvStartRequest env_start_req;
  if (params_.environment().has_config()) {
    *env_start_req.mutable_config() = params_.environment().config();
  }

  // TODO: Figure out where the impl_name should come from and use that instead of hardcoded "default"
  env_start_req.set_impl_name("default");

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

  state_ = Trial_state::pending;

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
        latest_observations_ = std::move(*env_rep.mutable_observation_set());
        state_ = Trial_state::running;

        run_environment();
        // Send the initial state
        dispatch_observations(false);
      })
      .finally([](auto) {});

  spdlog::debug("Trial {} is configured", to_string(id_));
}

void Trial::feedback_received(const cogment::Feedback& feedback) {
  // TODO: deferred feedback.

  // Feedback is not dispatched as we receive it. It is accumulated, and sent once
  // per update.
  auto actor_index_itor = actor_indexes_.find(feedback.actor_name());
  if (actor_index_itor != actor_indexes_.end()) {
    // This is immediate feedback, accumulate it.
    if (feedback.tick_id() == -1) {
      actors_[actor_index_itor->second]->add_immediate_feedback(feedback);
    }
  }
}

void Trial::message_received(const cogment::Message& message, const std::string& source) {
  // TODO: deferred message.

  // Message is not dispatched as we receive it. It is accumulated, and sent once
  // per update.
  auto actor_index_itor = actor_indexes_.find(message.receiver_name());
  if (actor_index_itor != actor_indexes_.end()) {
    // This is immediate message, accumulate it.
    if (message.tick_id() == -1) {
      actors_[actor_index_itor->second]->add_immediate_message(message, source);
    }
  }
}

void Trial::dispatch_observations(bool end_of_trial) {
  if (state_ == Trial_state::ended) {
    return;
  }

  std::uint32_t actor_index = 0;
  for (const auto& actor : actors_) {
    auto obs_index = latest_observations_.actors_map(actor_index);

    cogment::Observation obs;
    obs.set_tick_id(latest_observations_.tick_id());
    obs.set_timestamp(latest_observations_.timestamp());
    *obs.mutable_data() = latest_observations_.observations(obs_index);
    actor->dispatch_observation(obs, end_of_trial);

    auto feedbacks = actor->get_and_flush_immediate_feedback();
    if (!feedbacks.empty()) {
      auto reward = build_reward(feedbacks);
      actor->dispatch_reward(latest_observations_.tick_id(), reward);
    }

    auto messages = actor->get_and_flush_immediate_message();
    if (!messages.empty()) {
      for (const auto& message : messages) {
        actor->dispatch_message(latest_observations_.tick_id(), message);
      }
    }

    ++actor_index;
  }

  if (end_of_trial) {
    // Stop sending actions to the environment
    outgoing_actions_->complete();
    actors_.clear();
    actor_indexes_.clear();
    outgoing_actions_ = std::nullopt;
    state_ = Trial_state::ended;
  }
}

void Trial::run_environment() {
  auto self = shared_from_this();

  // Bootstrap the log with the initial observation
  step_data_.emplace_back();

  auto& sample = step_data_.back();
  sample.mutable_observations()->CopyFrom(latest_observations_);

  // Launch the main update stream
  auto streams = (*env_stub_)->OnAction(call_options_);

  outgoing_actions_ = std::move(std::get<0>(streams));
  auto incoming_updates = std::move(std::get<1>(streams));

  // Whenever we get an update, advance the datalog table.
  incoming_updates
      .for_each([this](auto update) {
        TRIAL_DEBUG_LOG("update: {}", update.DebugString());

        latest_observations_ = std::move(*update.mutable_observation_set());

        step_data_.emplace_back();
        auto& sample = step_data_.back();
        sample.mutable_observations()->CopyFrom(latest_observations_);

        for (const auto& fb : update.feedbacks()) {
          feedback_received(fb);
        }

        for (const auto& message : update.messages()) {
          message_received(message, ENVIRONMENT_ACTOR_NAME);
        }

        if (update.final_update()) {
          state_ = Trial_state::terminating;
        }
        dispatch_observations(update.final_update());
      })
      .finally([self](auto) {
        // We are holding on to self until the rpc is over.
        // This is important because we we still need to finish
        // this call while the trial is "deleted".
      });
}

void Trial::terminate() {
  auto self = shared_from_this();
  state_ = Trial_state::terminating;

  cogment::EnvActionRequest req;

  // Send the actions we have so far (partial set)
  // TODO: Should we instead send all empty actions?
  for (auto& act : actions_) {
    // TODO: Synchronize properly with actor_acted()
    if (act) {
      req.mutable_action_set()->add_actions(act->content());
    }
    else {
      req.mutable_action_set()->add_actions("");
    }

    act = std::nullopt;
    gathered_actions_count_--;
  }

  // TODO: Add fail safe in case communication is broken or environment is down
  (*env_stub_)
      ->OnEnd(req, call_options_)
      .then([this](auto rep) {
        latest_observations_ = std::move(*rep.mutable_observation_set());

        step_data_.emplace_back();
        auto& sample = step_data_.back();
        sample.mutable_observations()->CopyFrom(latest_observations_);

        dispatch_observations(true);
      })
      .finally([self](auto) {
        // We are holding on to self until the rpc is over.
        // This is important because we we still need to finish
        // this call while the trial is "deleted".
      });
}

void Trial::actor_acted(const std::string& actor_name, const cogment::Action& action) {
  if (!outgoing_actions_) {
    return;
  }

  // TODO: Do we want to manage the exception if the name is not found?
  auto actor_index = actor_indexes_.at(actor_name);

  if (actions_[actor_index] == std::nullopt) {
    ++gathered_actions_count_;
  }

  actions_[actor_index] = action;

  if (gathered_actions_count_ == actions_.size()) {
    gathered_actions_count_ = 0;

    ::cogment::EnvActionRequest req;
    for (auto& act : actions_) {
      if (act) {
        req.mutable_action_set()->add_actions(act->content());
      }
      else {
        req.mutable_action_set()->add_actions("");
      }

      act = std::nullopt;
    }

    outgoing_actions_->push(std::move(req));
  }
}

const cogment::TrialParams& Trial::params() const { return params_; }

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

Trial_state Trial::state() const { return state_; }

void Trial::refresh_activity() { last_activity_ = std::chrono::steady_clock::now(); }

bool Trial::is_stale() const {
  bool stale = std::chrono::steady_clock::now() - last_activity_ > std::chrono::seconds(params_.max_inactivity());
  return params_.max_inactivity() > 0 && stale;
}
}  // namespace cogment
