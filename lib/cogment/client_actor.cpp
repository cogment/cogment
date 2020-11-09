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

#include "cogment/client_actor.h"
#include "cogment/trial.h"

#include "spdlog/spdlog.h"

namespace cogment {
Client_actor::Client_actor(Trial* owner, std::uint32_t actor_id, const ActorClass* actor_class,
                           std::optional<std::string> config_data)
    : Actor(owner, actor_id, actor_class),
      joined_(false),
      config_data_(std::move(config_data)),
      outgoing_observations_future_(outgoing_observations_.get_future()) {}

Client_actor::~Client_actor() {}

Future<void> Client_actor::init() {
  // Client actors are ready once a client has connected to it.
  return ready_promise_.get_future();
}

bool Client_actor::is_active() const { return joined_; }

std::optional<std::string> Client_actor::join() {
  joined_ = true;
  ready_promise_.set_value();

  return config_data_;
}

Client_actor::Observation_future Client_actor::bind(Client_actor::Action_future actions) {
  std::weak_ptr trial_weak = trial()->get_shared();
  auto a_id = actor_id();

  actions
      .for_each([trial_weak, a_id](auto act) {
        auto trial = trial_weak.lock();
        if (trial) {
          trial->actor_acted(a_id, act.action());
        }
      })
      .finally([](auto) {});

  return std::move(outgoing_observations_future_);
}

void Client_actor::dispatch_observation(const cogment::Observation& obs, bool end_of_trial) {
  ::cogment::TrialActionReply req;
  req.set_final(end_of_trial);
  *req.mutable_observation() = obs;
  outgoing_observations_.push(std::move(req));

  if (end_of_trial) {
    outgoing_observations_.complete();
  }
}

void Client_actor::terminate() {
  if (outgoing_observations_) {
    outgoing_observations_.complete();
  }
}

void Client_actor::dispatch_reward(int /*tick_id*/, const ::cogment::Reward& /*reward*/) {}

}  // namespace cogment
