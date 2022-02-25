#include "slt/settings.h"
#include "slt/settings_context.h"

#include <cstdlib>
#include <iomanip>
#include <unordered_map>
#include <iostream>
#include <vector>

namespace slt {
namespace {
slt::Setting request_help =
    slt::Setting_builder<bool>().with_default(false).with_arg("help").with_description("Shows this help");

Settings_context* active_context = nullptr;
using settings_store_t = std::vector<slt::Setting_base*>;
using arg_settings_store_t = std::unordered_map<std::string, slt::Setting_base*>;

// Prevent the initialization order fiasco.
settings_store_t& settings_store() {
  static settings_store_t instance;
  return instance;
}

arg_settings_store_t& arg_settings_store() {
  static arg_settings_store_t instance;
  return instance;
}

}  // namespace

void Setting_base::register_setting(Setting_base* setting) {
  assert(setting);
  settings_store().push_back(setting);

  if (setting->arg()) {
    auto& store = arg_settings_store();
    const auto& name = *setting->arg();

    // TODO: reject bad argument names
    if (name.find('=') != std::string_view::npos) {
      throw Settings_error(std::string("Illegal argument name: ") + name);
    }

    if (name.find("no_") == 0) {
      throw Settings_error(std::string("argument cannot start with \"no_\": ") + name);
    }
    if (store.find(name) != store.end()) {
      throw Settings_error(std::string("trying to register the same argument twice: ") + name);
    }

    store[name] = setting;
  }
}

Settings_context::Settings_context(std::string app_name, int argc, const char* argv[]) : app_name_(app_name) {
  assert(active_context == nullptr);
  active_context = this;

  // the order is important!
  load_from_env();
  load_from_args(argc, argv);

  if (help_requested()) {
    print_usage(false);
  }
}

void Settings_context::print_usage(bool to_stderr) const {
  auto& store = settings_store();

  std::ostream& dst = to_stderr ? std::cerr : std::cout;

  dst << "Usage: " << app_name_ << " [options]\n";
  dst << "Options:\n";
  for (auto& setting : store) {
    setting->print_help(dst);
  }
}

bool Settings_context::help_requested() const { return request_help.get(); }

void Settings_context::load_from_args(int argc, const char* argv[]) {
  auto& store = arg_settings_store();

  for (int i = 1; i < argc; ++i) {
    std::string_view arg = argv[i];

    // Ww only care about arguments prefixed with --
    if (arg.size() > 2 && arg[0] == '-' && arg[1] == '-') {
      arg = arg.substr(2);
      std::string_view name = arg;
      std::string_view val;

      auto eq_pos = arg.find('=');
      if (eq_pos != std::string_view::npos) {
        name = arg.substr(0, eq_pos);
        val = arg.substr(eq_pos + 1);
      }
      else {
        if (name.find("no_") == 0) {
          name = name.substr(3);
          val = "false";
        }
        else {
          val = "true";
        }
      }

      auto setting = store.find(std::string(name));
      if (setting == store.end()) {
        throw Settings_error("not a setting");
      }

      setting->second->set_from_string(val);
    }
  }
}

void Settings_context::load_from_env() {
  auto& store = settings_store();
  for (auto& setting : store) {
    if (setting->env_var()) {
      auto env_var_val = std::getenv(setting->env_var()->c_str());
      if (env_var_val) {
        setting->set_from_string(env_var_val);
      }
    }
  }
}

void Settings_context::validate_all() {
  bool all_pass = true;
  auto& store = settings_store();
  for (auto& setting : store) {
    if (!setting->is_valid()) {
      std::cerr << "Invalid setting value: " << setting->get_string() << "\n";
      setting->print_help(std::cerr);
      all_pass = false;
    }
  }

  if (!all_pass) {
    throw Settings_error("Invalid Setting value.");
  }
}

Settings_context::~Settings_context() {
  assert(active_context == this);
  active_context = nullptr;
  for (auto& setting : settings_store()) {
    setting->reset();
  }
}
}  // namespace slt
