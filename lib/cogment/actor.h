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
  Actor(std::string tid) : trial_id_(std::move(tid)) {}

  virtual ~Actor();

  void set_actor_id(int actor_id) { actor_id_ = actor_id; }
  virtual aom::Future<void> init() = 0;

  virtual bool is_human() const { return false; }
  virtual void terminate() {}
  virtual void send_final_observation(cogment::Observation&& obs) {}

  virtual ::easy_grpc::Future<::cogment::TrialActionReply> user_acted(
      cogment::TrialActionRequest req) = 0;

  virtual void dispatch_reward(int tick_id,
                               const ::cogment::Reward& reward) = 0;
  virtual Future<cogment::Action> request_decision(
      cogment::Observation&& obs) = 0;

  const std::string& trial_id() const { return trial_id_; }
  int actor_id() const { return actor_id_; };

  // TODO: make accessor for this.
  const ActorClass* actor_class;

 private:
  std::string trial_id_;
  int actor_id_ = -1;
};

struct ActorClass {
  std::string name;
  std::uint32_t index;

  const google::protobuf::Message* observation_space_prototype = nullptr;
  std::vector<const google::protobuf::FieldDescriptor*>
      cleared_observation_fields;

  const google::protobuf::Message* observation_delta_prototype = nullptr;
  std::vector<const google::protobuf::FieldDescriptor*> cleared_delta_fields;

  const google::protobuf::Message* action_space_prototype = nullptr;
  std::vector<const google::protobuf::FieldDescriptor*> cleared_action_fields;

  const google::protobuf::Message* config_prototype = nullptr;
};

}  // namespace cogment
#endif