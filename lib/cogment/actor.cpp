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

#include "cogment/actor.h"

#include "cogment/config_file.h"
#include "cogment/trial.h"
#include "spdlog/spdlog.h"

namespace {

// Collapses a collection of reward sources into a reward.
cogment::Reward build_reward(cogment::Actor::SrcAccumulator* src_acc) {
  cogment::Reward reward;

  float value_accum = 0.0f;
  float confidence_accum = 0.0f;

  for (auto& src : *src_acc) {
    auto fb_conf = src.confidence();

    if (fb_conf > 0.0f) {
      value_accum += src.value() * fb_conf;
      confidence_accum += fb_conf;
    }

    auto new_src = reward.add_sources();
    *new_src = std::move(src);
  }

  if (confidence_accum > 0.0f) {
    value_accum /= confidence_accum;
  }

  reward.set_value(value_accum);
  return reward;
}

template <class FUNCTION>
void process_rewards(cogment::Actor::RewAccumulator* rew_acc, const std::string& name, FUNCTION func) {
  for (auto& tick_sources : *rew_acc) {
    const uint64_t tick_id = tick_sources.first;
    auto& sources = tick_sources.second;

    auto reward = build_reward(&sources);
    reward.set_tick_id(tick_id);
    reward.set_receiver_name(name);
    func(std::move(reward));
  }

  rew_acc->clear();
}

}  // namespace

namespace cogment {

Actor::Actor(Trial* trial, const std::string& actor_name, const ActorClass* actor_class)
    : m_trial(trial), m_actor_name(actor_name), m_actor_class(actor_class) {}

Actor::~Actor() {}

Trial* Actor::trial() const { return m_trial; }

const std::string& Actor::actor_name() const { return m_actor_name; }

const ActorClass* Actor::actor_class() const { return m_actor_class; }

void Actor::add_immediate_reward_src(const cogment::RewardSource& source, const std::string& sender, uint64_t tick_id) {
  const std::lock_guard<std::mutex> lg(m_lock);
  auto& src_acc = m_reward_accumulator[tick_id];
  src_acc.emplace_back(source);
  src_acc.back().set_sender_name(sender);
}

void Actor::add_immediate_message(const cogment::Message& message, const std::string& sender, uint64_t tick_id) {
  const std::lock_guard<std::mutex> lg(m_lock);
  m_message_accumulator.emplace_back(message);
  m_message_accumulator.back().set_tick_id(tick_id);
  m_message_accumulator.back().set_sender_name(sender);
}

void Actor::dispatch_tick(cogment::Observation&& obs, bool final_tick) {
  RewAccumulator reward_acc;
  std::vector<cogment::Message> msg_acc;
  {
    const std::lock_guard<std::mutex> lg(m_lock);
    reward_acc.swap(m_reward_accumulator);
    msg_acc.swap(m_message_accumulator);
  }

  if (!final_tick) {
    process_rewards(&reward_acc, m_actor_name, [this](cogment::Reward&& rew) { dispatch_reward(std::move(rew)); });

    for (auto& message : msg_acc) {
      dispatch_message(std::move(message));
    }

    dispatch_observation(std::move(obs));
  }
  else {
    cogment::ActorPeriodData data;

    process_rewards(&reward_acc, m_actor_name, [&data](cogment::Reward&& rew) {
      auto new_reward = data.add_rewards();
      *new_reward = std::move(rew);
    });

    for (auto& message : msg_acc) {
      auto new_msg = data.add_messages();
      *new_msg = std::move(message);
    }

    auto new_obs = data.add_observations();
    *new_obs = std::move(obs);

    dispatch_final_data(std::move(data));
  }
}

}  // namespace cogment
