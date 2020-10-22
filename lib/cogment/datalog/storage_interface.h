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

#ifndef AOM_DATALOG_STORAGE_INTERFACE_H
#define AOM_DATALOG_STORAGE_INTERFACE_H

#include "cogment/api/data.pb.h"
#include "uuid.h"

#include <fstream>
#include <memory>
#include "yaml-cpp/yaml.h"

namespace cogment {

class Trial;
// Per trial datalog interface.
class Trial_log_interface {
  public:
  virtual ~Trial_log_interface() {}

  virtual void add_sample(cogment::DatalogSample data) = 0;

  virtual void add_samples(std::vector<cogment::DatalogSample>&& data) {
    for (auto i = data.begin(); i != data.end(); ++i) {
      add_sample(std::move(*i));
    }
  }
};

// Orchestrator-wide datalog interface.
class Datalog_storage_interface {
  public:
  virtual ~Datalog_storage_interface() {}

  virtual std::unique_ptr<Trial_log_interface> begin_trial(Trial* trial) = 0;

  static std::unique_ptr<Datalog_storage_interface> create(const std::string& spec, const YAML::Node& cfg);
};
}  // namespace cogment
#endif