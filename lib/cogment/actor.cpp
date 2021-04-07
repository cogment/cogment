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

#include "cogment/actor.h"

#include "cogment/config_file.h"
#include "cogment/reward.h"
#include "cogment/trial.h"
#include "spdlog/spdlog.h"

namespace cogment {

Actor::Actor(Trial* trial, const std::string& actor_name, const ActorClass* actor_class)
    : trial_(trial), actor_name_(actor_name), actor_class_(actor_class) {}

Actor::~Actor() {}

Trial* Actor::trial() const { return trial_; }

const std::string& Actor::actor_name() const { return actor_name_; }

const ActorClass* Actor::actor_class() const { return actor_class_; }

void Actor::add_immediate_reward_src(const cogment::RewardSource& source, const std::string& sender) {
  reward_src_accumulator_.emplace_back(source);
  reward_src_accumulator_.back().set_sender_name(sender);
}

void Actor::add_immediate_message(const cogment::Message& message, const std::string& sender) {
  message_accumulator_.emplace_back(message);
  message_accumulator_.back().set_sender_name(sender);
}

void Actor::dispatch_tick(cogment::Observation&& obs, bool final_tick) {
  // TODO: Some of the messages and rewards should be sent with previous tick since they came with
  //       the actions in the previous tick (in reponse to obs in that tick).
  const auto tick_id = obs.tick_id();

  auto sources = std::move(reward_src_accumulator_);
  auto messages = std::move(message_accumulator_);

  if (!final_tick) {
    if (!sources.empty()) {
      auto reward = build_reward(sources);
      reward.set_tick_id(tick_id);
      reward.set_receiver_name(actor_name_);
      dispatch_reward(std::move(reward));
    }

    if (!messages.empty()) {
      for (auto& message : messages) {
        message.set_tick_id(tick_id);
        dispatch_message(std::move(message));
      }
    }

    dispatch_observation(std::move(obs));
  }
  else {
    cogment::ActorPeriodData data;

    if (!sources.empty()) {
      auto reward = build_reward(sources);
      reward.set_tick_id(tick_id);
      reward.set_receiver_name(actor_name_);
      auto new_reward = data.add_rewards();
      *new_reward = std::move(reward);
    }

    if (!messages.empty()) {
      for (auto& message : messages) {
        message.set_tick_id(tick_id);
        auto new_msg = data.add_messages();
        *new_msg = std::move(message);
      }
    }

    auto new_obs = data.add_observations();
    *new_obs = std::move(obs);

    dispatch_final_data(std::move(data));
  }
}

}  // namespace cogment
