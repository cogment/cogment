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

#ifndef AOM_ORCHESTRATOR_AGENT_H
#define AOM_ORCHESTRATOR_AGENT_H

#include "cogment/actor.h"
#include "cogment/api/agent.egrpc.pb.h"
#include "cogment/stub_pool.h"

#include <optional>

namespace cogment {

class Trial;
class Agent : public Actor {
  public:
  using stub_type = std::shared_ptr<Stub_pool<cogment::AgentEndpoint>::Entry>;
  Agent(Trial* owner, std::uint32_t actor_id, const ActorClass* actor_class, stub_type stub,
        std::optional<std::string> config_data);

  ~Agent();

  Future<void> init() override;
  void terminate() override;

  void dispatch_observation(cogment::Observation&& obs) override;
  void dispatch_reward(int tick_id, const ::cogment::Reward& reward) override;
  Future<cogment::Action> request_decision(cogment::Observation&& obs) override;

  ::easy_grpc::Future<::cogment::TrialActionReply> user_acted(cogment::TrialActionRequest req) override {
    throw std::runtime_error("agent is recieving human action.");
  }

  private:
  void lazy_start_decision_stream();

  stub_type stub_;
  std::vector<grpc_metadata> headers_;

  cogment::Action latest_action_;
  std::optional<std::string> config_data_;

  std::optional<::easy_grpc::Stream_promise<::cogment::AgentDataRequest>> outgoing_observations_;
  Promise<cogment::Action> expected_action;
};
}  // namespace cogment
#endif
