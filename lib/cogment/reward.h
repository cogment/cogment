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

namespace cogment {

// Collapses a collection of feedback into a reward.
//
// This is a template so that we can do it on either a std::vector<>
// Or a regular protobuf collection
template <typename contT>
cogment::Reward build_reward(const contT& reward_source_container) {
  cogment::Reward result;

  float value_accum = 0.0f;
  float confidence_accum = 0.0f;

  for (const auto& reward_source : reward_source_container) {
    auto fb_conf = reward_source.confidence();

    if (fb_conf > 0.0f) {
      value_accum += reward_source.value() * fb_conf;
      confidence_accum += fb_conf;
    }

    result.add_sources()->CopyFrom(reward_source);
  }

  if (confidence_accum > 0.0f) {
    value_accum /= confidence_accum;
  }

  result.set_value(value_accum);
  return result;
}
}  // namespace cogment