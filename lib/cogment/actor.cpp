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

#include "cogment/actor.h"

#include "cogment/config_file.h"
#include "cogment/human.h"
#include "cogment/trial.h"
#include "spdlog/spdlog.h"

namespace cogment {

Actor::Actor(Trial* trial, std::uint32_t actor_id, const ActorClass* actor_class)
    : trial_(trial), actor_id_(actor_id), actor_class_(actor_class) {}

Actor::~Actor() {}

Trial* Actor::trial() const { return trial_; }

std::uint32_t Actor::actor_id() const { return actor_id_; }

const ActorClass* Actor::actor_class() const { return actor_class_; }
}  // namespace cogment