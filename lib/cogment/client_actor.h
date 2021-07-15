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
  using Action_future = ::easy_grpc::Stream_future<::cogmentAPI::TrialActionRequest>;

  using Observation_promise = ::easy_grpc::Stream_promise<::cogmentAPI::TrialActionReply>;
  using Observation_future = ::easy_grpc::Stream_future<::cogmentAPI::TrialActionReply>;

  Client_actor(Trial* owner, const std::string& actor_name, const ActorClass* actor_class,
               std::optional<std::string> config_data);

  ~Client_actor();

  aom::Future<void> init() override;

  bool is_active() const override;

  // Indicate that a client has claimed this actor
  std::optional<std::string> join();
  Observation_future bind(Action_future actions);

protected:
  void dispatch_observation(cogmentAPI::Observation&& obs) override;
  void dispatch_final_data(cogmentAPI::ActorPeriodData&& data) override;
  void dispatch_reward(cogmentAPI::Reward&& reward) override;
  void dispatch_message(cogmentAPI::Message&& message) override;

private:
  bool m_joined;

  cogmentAPI::Action m_latest_action;
  std::optional<std::string> m_config_data;

  Observation_promise m_outgoing_observations;
  Observation_future m_outgoing_observations_future;
  std::promise<void> m_stream_end_prom;
  std::future<void> m_stream_end_fut;

  aom::Promise<void> m_ready_promise;
};
}  // namespace cogment
#endif
