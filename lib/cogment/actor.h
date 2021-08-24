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

#include "cogment/api/agent.egrpc.pb.h"
#include "cogment/api/orchestrator.pb.h"

#include "cogment/utils.h"

#include "easy_grpc/easy_grpc.h"
#include "yaml-cpp/yaml.h"

#include <google/protobuf/compiler/importer.h>
#include <google/protobuf/descriptor.h>
#include <google/protobuf/dynamic_message.h>

#include <memory>
#include <vector>

namespace cogment {

class Trial;
struct ActorClass;

class Actor {
public:
  using SrcAccumulator = std::vector<cogmentAPI::RewardSource>;
  using RewAccumulator = std::map<uint64_t, SrcAccumulator>;

  Actor(Trial* trial, const std::string& actor_name, const ActorClass* actor_class);
  virtual ~Actor();

  virtual aom::Future<void> init() = 0;

  virtual bool is_active() const = 0;

  Trial* trial() const;
  const std::string& actor_name() const;
  const ActorClass* actor_class() const;

  void add_immediate_reward_src(const cogmentAPI::RewardSource& source, const std::string& sender, uint64_t tick_id);
  void add_immediate_message(const cogmentAPI::Message& message, const std::string& source, uint64_t tick_id);

  void dispatch_tick(cogmentAPI::Observation&& obs, bool final_tick);

  aom::Future<void> last_ack() { return m_last_ack_prom.get_future(); }

protected:
  virtual void dispatch_observation(cogmentAPI::Observation&& obs) = 0;
  virtual void dispatch_final_data(cogmentAPI::ActorPeriodData&& data) = 0;
  virtual void dispatch_reward(cogmentAPI::Reward&& reward) = 0;
  virtual void dispatch_message(cogmentAPI::Message&& message) = 0;

  void ack_last() { m_last_ack_prom.set_value(); }

private:
  Trial* m_trial;
  std::string m_actor_name;
  const ActorClass* m_actor_class;
  std::mutex m_lock;

  RewAccumulator m_reward_accumulator;
  std::vector<cogmentAPI::Message> m_message_accumulator;

  aom::Promise<void> m_last_ack_prom;
};

struct ActorClass {
  std::string name;
  std::uint32_t index;

  const google::protobuf::Message* observation_space_prototype = nullptr;
  std::vector<const google::protobuf::FieldDescriptor*> cleared_observation_fields;

  const google::protobuf::Message* observation_delta_prototype = nullptr;
  std::vector<const google::protobuf::FieldDescriptor*> cleared_delta_fields;

  const google::protobuf::Message* action_space_prototype = nullptr;
  std::vector<const google::protobuf::FieldDescriptor*> cleared_action_fields;

  const google::protobuf::Message* config_prototype = nullptr;
};

}  // namespace cogment
#endif
