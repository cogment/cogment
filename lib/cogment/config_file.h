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

#ifndef AOM_CONFIG_FILE_H
#define AOM_CONFIG_FILE_H

#include <optional>
#include "yaml-cpp/yaml.h"

namespace cogment {
namespace cfg_file {
// root
constexpr const char* actors_key = "actor_classes";

// actor_classes
constexpr const char* ac_action_space_key = "action_space";
constexpr const char* instances_key = "instances";

// actors
constexpr const char* a_type_key = "type";
constexpr const char* a_url_key = "url";
constexpr const char* a_count_key = "count";
constexpr const char* actortype_human = "human";
constexpr const char* actortype_agent = "agent";

}  // namespace cfg_file
}  // namespace cogment
#endif