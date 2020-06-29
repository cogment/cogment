#ifndef SLT_SETTINGS_SETTING_BUILDER_H
#define SLT_SETTINGS_SETTING_BUILDER_H

#include <optional>
#include <string>

namespace slt {
// This is meant to be used as a builder-pattern argument
// to the constructor of slt::Setting.
template <typename T>
struct Setting_builder {
  Setting_builder() {}

  Setting_builder& with_default(T v) {
    default_val = std::move(v);
    return *this;
  }

  Setting_builder& with_description(std::string v) {
    description = std::move(v);
    return *this;
  }

  Setting_builder& with_env_variable(std::string v) {
    env_var = std::move(v);
    return *this;
  }

  Setting_builder& with_arg(std::string v) {
    arg = std::move(v);
    return *this;
  }

  Setting_builder& with_validator(std::function<bool(const T&)> v) {
    validator = std::move(v);
    return *this;
  }

  T default_val;
  std::string description;

  std::optional<std::string> arg;
  std::optional<std::string> env_var;

  std::function<bool(const T&)> validator;
};
}  // namespace slt

#endif