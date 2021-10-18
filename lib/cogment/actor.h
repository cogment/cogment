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

#ifndef AOM_ORCHESTRATOR_ACTOR_H
#define AOM_ORCHESTRATOR_ACTOR_H

#include "cogment/api/agent.grpc.pb.h"
#include "cogment/api/orchestrator.pb.h"

#include "grpc++/grpc++.h"

#include "yaml-cpp/yaml.h"

#include <google/protobuf/compiler/importer.h>
#include <google/protobuf/descriptor.h>
#include <google/protobuf/dynamic_message.h>

#include <memory>
#include <vector>
#include <future>

namespace cogment {

class Trial;
struct ActorClass;

class Actor {
public:
  using SrcAccumulator = std::vector<cogmentAPI::RewardSource>;
  using RewAccumulator = std::map<uint64_t, SrcAccumulator>;

  Actor(Trial* trial, const std::string& actor_name, const ActorClass* actor_class, const std::string& impl,
                           std::optional<std::string> config_data);
  virtual ~Actor();

  virtual std::future<void> init() = 0;
  virtual bool is_active() const = 0;
  virtual void trial_ended(std::string_view details) = 0;

  std::future<void> last_ack() { return m_last_ack_prom.get_future(); }

  Trial* trial() const { return m_trial; }
  const std::string& actor_name() const { return m_actor_name; }
  const ActorClass* actor_class() const { return m_actor_class; }
  const std::string& impl() const { return m_impl; }
  const std::optional<std::string>& config() const { return m_config_data; }

  void add_immediate_reward_src(const cogmentAPI::RewardSource& source, const std::string& sender, uint64_t tick_id);
  void send_message(const cogmentAPI::Message& message, const std::string& source, uint64_t tick_id);

  void dispatch_tick(cogmentAPI::Observation&& obs, bool final_tick);

protected:
  virtual void dispatch_observation(cogmentAPI::Observation&& obs, bool last) = 0;
  virtual void dispatch_reward(cogmentAPI::Reward&& reward) = 0;
  virtual void dispatch_message(cogmentAPI::Message&& message) = 0;

  void process_incoming_state(cogmentAPI::CommunicationState in_state, const std::string* details);
  void last_sent() { m_last_sent = true; }

private:
  Trial* const m_trial;
  const std::string m_actor_name;
  const ActorClass* m_actor_class;
  const std::string m_impl;
  std::optional<std::string> m_config_data;

  std::mutex m_lock;
  RewAccumulator m_reward_accumulator;

  bool m_last_sent;
  std::promise<void> m_last_ack_prom;
};

struct ActorClass {
  std::string name;
  std::uint32_t index;

  const google::protobuf::Message* observation_space_prototype = nullptr;
  std::vector<const google::protobuf::FieldDescriptor*> cleared_observation_fields;

  const google::protobuf::Message* action_space_prototype = nullptr;
  std::vector<const google::protobuf::FieldDescriptor*> cleared_action_fields;

  const google::protobuf::Message* config_prototype = nullptr;
};

}  // namespace cogment
#endif
