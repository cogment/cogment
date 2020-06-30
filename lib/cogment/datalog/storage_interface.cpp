#include "cogment/datalog/storage_interface.h"
#include "cogment/datalog/grpc_exporter.h"

#include "slt/settings.h"

#include <algorithm>
#include <filesystem>

namespace fs = std::filesystem;

namespace settings {
slt::Setting datalog_filename = slt::Setting_builder<std::string>()
                                    .with_default("datalog")
                                    .with_description(
                                        "When using \"csv, or raw\" storage, "
                                        "the file where the log is saved.")
                                    .with_env_variable("DATALOG_FILE")
                                    .with_arg("datalog_file");

slt::Setting datalog_flush_frequency =
    slt::Setting_builder<int>()
        .with_default(-1)
        .with_description(
            "if >= 0 flush trial data when N samples have been collected "
            "(flush always happens on tiral end)")
        .with_env_variable("DATA_LOG_FLUSH_FREQUENCY");
}  // namespace settings

namespace {
class Noop_trial_log_interface : public cogment::Trial_log_interface {
 public:
  void add_sample(cogment::DatalogSample) override {}
};

class Noop_datalog_storage : public cogment::Datalog_storage_interface {
  std::unique_ptr<cogment::Trial_log_interface> begin_trial(
      cogment::Trial*) override {
    return std::make_unique<Noop_trial_log_interface>();
  }
};
}  // namespace

namespace cogment {
std::unique_ptr<Datalog_storage_interface> Datalog_storage_interface::create(
    const std::string& spec, const YAML::Node& cfg) {
  std::string spec_l = spec;
  std::transform(spec_l.begin(), spec_l.end(), spec_l.begin(), ::tolower);

  if (spec_l == "none") {
    return std::make_unique<Noop_datalog_storage>();
  }

  if (spec_l == "grpc") {
    return std::make_unique<Grpc_datalog_exporter>(
        cfg["datalog"]["url"].as<std::string>());
  }

  throw std::runtime_error("invalid datalog specification.");
}

}  // namespace cogment