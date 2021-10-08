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
#include "grpc++/grpc++.h"

#include <optional>
#include <thread>

namespace cogment {

class Trial;

// TODO: See what common aspects (with 'ServiceActor') could be moved to 'Actor'
class ClientActor : public Actor {
public:
  using StreamType = grpc::ServerReaderWriter<cogmentAPI::ActorRunTrialInput, cogmentAPI::ActorRunTrialOutput>;

  ClientActor(Trial* owner, const std::string& actor_name, const ActorClass* actor_class, const std::string& impl,
               std::optional<std::string> config_data);

  ~ClientActor();

  std::future<void> init() override;
  bool is_active() const override;
  void trial_ended(std::string_view details) override;

  static grpc::Status run_an_actor(std::weak_ptr<Trial> trial, StreamType* stream);

protected:
  void dispatch_observation(cogmentAPI::Observation&& obs) override;
  void dispatch_final_data(cogmentAPI::ActorPeriodData&& data) override;  // TODO: Refactor this for API2.0
  void dispatch_reward(cogmentAPI::Reward&& reward) override;
  void dispatch_message(cogmentAPI::Message&& message) override;

private:
  grpc::Status run(StreamType* stream);
  void process_incoming_data(cogmentAPI::ActorRunTrialOutput&& data);
  void finish_stream();
  void write_to_stream(cogmentAPI::ActorRunTrialInput&& data);

  cogmentAPI::Action m_latest_action;

  StreamType* m_stream;
  bool m_stream_valid;
  std::future<void> m_incoming_thread;

  std::promise<void> m_ready_prom;
  std::promise<void> m_finished_prom;
  std::future<void> m_finished_fut;
  grpc::Status m_incoming_stream_status;

  std::mutex m_finishing_mutex;
  mutable std::mutex m_active_mutex;
  mutable std::mutex m_writing;
};
}  // namespace cogment
#endif
