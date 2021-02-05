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

#ifndef AOM_ORCHESTRATOR_CLIENT_ACTOR_H
#define AOM_ORCHESTRATOR_CLIENT_ACTOR_H

#include "cogment/actor.h"
#include "cogment/api/agent.egrpc.pb.h"
#include "cogment/stub_pool.h"

#include <optional>

namespace cogment {

class Trial;
class Client_actor : public Actor {
  public:
  using Action_future = ::easy_grpc::Stream_future<::cogment::TrialActionRequest>;

  using Observation_promise = ::easy_grpc::Stream_promise<::cogment::TrialActionReply>;
  using Observation_future = ::easy_grpc::Stream_future<::cogment::TrialActionReply>;

  Client_actor(Trial* owner, const std::string& actor_name, const ActorClass* actor_class,
               std::optional<std::string> config_data);

  ~Client_actor();

  Future<void> init() override;

  void dispatch_observation(const cogment::Observation& obs, bool end_of_trial) override;
  void dispatch_reward(const ::cogment::Reward& reward) override;
  void dispatch_message(int tick_id, const ::cogment::Message& message) override;

  bool is_active() const override;

  // Indicate that a client has claimed this actor
  std::optional<std::string> join();
  Observation_future bind(Action_future actions);

  private:
  bool joined_;

  cogment::Action latest_action_;
  std::optional<std::string> config_data_;

  Observation_promise outgoing_observations_;
  Observation_future outgoing_observations_future_;

  Promise<void> ready_promise_;
};
}  // namespace cogment
#endif
