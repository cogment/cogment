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

#ifndef COGMENT_ORCHESTRATOR_CONFIG_FILE_H
#define COGMENT_ORCHESTRATOR_CONFIG_FILE_H

#include "yaml-cpp/yaml.h"

namespace cogment {
namespace cfg_file {
// root
constexpr const char* import_key = "import";
constexpr const char* commands_key = "commands";
constexpr const char* trial_key = "trial";
constexpr const char* environment_key = "environment";
constexpr const char* actors_key = "actor_classes";
constexpr const char* params_key = "trial_params";
constexpr const char* datalog_key = "datalog";

// import
constexpr const char* i_proto_key = "proto";
constexpr const char* i_python_key = "python";
constexpr const char* i_javascript_key = "javascript";

// trial
constexpr const char* t_config_type_key = "config_type";
constexpr const char* t_pre_hooks_key = "pre_hooks";

// environment
constexpr const char* e_config_type_key = "config_type";

// actor classes
constexpr const char* ac_name_key = "name";
constexpr const char* ac_action_key = "action";
constexpr const char* ac_act_space_key = "space";
constexpr const char* ac_observation_key = "observation";
constexpr const char* ac_obs_space_key = "space";
constexpr const char* ac_config_type_key = "config_type";

// params
constexpr const char* p_trial_config_key = "trial_config";
constexpr const char* p_datalog_key = "datalog";
constexpr const char* p_log_endpoint_key = "endpoint";
constexpr const char* p_log_exclude_fields_key = "exclude_fields";
constexpr const char* p_environment_key = "environment";
constexpr const char* p_env_name_key = "name";
constexpr const char* p_env_endpoint_key = "endpoint";
constexpr const char* p_env_implementation_key = "implementation";
constexpr const char* p_actors_key = "actors";
constexpr const char* p_act_name_key = "name";
constexpr const char* p_act_ac_name_key = "actor_class";
constexpr const char* p_act_endpoint_key = "endpoint";
constexpr const char* p_act_implementation_key = "implementation";
constexpr const char* p_max_steps_key = "max_steps";
constexpr const char* p_max_inactivity_key = "max_inactivity";

// datalog
constexpr const char* d_fields_key = "fields";
constexpr const char* d_fld_exclude_key = "exclude";
constexpr const char* d_type_key = "type";
constexpr const char* d_url_key = "url";

}  // namespace cfg_file
}  // namespace cogment
#endif