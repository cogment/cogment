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

#ifndef NDEBUG
  #define SPDLOG_ACTIVE_LEVEL SPDLOG_LEVEL_TRACE
#endif

#include "cogment/trial_params.h"
#include "cogment/base64.h"
#include "cogment/config_file.h"
#include "cogment/utils.h"

#include <google/protobuf/util/json_util.h>
#include "spdlog/spdlog.h"

namespace cogment {

namespace {

std::string yaml_to_json(YAML::Node yaml) {
  yaml.SetStyle(YAML::EmitterStyle::Flow);
  YAML::Emitter emmiter;
  emmiter << YAML::DoubleQuoted << yaml;

  return emmiter.c_str();
}

}  // namespace

// Loads and interprets the default params structure from the root cogment.yaml
cogmentAPI::TrialParams load_params(const YAML::Node& yaml) {
  cogmentAPI::TrialParams result;

  if (yaml[cfg_file::params_key] != nullptr) {
    YAML::Node yaml_params = yaml[cfg_file::params_key];

    auto json_params = yaml_to_json(yaml_params);
    auto status = google::protobuf::util::JsonStringToMessage(json_params, &result);
    if (!status.ok()) {
      spdlog::error("Problematic parameters: {}", json_params);
      spdlog::debug("Problematic message type: {}", result.descriptor()->DebugString());
      throw MakeException("Problem rebuilding trial params [%s]", status.error_message().as_string().c_str());
    }
  }

  spdlog::debug("Default trial params:\n {}", result.DebugString());
  return result;
}
}  // namespace cogment
