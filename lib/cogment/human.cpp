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

#include "cogment/human.h"
#include "spdlog/spdlog.h"

namespace cogment {
Human::Human(std::string tid) : Actor(std::move(tid)) {}

Human::~Human() {}

void Human::terminate() { human_action_promise_ = Promise<cogment::Action>{}; }

Future<void> Human::init() {
  Promise<void> prom;
  auto result = prom.get_future();
  prom.set_value();
  return result;
}

::easy_grpc::Future<::cogment::TrialActionReply> Human::user_acted(cogment::TrialActionRequest req) {
  human_observation_promise_ = Promise<::cogment::TrialActionReply>();
  auto result = human_observation_promise_.get_future();

  human_action_promise_.set_value(req.action());
  return result;
}

void Human::send_final_observation(cogment::Observation&& obs) {
  ::cogment::TrialActionReply rep;

  *rep.mutable_observation() = std::move(obs);

  if (latest_reward_) {
    *rep.mutable_reward() = std::move(*latest_reward_);
  }
  latest_reward_ = std::nullopt;

  rep.set_trial_is_over(true);

  human_observation_promise_.set_value(std::move(rep));
}

}  // namespace cogment