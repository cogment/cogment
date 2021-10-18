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
#include "cogment/stub_pool.h"

#include <optional>

namespace cogment {

class Trial;

// TODO: See what common aspects (with 'ClientActor') could be moved to 'Actor'
class ServiceActor : public Actor {
public:
  using StubEntryType = std::shared_ptr<StubPool<cogmentAPI::ServiceActorSP>::Entry>;
  using StreamType = grpc::ClientReaderWriter<cogmentAPI::ActorRunTrialInput, cogmentAPI::ActorRunTrialOutput>;

  ServiceActor(Trial* owner, const std::string& actor_name, const std::string& actor_class, const std::string& impl,
        StubEntryType stub_entry, std::optional<std::string> config_data);

  ~ServiceActor();

  std::future<void> init() override;
  bool is_active() const override;
  void trial_ended(std::string_view details) override;

protected:
  void dispatch_observation(cogmentAPI::Observation&& obs, bool last) override;
  void dispatch_reward(cogmentAPI::Reward&& reward) override;
  void dispatch_message(cogmentAPI::Message&& message) override;

private:
  void write_to_stream(cogmentAPI::ActorRunTrialInput&& data);
  void process_incoming_data(cogmentAPI::ActorRunTrialOutput&& data);

  StubEntryType m_stub_entry;

  cogmentAPI::Action m_latest_action;

  std::promise<void> m_init_prom;
  std::unique_ptr<StreamType> m_stream;
  bool m_stream_valid;
  grpc::ClientContext m_context;
  std::future<void> m_incoming_thread;

  bool m_init_completed;
  mutable std::mutex m_writing;
};

}  // namespace cogment

#endif
