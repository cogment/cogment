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
#include "cogment/trial.h"
#include "spdlog/spdlog.h"

namespace cogment {

Actor::Actor(Trial* trial, const std::string& actor_name, const ActorClass* actor_class)
    : trial_(trial), actor_name_(actor_name), actor_class_(actor_class) {}

Actor::~Actor() {}

Trial* Actor::trial() const { return trial_; }

const std::string& Actor::actor_name() const { return actor_name_; }

const ActorClass* Actor::actor_class() const { return actor_class_; }

void Actor::add_immediate_feedback(cogment::Feedback feedback) { feedback_accumulator_.push_back(feedback); }

std::vector<cogment::Feedback> Actor::get_and_flush_immediate_feedback() { return std::move(feedback_accumulator_); }

void Actor::add_immediate_message(cogment::Message message) { message_accumulator_.push_back(message); }

std::vector<cogment::Message> Actor::get_and_flush_immediate_message() { return std::move(message_accumulator_); }
}  // namespace cogment