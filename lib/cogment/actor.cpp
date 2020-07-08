#include "cogment/actor.h"

#include "cogment/agent.h"
#include "cogment/config_file.h"
#include "cogment/human.h"
#include "cogment/trial.h"
#include "spdlog/spdlog.h"

namespace cogment {

Actor::Actor(Trial* trial, std::uint32_t actor_id, const ActorClass* actor_class) 
    : trial_(trial)
    , actor_id_(actor_id) 
    , actor_class_(actor_class) {}


Actor::~Actor() {}

Trial* Actor::trial() const {
  return trial_;
}

std::uint32_t Actor::actor_id() const {
  return actor_id_;
}

const ActorClass* Actor::actor_class() const { 
  return actor_class_; 
}
}  // namespace cogment