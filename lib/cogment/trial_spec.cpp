#include "cogment/trial_spec.h"
#include "cogment/config_file.h"
#include "spdlog/spdlog.h"

namespace {
class ProtoErrorCollector
    : public google::protobuf::compiler::MultiFileErrorCollector {
 public:
  void AddError(const google::protobuf::string& filename, int line, int column,
                const google::protobuf::string& message) override {
    spdlog::error("{}", message);
  }

  void AddWarning(const google::protobuf::string& filename, int line,
                  int column,
                  const google::protobuf::string& message) override {
    spdlog::warn("{}", message);
  }
};
}  // namespace

namespace cogment {

Trial_spec::Trial_spec(const YAML::Node& root) {
  ProtoErrorCollector error_collector_;

  source_tree_ = std::make_unique<google::protobuf::compiler::DiskSourceTree>();
  source_tree_->MapPath("", ".");

  importer_ = std::make_unique<google::protobuf::compiler::Importer>(
      source_tree_.get(), &error_collector_);

  for (const auto& i : root["import"]["proto"]) {
    spdlog::info("importing protobuf: {}", i.as<std::string>());
    importer_->Import(i.as<std::string>());
  }

  message_factory_ = std::make_unique<google::protobuf::DynamicMessageFactory>(
      importer_->pool());

  if (root["environment"] && root["environment"]["config_type"]) {
    auto config_type = importer_->pool()->FindMessageTypeByName(
        root["environment"]["config_type"].as<std::string>());
    if (!config_type) {
      spdlog::error("Failed to lookup message type: {}",
                    root["environment"]["config_type"].as<std::string>());
      throw std::runtime_error("init failure");
    }

    env_config_prototype = message_factory_->GetPrototype(config_type);
  }

  if (root["trial"] && root["trial"]["config_type"]) {
    auto config_type = importer_->pool()->FindMessageTypeByName(
        root["trial"]["config_type"].as<std::string>());
    if (!config_type) {
      spdlog::error("Failed to lookup message type: {}",
                    root["trial"]["config_type"].as<std::string>());
      throw std::runtime_error("init failure");
    }

    trial_config_prototype = message_factory_->GetPrototype(config_type);
  }

  const auto& actors = root[cfg_file::actors_key];

  for (const auto& a_class : root[cfg_file::actors_key]) {
    actor_classes.push_back({});

    auto& actor_class = actor_classes.back();
    actor_class.name = a_class["id"].as<std::string>();
    spdlog::info("Adding actor class {}", actor_class.name);

    if (a_class["config_type"]) {
      auto config_type = importer_->pool()->FindMessageTypeByName(
          a_class["config_type"].as<std::string>());
      if (!config_type) {
        spdlog::error("Failed to lookup message type: {}",
                      a_class["config_type"].as<std::string>());
        throw std::runtime_error("init failure");
      }

      actor_class.config_prototype =
          message_factory_->GetPrototype(config_type);
    }

    auto observation_space = importer_->pool()->FindMessageTypeByName(
        a_class["observation"]["space"].as<std::string>());
    if (!observation_space) {
      spdlog::error("Failed to lookup message type: {}",
                    a_class["observation"]["space"].as<std::string>());
      throw std::runtime_error("init failure");
    }

    actor_class.observation_space_prototype =
        message_factory_->GetPrototype(observation_space);

    if (root["datalog"] && root["datalog"]["fields"] &&
        root["datalog"]["fields"]["exclude"]) {
      for (const auto& f : root["datalog"]["fields"]["exclude"]) {
        auto field_name = f.as<std::string>();
        if (field_name.find(observation_space->full_name()) == 0) {
          field_name =
              field_name.substr(observation_space->full_name().size() + 1);
        } else {
          continue;
        }

        auto x = observation_space->FindFieldByName(field_name);
        if (x) {
          actor_class.cleared_observation_fields.push_back(x);
        }
      }
    }
    if (a_class["observation"]["delta"]) {
      auto observation_delta = importer_->pool()->FindMessageTypeByName(
          a_class["observation"]["delta"].as<std::string>());
      if (!observation_delta) {
        spdlog::error("Failed to lookup message type: {}",
                      a_class["observation"]["delta"].as<std::string>());
        throw std::runtime_error("init failure");
      }

      actor_class.observation_delta_prototype =
          message_factory_->GetPrototype(observation_delta);
      if (root["datalog"] && root["datalog"]["fields"] &&
          root["datalog"]["fields"]["exclude"]) {
        for (const auto& f : root["datalog"]["fields"]["exclude"]) {
          auto field_name = f.as<std::string>();
          if (field_name.find(observation_delta->full_name()) == 0 &&
              field_name.size() > observation_delta->full_name().size() &&
              field_name[observation_delta->full_name().size()] == '.') {
            field_name =
                field_name.substr(observation_delta->full_name().size() + 1);
          } else {
            continue;
          }

          auto x = observation_delta->FindFieldByName(field_name);
          if (x) {
            actor_class.cleared_delta_fields.push_back(x);
          }
        }
      }
    } else {
      actor_class.observation_delta_prototype =
          actor_class.observation_space_prototype;
      actor_class.cleared_delta_fields = actor_class.cleared_observation_fields;
    }

    spdlog::info("clearing {} delta fields",
                 actor_class.cleared_delta_fields.size());

    auto action_space = importer_->pool()->FindMessageTypeByName(
        a_class["action"]["space"].as<std::string>());
    if (!action_space) {
      spdlog::error("Failed to lookup message type: {}",
                    a_class["action"]["space"].as<std::string>());
      throw std::runtime_error("init failure");
    }
    actor_class.action_space_prototype =
        message_factory_->GetPrototype(action_space);
  }
}

std::size_t Trial_spec::get_class_id(const std::string class_name) const {
  for (std::size_t i = 0; i < actor_classes.size(); ++i) {
    if (actor_classes[i].name == class_name) {
      return i;
    }
  }

  spdlog::error("trying to use unregistered actor class: {}", class_name);
  throw std::runtime_error("unknown actor class");
}
}  // namespace cogment
