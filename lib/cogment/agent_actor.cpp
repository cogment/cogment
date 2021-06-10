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

#include "cogment/agent_actor.h"
#include "cogment/trial.h"

#include "spdlog/spdlog.h"

namespace cogment {

Agent::Agent(Trial* owner, const std::string& in_actor_name, const ActorClass* actor_class, const std::string& impl,
             StubEntryType stub_entry, std::optional<std::string> config_data)
    : Actor(owner, in_actor_name, actor_class),
      stub_entry_(std::move(stub_entry)),
      config_data_(std::move(config_data)),
      impl_(impl) {
  SPDLOG_TRACE("Agent(): [{}] [{}] [{}]", to_string(trial()->id()), actor_name(), impl);

  grpc_metadata trial_header;
  trial_header.key = grpc_slice_from_static_string("trial-id");
  trial_header.value = grpc_slice_from_copied_string(to_string(trial()->id()).c_str());

  grpc_metadata actor_header;
  actor_header.key = grpc_slice_from_static_string("actor-name");
  actor_header.value = grpc_slice_from_copied_string(actor_name().c_str());

  headers_ = {trial_header, actor_header};
  options_.headers = &headers_;
}

Agent::~Agent() {
  SPDLOG_TRACE("~Agent(): [{}] [{}]", to_string(trial()->id()), actor_name());

  if (outgoing_observations_) {
    outgoing_observations_->complete();
  }
}

aom::Future<void> Agent::init() {
  SPDLOG_TRACE("Agent::init(): [{}] [{}]", to_string(trial()->id()), actor_name());

  cogment::AgentStartRequest req;

  req.set_impl_name(impl_);

  if (config_data_) {
    req.mutable_config()->set_content(config_data_.value());
  }

  for (const auto& actor : trial()->actors()) {
    auto actor_in_trial = req.add_actors_in_trial();
    actor_in_trial->set_actor_class(actor->actor_class()->name);
    actor_in_trial->set_name(actor->actor_name());
  }

  return stub_entry_->get_stub().OnStart(req, options_).then([this](auto rep) {
    (void)rep;
    (void)this;
    SPDLOG_DEBUG("Agent init start complete: [{}] [{}]", to_string(trial()->id()), actor_name());
  });
}

void Agent::dispatch_observation(cogment::Observation&& observation) {
  lazy_start_decision_stream();

  // We serialize the observations to prevent long lags where
  // rewards/messages arrive much later and cause errors.
  stub_entry_->serialize([this, obs = std::move(observation)]() {
    cogment::AgentObservationRequest req;
    *req.mutable_observation() = std::move(obs);

    outgoing_observations_->push(std::move(req));
  });
}

void Agent::dispatch_final_data(cogment::ActorPeriodData&& data) {
  ::cogment::AgentEndRequest req;
  *(req.mutable_final_data()) = std::move(data);

  stub_entry_->get_stub().OnEnd(req, options_);
}

void Agent::lazy_start_decision_stream() {
  if (!outgoing_observations_) {
    auto stream = stub_entry_->get_stub().OnObservation(options_);

    outgoing_observations_ = std::move(std::get<0>(stream));
    auto incoming_actions = std::move(std::get<1>(stream));

    std::weak_ptr trial_weak = trial()->get_shared();
    auto name = actor_name();

    incoming_actions
        .for_each([trial_weak, name](auto rep) {
          auto trial = trial_weak.lock();
          if (trial && trial->state() != Trial::InternalState::ended) {
            trial->actor_acted(name, rep.action());
            for (auto& rew : rep.rewards()) {
              trial->reward_received(rew, name);
            }
            for (auto& message : rep.messages()) {
              trial->message_received(message, name);
            }
          }
        })
        .finally([](auto) {});
  }
}

bool Agent::is_active() const {
  // Actors driven by agent services are always active since
  // they are driven by the orchestrator itself.
  return true;
}

void Agent::dispatch_reward(cogment::Reward&& reward) {
  stub_entry_->serialize([this, rew = std::move(reward)]() {
    cogment::AgentRewardRequest req;
    *req.mutable_reward() = std::move(rew);

    stub_entry_->get_stub().OnReward(req, options_).get();
  });
}

void Agent::dispatch_message(cogment::Message&& message) {
  stub_entry_->serialize([this, msg = std::move(message)]() {
    cogment::AgentMessageRequest req;
    auto new_msg = req.add_messages();
    *new_msg = std::move(msg);

    stub_entry_->get_stub().OnMessage(req, options_).get();
  });
}

}  // namespace cogment
