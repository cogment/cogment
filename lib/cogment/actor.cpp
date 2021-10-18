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

#include "cogment/utils.h"
#include "cogment/config_file.h"
#include "cogment/trial.h"
#include "spdlog/spdlog.h"

namespace {

// Collapses a collection of reward sources into a reward.
cogmentAPI::Reward build_reward(cogment::Actor::SrcAccumulator* src_acc) {
  cogmentAPI::Reward reward;

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

Actor::Actor(Trial* trial, const std::string& actor_name, const ActorClass* actor_class, const std::string& impl,
                           std::optional<std::string> config_data) :
    m_trial(trial),
    m_actor_name(actor_name),
    m_actor_class(actor_class),
    m_impl(impl),
    m_config_data(std::move(config_data)),
    m_last_sent(false) {}

Actor::~Actor() {}

void Actor::add_immediate_reward_src(const cogmentAPI::RewardSource& source, const std::string& sender, uint64_t tick_id) {
  const std::lock_guard<std::mutex> lg(m_lock);
  auto& src_acc = m_reward_accumulator[tick_id];
  src_acc.emplace_back(source);
  src_acc.back().set_sender_name(sender);
}

// TODO: Send message immediately
void Actor::send_message(const cogmentAPI::Message& message, const std::string& sender, uint64_t tick_id) {
  cogmentAPI::Message msg(message);
  msg.set_tick_id(tick_id);
  msg.set_sender_name(sender);
  msg.set_receiver_name(m_actor_name);  // Because of possible wildcards in message receiver

  dispatch_message(std::move(msg));
}

void Actor::dispatch_tick(cogmentAPI::Observation&& obs, bool final_tick) {
  RewAccumulator reward_acc;
  {
    const std::lock_guard<std::mutex> lg(m_lock);
    reward_acc.swap(m_reward_accumulator);
  }

  try {
    process_rewards(&reward_acc, m_actor_name, [this](cogmentAPI::Reward&& rew) {
      dispatch_reward(std::move(rew));
    });

    dispatch_observation(std::move(obs), final_tick);
  }
  catch(const std::exception& exc) {
    spdlog::error("Trial [{}] - Actor [{}]: Failed to process outgoing data [{}]", m_trial->id(), m_actor_name, exc.what());
  }
  catch(...) {
    spdlog::error("Trial [{}] - Actor [{}]: Failed to process outgoing data", m_trial->id(), m_actor_name);
  }
}

void Actor::process_incoming_state(cogmentAPI::CommunicationState in_state, const std::string* details) {
  switch(in_state) {
    case cogmentAPI::CommunicationState::UNKNOWN_COM_STATE:
      if (details != nullptr) {
        throw MakeException<std::invalid_argument>("Unknown communication state: [%s]", details->c_str());
      } else {
        throw MakeException<std::invalid_argument>("Unknown communication state");
      }
      break;

    case cogmentAPI::CommunicationState::NORMAL:
      if (details != nullptr) {
        spdlog::info("Trial [{}] - Actor [{}] Communication details received [{}]", trial()->id(), actor_name(), *details);
      } else {
        spdlog::warn("Trial [{}] - Actor [{}] No data in normal communication received", trial()->id(), actor_name());
      }
      break;

    case cogmentAPI::CommunicationState::HEARTBEAT:
      SPDLOG_TRACE("Trial [{}] - Actor [{}] 'HEARTBEAT' received", trial()->id(), actor_name());
      if (details != nullptr) {
        spdlog::info("Trial [{}] - Actor [{}] Heartbeat requested [{}]", trial()->id(), actor_name(), *details);
      }
      // TODO : manage heartbeats
      break;

    case cogmentAPI::CommunicationState::LAST:
      if (details != nullptr) {
        spdlog::error("Trial [{}] - Actor [{}] Unexpected communication state (LAST) received [{}]", trial()->id(), actor_name(), *details);
      } else {
        spdlog::error("Trial [{}] - Actor [{}] Unexpected communication state (LAST) received", trial()->id(), actor_name());
      }
      break;

    case cogmentAPI::CommunicationState::LAST_ACK:
      SPDLOG_DEBUG("Trial [{}] - Actor [{}] 'LAST_ACK' received", trial()->id(), actor_name());
      if (!m_last_sent) {
        if (details != nullptr) {
          spdlog::error("Trial [{}] - Actor [{}] Unexpected communication state (LAST_ACK) received [{}]", trial()->id(), actor_name(), *details);
        } else {
          spdlog::error("Trial [{}] - Actor [{}] Unexpected communication state (LAST_ACK) received", trial()->id(), actor_name());
        }
      }
      // TODO: Should we accept even if the "LAST" was not sent?
      //       This could be used to indicate that the actor has finished interacting with the trial.
      m_last_ack_prom.set_value();
      break;

    case cogmentAPI::CommunicationState::END:
      // TODO: Decide what to do about "END" received from actors
      if (details != nullptr) {
        spdlog::error("Trial [{}] - Actor [{}] Unexpected communication state (END) received [{}]", trial()->id(), actor_name(), *details);
      } else {
        spdlog::error("Trial [{}] - Actor [{}] Unexpected communication state (END) received", trial()->id(), actor_name());
      }
      break;

    default:
      throw MakeException<std::invalid_argument>("Invalid communication state: [%d]", static_cast<int>(in_state));
      break;
  }
}

}  // namespace cogment
