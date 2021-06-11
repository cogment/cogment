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

#include "cogment/client_actor.h"
#include "cogment/trial.h"

#include "spdlog/spdlog.h"

namespace cogment {
Client_actor::Client_actor(Trial* owner, const std::string& actor_name, const ActorClass* actor_class,
                           std::optional<std::string> config_data)
    : Actor(owner, actor_name, actor_class),
      m_joined(false),
      m_config_data(std::move(config_data)),
      m_outgoing_observations_future(m_outgoing_observations.get_future()) {
  SPDLOG_TRACE("Client_actor(): [{}] [{}]", to_string(trial()->id()), actor_name);
}

Client_actor::~Client_actor() {
  SPDLOG_TRACE("~Client_actor(): [{}] [{}]", to_string(trial()->id()), actor_name());
  if (m_outgoing_observations) {
    m_outgoing_observations.complete();
  }
}

aom::Future<void> Client_actor::init() {
  SPDLOG_TRACE("Client_actor::init(): [{}] [{}]", to_string(trial()->id()), actor_name());
  // Client actors are ready once a client has connected to it.
  return m_ready_promise.get_future();
}

bool Client_actor::is_active() const { return m_joined; }

std::optional<std::string> Client_actor::join() {
  m_joined = true;
  m_ready_promise.set_value();

  return m_config_data;
}

Client_actor::Observation_future Client_actor::bind(Client_actor::Action_future actions) {
  std::weak_ptr trial_weak = trial()->get_shared();
  auto name = actor_name();

  actions
      .for_each([trial_weak, name](auto rep) {
        auto trial = trial_weak.lock();
        if (trial) {
          trial->actor_acted(name, rep.action());
        }
      })
      .finally([](auto) {});

  return std::move(m_outgoing_observations_future);
}

void Client_actor::dispatch_observation(cogment::Observation&& obs) {
  ::cogment::TrialActionReply req;
  req.set_final_data(false);
  auto new_obs = req.mutable_data()->add_observations();
  *new_obs = obs;

  m_outgoing_observations.push(std::move(req));
}

void Client_actor::dispatch_final_data(cogment::ActorPeriodData&& data) {
  ::cogment::TrialActionReply req;
  req.set_final_data(true);
  *(req.mutable_data()) = std::move(data);

  m_outgoing_observations.push(std::move(req));
}

void Client_actor::dispatch_reward(cogment::Reward&& reward) {
  ::cogment::TrialActionReply req;
  auto new_reward = req.mutable_data()->add_rewards();
  *new_reward = reward;

  m_outgoing_observations.push(std::move(req));
}

void Client_actor::dispatch_message(cogment::Message&& message) {
  ::cogment::TrialActionReply req;
  auto new_mess = req.mutable_data()->add_messages();
  *new_mess = message;

  m_outgoing_observations.push(std::move(req));
}

}  // namespace cogment
