#ifndef SLT_SETTINGS_SETTINGS_H
#define SLT_SETTINGS_SETTINGS_H

#include <cassert>
#include <functional>
#include <iostream>
#include <string>
#include <string_view>

#include <cstdlib>
#include <memory>
#include <sstream>

#include "slt/setting_builder.h"
#include "slt/settings_context.h"

#ifdef __GNUG__
#include <cxxabi.h>
#endif

namespace slt {
#ifdef __GNUG__
template <typename T>
struct type_name {
  static std::string get() {
    int status = 0;
    auto name = abi::__cxa_demangle(typeid(T).name(), nullptr, 0, &status);

    std::string result = name;
    free(name);
    return result;
  }
};

#else

template <typename T>
struct type_name {
  static std::string get() { return typeid(T).name(); }
};

#endif

template <class CharT, class Traits, class Allocator>
struct type_name<std::basic_string<CharT, Traits, Allocator>> {
  static std::string get() { return "string"; }
};

// Error Thrown when something goes wrong will seting up or processing Settings.
struct Settings_error : public std::runtime_error {
  Settings_error(std::string const& err) : std::runtime_error(err) {}
};

class Setting_base {
 public:
  template <typename T>
  Setting_base(Setting_builder<T>& builder)
      : description_(std::move(builder.description)),
        arg_(std::move(builder.arg)),
        env_var_(std::move(builder.env_var)) {}

  virtual ~Setting_base() {}

  std::string const& description() const { return description_; }
  std::optional<std::string> const& arg() const { return arg_; }
  std::optional<std::string> const& env_var() const { return env_var_; }

  virtual void print_help(std::ostream& dst) const = 0;
  virtual void set_from_string(std::string_view val) = 0;
  virtual void reset() = 0;
  virtual bool is_valid() const = 0;
  virtual std::string get_string() const = 0;

 protected:
  static void register_setting(Setting_base*);

 private:
  std::string description_;

  std::optional<std::string> arg_;
  std::optional<std::string> env_var_;
};

namespace detail {
template <typename T>
struct Setting_value_assign {
  static T assign(std::string_view val) {
    T result;
    std::string as_string(val);
    std::istringstream stream(as_string);
    stream >> result;
    return result;
  }

  static std::string to_string(const T& val) {
    std::ostringstream dst;
    dst << val;
    return dst.str();
  }
};

template <class CharT, class Traits, class Allocator>
struct Setting_value_assign<std::basic_string<CharT, Traits, Allocator>> {
  static std::basic_string<CharT, Traits, Allocator> assign(std::string_view val) {

    return std::basic_string<CharT, Traits, Allocator>(val);
  }

  static std::string to_string(const std::basic_string<CharT, Traits, Allocator>& val) {
    return "\"" + val + "\""; 
  }
};


template <typename T>
struct Setting_value_assign<std::vector<T>> {
  static std::vector<T> assign(std::string_view val) {
    std::vector<T> result;
    (void)val;
    assert(false);
    return result;
  }

  static std::string to_string(const bool& val) {
    (void)val;
    return "[]";
  }
};

template <>
struct Setting_value_assign<bool> {
  static bool assign(std::string_view val) {
    return val == "true" || val == "1";
  }

  static std::string to_string(const bool& val) {
    return val ? "true" : "false";
  }
};

template <typename Setting_T>
struct Setting_help_string {
  using value_type = typename Setting_T::value_type;

  static void print(const Setting_T& setting, std::ostream& dst) {
    dst << "  <" << type_name<value_type>::get()
        << "> : " << setting.description() << "\n";

    dst << "    Default:  " << Setting_value_assign<value_type>::to_string(setting.get_default()) << "\n";
    if (setting.arg()) {
      if constexpr (std::is_same_v<value_type, bool>) {
        dst << "    From Arg: --[no_]" << *setting.arg() << "\n";
      } else {
        dst << "    From Arg: --" << *setting.arg() << "=<value>\n";
      }
    }
    if (setting.env_var()) {
      dst << "    From env: " << *setting.env_var() << "\n";
    }
    dst << "\n";
  }
};

}  // namespace detail

// Settings are self-registering global properties that will be set when the slt
// Core is being initialized.
//
// Settings should always be defined as global variables.
// Settings will have their default until the core has been initialized, and
// after the core has shut down.
template <typename T>
class [[nodiscard]] Setting : public Setting_base {
 public:
  using value_type = T;

  Setting(Setting_builder<T> builder)
      : Setting_base(builder),
        default_(std::move(builder.default_val)),
        value_(default_),
        validator_(std::move(builder.validator)) {
    register_setting(this);
  }

  // Retrieve the value of the setting.
  T const& get() const { return value_; }

  // Sets the value of the setting.
  void set(T v) { 
    if(validator_ && !validator_(v)) {
      throw Settings_error("validation failure");
    }
    value_ = std::move(v); 
  }

  bool is_valid() const override {
    return validator_ == nullptr || validator_(value_);
  }

  void set_from_string(std::string_view val) override {
    set( detail::Setting_value_assign<T>::assign(val));
  }

  void print_help(std::ostream & dst) const override {
    detail::Setting_help_string<Setting<T>>::print(*this, dst);
  }

  std::string get_string() const override {
    return detail::Setting_value_assign<T>::to_string(value_);
  }

  // Resets the setting to whatever its default is.
  void reset() override { value_ = default_; }

  // Changes the default value of the setting. Be careful, this change does NOT
  // revert when resetting the setting.
  void set_default(T new_val) {
    default_ = std::move(new_val);
    value_ = default_;
  }

  const T & get_default() const {
    return default_;
  }
 private:
  T default_;
  T value_;
  std::function<bool(const T&)> validator_;
};

}  // namespace slt

#endif
