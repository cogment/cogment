#ifndef AOM_ORCHESTRATOR_TRIAL_PARAMS_H
#define AOM_ORCHESTRATOR_TRIAL_PARAMS_H

#include "cogment/api/common.pb.h"
#include "yaml-cpp/yaml.h"

namespace cogment {
// This expects the `trial_params` root node of cogment.yaml,
// and generates a TrialParams.
struct Trial_spec;
cogment::TrialParams load_params(const YAML::Node& yaml, const Trial_spec& spec);
}  // namespace cogment

#endif