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

#ifndef AOM_ORCHESTRATOR_HUMAN_H
#define AOM_ORCHESTRATOR_HUMAN_H

#include "cogment/actor.h"

namespace cogment {

class Human : public Actor {
  public:
  Human(std::string tid);
  ~Human();

  Future<void> init() override;
  void terminate() override;
  void send_final_observation(cogment::Observation&& obs) override;

  bool is_human() const override { return true; }

  void dispatch_reward(int tick_id, const ::cogment::Reward& reward) override { latest_reward_ = reward; }

  ::easy_grpc::Future<::cogment::TrialActionReply> user_acted(cogment::TrialActionRequest req) override;

  private:
  std::optional<::cogment::Reward> latest_reward_;
  Promise<cogment::Action> human_action_promise_;
  Promise<::cogment::TrialActionReply> human_observation_promise_;
};
}  // namespace cogment
#endif
