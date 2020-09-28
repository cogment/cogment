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
Agent::Agent(Trial* owner, std::uint32_t in_actor_id, const ActorClass* actor_class, const std::string& impl,
             stub_type stub, std::optional<std::string> config_data)
    : Actor(owner, in_actor_id, actor_class),
      stub_(std::move(stub)),
      config_data_(std::move(config_data)),
      impl_(impl) {
  grpc_metadata trial_header;
  trial_header.key = grpc_slice_from_static_string("trial-id");
  trial_header.value = grpc_slice_from_copied_string(to_string(trial()->id()).c_str());

  grpc_metadata actor_header;
  actor_header.key = grpc_slice_from_static_string("actor-id");
  actor_header.value = grpc_slice_from_copied_string(std::to_string(actor_id()).c_str());

  headers_ = {trial_header, actor_header};

  AGENT_DEBUG_LOG("Agent(): {} {}", to_string(trial()->id()), actor_id());
}

Agent::~Agent() { AGENT_DEBUG_LOG("~Agent(): {} {}", to_string(trial()->id()), actor_id()); }

Agent::~Agent() { AGENT_DEBUG_LOG("~Agent(): {} {}", to_string(trial()->id()), actor_id()); }

Future<void> Agent::init() {
  AGENT_DEBUG_LOG("Agent::init(): {} {}", to_string(trial()->id()), actor_id());

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
    actor_in_trial->set_name(actor->name());
  }
  easy_grpc::client::Call_options options;
  options.headers = &headers_;

  return stub_->stub.Start(req, options).then([this](auto rep) {
    (void)rep;
    (void)this;
    AGENT_DEBUG_LOG("Agent init start complete: {} {}", to_string(trial()->id()), actor_id());
  });
}

void Agent::dispatch_observation(const cogment::Observation& obs) {
  spdlog::info("pushing observation??");
  lazy_start_decision_stream();

  spdlog::info("pushing observation");
  ::cogment::AgentDataRequest req;
  *req.mutable_observation() = obs;
  outgoing_observations_->push(std::move(req));
  spdlog::info("observation pushed...");
}

void Agent::lazy_start_decision_stream() {
  if (!outgoing_observations_) {
    easy_grpc::client::Call_options options;
    options.headers = &headers_;

    auto stream = stub_->stub.Decide(options);

    outgoing_observations_ = std::move(std::get<0>(stream));
    auto incoming_actions = std::move(std::get<1>(stream));
    incoming_actions.for_each([this](auto act) { trial()->actor_acted(actor_id(), act.action()); })
        .finally([this](auto) { outgoing_observations_ = std::nullopt; });
  }
}

void Agent::terminate() {
  AGENT_DEBUG_LOG("Agent::terminate: {} {}", to_string(trial()->id()), actor_id());
  cogment::AgentEndRequest req;

  easy_grpc::client::Call_options options;
  options.headers = &headers_;

  stub_->stub.End(req, options).finally([](auto) {});
}

void Agent::dispatch_reward(int tick_id, const ::cogment::Reward& reward) {
  cogment::AgentRewardRequest req;
  req.set_tick_id(tick_id);
  req.mutable_reward()->CopyFrom(reward);

  easy_grpc::client::Call_options options;
  options.headers = &headers_;

  stub_->stub.Reward(req, options).finally([](auto) {});
}

}  // namespace cogment
