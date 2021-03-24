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

#include "cogment/datalog/storage_interface.h"
#include "cogment/config_file.h"
#include "cogment/datalog/grpc_exporter.h"

#include "slt/settings.h"

#include <algorithm>
#include <filesystem>

namespace fs = std::filesystem;

namespace settings {
slt::Setting datalog_filename = slt::Setting_builder<std::string>()
                                    .with_default("datalog")
                                    .with_description("When using \"csv, or raw\" storage, "
                                                      "the file where the log is saved.")
                                    .with_env_variable("DATALOG_FILE")
                                    .with_arg("datalog_file");

slt::Setting datalog_flush_frequency =
    slt::Setting_builder<int>()
        .with_default(-1)
        .with_description("if >= 0 flush trial data when N samples have been collected "
                          "(flush always happens on tiral end)")
        .with_env_variable("DATA_LOG_FLUSH_FREQUENCY");
}  // namespace settings

namespace {
class Noop_trial_log_interface : public cogment::TrialLogInterface {
  public:
  void add_sample(cogment::DatalogSample) override {}
};

class Noop_datalog_storage : public cogment::Datalog_storage_interface {
  std::unique_ptr<cogment::TrialLogInterface> start_log(const cogment::Trial*) override {
    return std::make_unique<Noop_trial_log_interface>();
  }
};
}  // namespace

namespace cogment {
std::unique_ptr<Datalog_storage_interface> Datalog_storage_interface::create(const std::string& spec,
                                                                             const YAML::Node& cfg) {
  std::string spec_l = spec;
  std::transform(spec_l.begin(), spec_l.end(), spec_l.begin(), ::tolower);

  if (spec_l == "none") {
    return std::make_unique<Noop_datalog_storage>();
  }

  if (spec_l == "grpc") {
    return std::make_unique<Grpc_datalog_exporter>(cfg[cfg_file::datalog_key][cfg_file::d_url_key].as<std::string>());
  }

  throw std::runtime_error("invalid datalog specification.");
}

}  // namespace cogment