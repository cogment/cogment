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

#ifndef AOM_ORCHESTRATOR_TRIAL_SPEC_H
#define AOM_ORCHESTRATOR_TRIAL_SPEC_H

#include "cogment/actor.h"

namespace cogment {

// A
struct Trial_spec {
  Trial_spec(const YAML::Node& cfg_root);

  const ActorClass& get_actor_class(const std::string& class_name) const;

  const google::protobuf::Message* get_trial_config_proto() const { return trial_config_prototype; }
  const google::protobuf::Message* get_env_config_prototype() const { return env_config_prototype; }

  private:
  std::unique_ptr<google::protobuf::compiler::DiskSourceTree> source_tree_;
  std::unique_ptr<google::protobuf::compiler::Importer> importer_;
  std::unique_ptr<google::protobuf::DynamicMessageFactory> message_factory_;

  const google::protobuf::Message* trial_config_prototype = nullptr;
  const google::protobuf::Message* env_config_prototype = nullptr;

  std::vector<ActorClass> actor_classes;
};

}  // namespace cogment
#endif