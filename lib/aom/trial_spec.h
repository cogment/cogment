#ifndef AOM_ORCHESTRATOR_TRIAL_SPEC_H
#define AOM_ORCHESTRATOR_TRIAL_SPEC_H

#include "aom/actor.h"

namespace cogment {

// A
struct Trial_spec {
  Trial_spec(const YAML::Node& cfg_root);

  std::size_t get_class_id(const std::string class_name) const;

  std::unique_ptr<google::protobuf::compiler::DiskSourceTree> source_tree_;
  std::unique_ptr<google::protobuf::compiler::Importer> importer_;
  std::unique_ptr<google::protobuf::DynamicMessageFactory> message_factory_;

  const google::protobuf::Message* trial_config_prototype = nullptr;
  const google::protobuf::Message* env_config_prototype = nullptr;

  std::vector<ActorClass> actor_classes;
};

}  // namespace cogment
#endif