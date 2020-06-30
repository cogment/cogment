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

  static std::unique_ptr<Datalog_storage_interface> create(
      const std::string& spec, const YAML::Node& cfg);
};
}  // namespace cogment
#endif