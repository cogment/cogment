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
  Actor(Trial* trial, const std::string& actor_name, const ActorClass* actor_class);
  virtual ~Actor();

  virtual aom::Future<void> init() = 0;

  virtual void send_final_observation(cogment::Observation&& /*obs*/) {}

  virtual void dispatch_observation(const cogment::Observation& obs, bool end_of_trial) = 0;
  virtual void dispatch_reward(int tick_id, const ::cogment::Reward& reward) = 0;
  virtual void dispatch_message(int tick_id, const ::cogment::Message& message) = 0;

  virtual bool is_active() const = 0;

  Trial* trial() const;
  const std::string& actor_name() const;
  const ActorClass* actor_class() const;

  void add_immediate_feedback(cogment::Feedback feedback);
  std::vector<cogment::Feedback> get_and_flush_immediate_feedback();

  void add_immediate_message(const cogment::Message& message, const std::string& source);
  std::vector<cogment::Message> get_and_flush_immediate_message();

  private:
  Trial* trial_;
  std::string actor_name_;
  const ActorClass* actor_class_;

  // This accumulates non-retroactive feedbacks.
  std::vector<cogment::Feedback> feedback_accumulator_;

  std::vector<cogment::Message> message_accumulator_;
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
