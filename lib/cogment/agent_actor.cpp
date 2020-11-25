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

#include "cogment/agent_actor.h"
#include "cogment/trial.h"

#include "spdlog/spdlog.h"

#define COGMENT_DEBUG_AGENTS
#ifdef COGMENT_DEBUG_AGENTS
  #define AGENT_DEBUG_LOG(...) spdlog::info(__VA_ARGS__)
#else
  #define AGENT_DEBUG_LOG(...)
#endif

namespace cogment {
Agent::Agent(Trial* owner, const std::string& in_actor_name, const ActorClass* actor_class, const std::string& impl,
             stub_type stub, std::optional<std::string> config_data)
    : Actor(owner, in_actor_name, actor_class),
      stub_(std::move(stub)),
      config_data_(std::move(config_data)),
      impl_(impl) {
  grpc_metadata trial_header;
  trial_header.key = grpc_slice_from_static_string("trial-id");
  trial_header.value = grpc_slice_from_copied_string(to_string(trial()->id()).c_str());

  grpc_metadata actor_header;
  actor_header.key = grpc_slice_from_static_string("actor-name");
  actor_header.value = grpc_slice_from_copied_string(actor_name().c_str());

  headers_ = {trial_header, actor_header};
  options_.headers = &headers_;

  AGENT_DEBUG_LOG("Agent(): {} {}", to_string(trial()->id()), actor_name());
}

Agent::~Agent() {
  AGENT_DEBUG_LOG("~Agent(): {} {}", to_string(trial()->id()), actor_name());

  if (outgoing_observations_) {
    outgoing_observations_->complete();
  }
}

Future<void> Agent::init() {
  AGENT_DEBUG_LOG("Agent::init(): {} {}", to_string(trial()->id()), actor_name());

  cogment::AgentStartRequest req;

  req.set_impl_name(impl_);

  if (config_data_) {
    req.mutable_config()->set_content(*config_data_);
  }

  const auto& trial_params = trial()->params();
  if (trial_params.has_trial_config()) {
    *req.mutable_trial_config() = trial_params.trial_config();
  }

  for (const auto& actor : trial()->actors()) {
    auto actor_in_trial = req.add_actors_in_trial();
    actor_in_trial->set_actor_class(actor->actor_class()->name);
    actor_in_trial->set_name(actor->actor_name());
  }

  return stub_->stub.OnStart(req, options_).then([this](auto rep) {
    (void)rep;
    (void)this;
    AGENT_DEBUG_LOG("Agent init start complete: {} {}", to_string(trial()->id()), actor_name());
  });
}

void Agent::dispatch_observation(const cogment::Observation& obs, bool end_of_trial) {
  if (!end_of_trial) {
    lazy_start_decision_stream();

    ::cogment::AgentObservationRequest req;
    *req.mutable_observation() = obs;
    outgoing_observations_->push(std::move(req));
  }
  else {
    ::cogment::AgentEndRequest req;
    auto new_obs = req.mutable_final_data()->add_observations();
    *new_obs = obs;

    stub_->stub.OnEnd(req, options_);
  }
}

void Agent::lazy_start_decision_stream() {
  if (!outgoing_observations_) {
    auto stream = stub_->stub.OnObservation(options_);

    outgoing_observations_ = std::move(std::get<0>(stream));
    auto incoming_actions = std::move(std::get<1>(stream));

    std::weak_ptr trial_weak = trial()->get_shared();
    auto name = actor_name();

    incoming_actions
        .for_each([trial_weak, name](auto act) {
          auto trial = trial_weak.lock();
          if (trial) {
            trial->actor_acted(name, act.action());
          }
        })
        .finally([](auto) {});
  }
}

bool Agent::is_active() const {
  // Actors driven by agent services are always active since
  // they are driven by the orchestrator itself.
  return true;
}

void Agent::dispatch_reward(int tick_id, const ::cogment::Reward& reward) {
  cogment::AgentRewardRequest req;
  req.set_tick_id(tick_id);
  req.mutable_reward()->CopyFrom(reward);

  stub_->stub.OnReward(req, options_).finally([](auto) {});
}

}  // namespace cogment
