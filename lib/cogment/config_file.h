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