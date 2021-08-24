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

#ifndef AOM_ORCHESTRATOR_AGENT_ACTOR_H
#define AOM_ORCHESTRATOR_AGENT_ACTOR_H

#include "cogment/actor.h"
#include "cogment/api/agent.egrpc.pb.h"
#include "cogment/stub_pool.h"

#include <optional>

namespace cogment {

class Trial;

class Agent : public Actor {
public:
  using StubEntryType = std::shared_ptr<Stub_pool<cogmentAPI::ServiceActorSP>::Entry>;
  Agent(Trial* owner, const std::string& actor_name, const ActorClass* actor_class, const std::string& impl,
        StubEntryType stub_entry, std::optional<std::string> config_data);

  ~Agent();

  aom::Future<void> init() override;

  bool is_active() const override;

protected:
  void dispatch_observation(cogmentAPI::Observation&& obs) override;
  void dispatch_reward(cogmentAPI::Reward&& reward) override;
  void dispatch_message(cogmentAPI::Message&& message) override;
  void dispatch_final_data(cogmentAPI::ActorPeriodData&& data) override;

private:
  void process_communication_state(cogmentAPI::CommunicationState in_state, const std::string* details);
  void process_incoming_data(const cogmentAPI::ServiceActorRunOutput& data);

  StubEntryType m_stub_entry;

  cogmentAPI::Action m_latest_action;
  std::optional<std::string> m_config_data;

  std::promise<void> m_stream_end_prom;
  std::future<void> m_stream_end_fut;
  aom::Promise<void> m_init_prom;
  easy_grpc::Stream_promise<cogmentAPI::ServiceActorRunInput> m_outgoing_data;
  easy_grpc::Stream_future<cogmentAPI::ServiceActorRunOutput> m_incoming_data;

  std::string m_impl;

  // Communication
  bool m_last_sent;
  bool m_init_completed;
};

}  // namespace cogment

#endif
