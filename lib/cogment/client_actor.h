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
#include "cogment/stub_pool.h"
#include "grpc++/grpc++.h"

#include <optional>
#include <thread>

namespace cogment {

class Trial;
class Client_actor : public Actor {
public:
  using StreamType = grpc::ServerReaderWriter<cogmentAPI::TrialActionReply, cogmentAPI::TrialActionRequest>;

  Client_actor(Trial* owner, const std::string& actor_name, const ActorClass* actor_class,
               std::optional<std::string> config_data);

  ~Client_actor();

  std::future<void> init() override;
  bool is_active() const override;
  void trial_ended(std::string_view details) override;

  // Indicate that a client has claimed this actor
  std::optional<std::string> join();
  grpc::Status bind(StreamType* stream);

protected:
  void dispatch_observation(cogmentAPI::Observation&& obs) override;
  void dispatch_final_data(cogmentAPI::ActorPeriodData&& data) override;
  void dispatch_reward(cogmentAPI::Reward&& reward) override;
  void dispatch_message(cogmentAPI::Message&& message) override;

private:
  void finish_stream();

  bool m_joined;

  cogmentAPI::Action m_latest_action;
  std::optional<std::string> m_config_data;

  StreamType* m_stream;
  std::thread m_incoming_thread;

  std::promise<void> m_ready_prom;
  std::promise<void> m_finished_prom;
  std::future<void> m_finished_fut;
  grpc::Status m_incoming_stream_status;
  std::mutex m_finishing_mutex;
};
}  // namespace cogment
#endif
