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

#include "cogment/trial_spec.h"
#include "cogment/config_file.h"
#include "spdlog/spdlog.h"

namespace {
class ProtoErrorCollector : public google::protobuf::compiler::MultiFileErrorCollector {
  public:
  void AddError(const google::protobuf::string& filename, int line, int column,
                const google::protobuf::string& message) override {
    (void)filename;
    (void)line;
    (void)column;
    spdlog::error("e: {}", message);
  }

  void AddWarning(const google::protobuf::string& filename, int line, int column,
                  const google::protobuf::string& message) override {
    (void)filename;
    (void)line;
    (void)column;
    spdlog::warn("w: {}", message);
  }
};
}  // namespace

namespace cogment {

Trial_spec::Trial_spec(const YAML::Node& root) {
  ProtoErrorCollector error_collector_;

  source_tree_ = std::make_unique<google::protobuf::compiler::DiskSourceTree>();
  source_tree_->MapPath("", ".");

  importer_ = std::make_unique<google::protobuf::compiler::Importer>(source_tree_.get(), &error_collector_);

  for (const auto& i : root[cfg_file::import_key][cfg_file::i_proto_key]) {
    spdlog::info("Importing protobuf: {}", i.as<std::string>());
    auto fd = importer_->Import(i.as<std::string>());

    if (!fd) {
      spdlog::error("Failed to load proto file: {}", i.as<std::string>());
      throw std::runtime_error("init failure");
    }
  }

  message_factory_ = std::make_unique<google::protobuf::DynamicMessageFactory>(importer_->pool());

  if (root[cfg_file::environment_key] != nullptr &&
      root[cfg_file::environment_key][cfg_file::e_config_type_key] != nullptr) {
    auto type = root[cfg_file::environment_key][cfg_file::e_config_type_key].as<std::string>();
    const auto* config_type = importer_->pool()->FindMessageTypeByName(type);
    if (config_type == nullptr) {
      spdlog::error("Failed to lookup message type: {}", type);
      throw std::runtime_error("init failure");
    }

    env_config_prototype = message_factory_->GetPrototype(config_type);
  }

  if (root[cfg_file::trial_key] != nullptr && root[cfg_file::trial_key][cfg_file::t_config_type_key] != nullptr) {
    auto type = root[cfg_file::trial_key][cfg_file::t_config_type_key].as<std::string>();
    const auto* config_type = importer_->pool()->FindMessageTypeByName(type);
    if (config_type == nullptr) {
      spdlog::error("Failed to lookup message type: {}", type);
      throw std::runtime_error("init failure");
    }

    trial_config_prototype = message_factory_->GetPrototype(config_type);
  }

  for (const auto& a_class : root[cfg_file::actors_key]) {
    actor_classes.push_back({});

    auto& actor_class = actor_classes.back();
    actor_class.name = a_class[cfg_file::ac_name_key].as<std::string>();
    spdlog::info("Adding actor class {}", actor_class.name);

    if (a_class[cfg_file::ac_config_type_key] != nullptr) {
      auto type = a_class[cfg_file::ac_config_type_key].as<std::string>();
      const auto* config_type = importer_->pool()->FindMessageTypeByName(type);
      if (config_type == nullptr) {
        spdlog::error("Failed to lookup message type: {}", type);
        throw std::runtime_error("init failure");
      }

      actor_class.config_prototype = message_factory_->GetPrototype(config_type);
    }

    auto obs_space = a_class[cfg_file::ac_observation_key][cfg_file::ac_obs_space_key].as<std::string>();
    const auto* observation_space = importer_->pool()->FindMessageTypeByName(obs_space);
    if (observation_space == nullptr) {
      spdlog::error("Failed to lookup message type: \"{}\"", obs_space);
      throw std::runtime_error("init failure");
    }

    actor_class.observation_space_prototype = message_factory_->GetPrototype(observation_space);

    if (root[cfg_file::datalog_key] != nullptr && root[cfg_file::datalog_key][cfg_file::d_fields_key] != nullptr &&
        root[cfg_file::datalog_key][cfg_file::d_fields_key][cfg_file::d_fld_exclude_key] != nullptr) {
      for (const auto& f : root[cfg_file::datalog_key][cfg_file::d_fields_key][cfg_file::d_fld_exclude_key]) {
        auto field_name = f.as<std::string>();
        if (field_name.find(observation_space->full_name()) == 0) {
          field_name = field_name.substr(observation_space->full_name().size() + 1);
        }
        else {
          continue;
        }

        const auto* x = observation_space->FindFieldByName(field_name);
        if (x != nullptr) {
          actor_class.cleared_observation_fields.push_back(x);
        }
      }
    }
    if (a_class[cfg_file::ac_observation_key][cfg_file::ac_obs_delta_key] != nullptr) {
      auto delta = a_class[cfg_file::ac_observation_key][cfg_file::ac_obs_delta_key].as<std::string>();
      const auto* observation_delta = importer_->pool()->FindMessageTypeByName(delta);
      if (observation_delta == nullptr) {
        spdlog::error("Failed to lookup message type: {}", delta);
        throw std::runtime_error("init failure");
      }

      actor_class.observation_delta_prototype = message_factory_->GetPrototype(observation_delta);
      if (root[cfg_file::datalog_key] != nullptr && root[cfg_file::datalog_key][cfg_file::d_fields_key] != nullptr &&
          root[cfg_file::datalog_key][cfg_file::d_fields_key][cfg_file::d_fld_exclude_key] != nullptr) {
        for (const auto& f : root[cfg_file::datalog_key][cfg_file::d_fields_key][cfg_file::d_fld_exclude_key]) {
          auto field_name = f.as<std::string>();
          if (field_name.find(observation_delta->full_name()) == 0 &&
              field_name.size() > observation_delta->full_name().size() &&
              field_name[observation_delta->full_name().size()] == '.') {
            field_name = field_name.substr(observation_delta->full_name().size() + 1);
          }
          else {
            continue;
          }

          const auto* x = observation_delta->FindFieldByName(field_name);
          if (x != nullptr) {
            actor_class.cleared_delta_fields.push_back(x);
          }
        }
      }
    }
    else {
      actor_class.observation_delta_prototype = actor_class.observation_space_prototype;
      actor_class.cleared_delta_fields = actor_class.cleared_observation_fields;
    }

    spdlog::info("Clearing {} delta fields", actor_class.cleared_delta_fields.size());

    auto act_space = a_class[cfg_file::ac_action_key][cfg_file::ac_act_space_key].as<std::string>();
    const auto* action_space = importer_->pool()->FindMessageTypeByName(act_space);
    if (action_space == nullptr) {
      spdlog::error("Failed to lookup message type: \"{}\"", act_space);
      throw std::runtime_error("init failure");
    }
    actor_class.action_space_prototype = message_factory_->GetPrototype(action_space);
  }
}

const ActorClass& Trial_spec::get_actor_class(const std::string& class_name) const {
  for (const auto& actor_class : actor_classes) {
    if (actor_class.name == class_name) {
      return actor_class;
    }
  }

  spdlog::error("trying to use unregistered actor class: {}", class_name);
  throw std::runtime_error("unknown actor class");
}
}  // namespace cogment
