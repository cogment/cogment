
#include "cogment/agent.h"
#include "cogment/trial.h"

#include "spdlog/spdlog.h"

#define COGMENT_DEBUG_AGENTS
#ifdef COGMENT_DEBUG_AGENTS
  #define AGENT_DEBUG_LOG(...) spdlog::info(__VA_ARGS__)
#else
  #define AGENT_DEBUG_LOG(...)
#endif

namespace cogment {
Agent::Agent(Trial* owner, std::uint32_t in_actor_id, const ActorClass* actor_class, stub_type stub,
             std::optional<std::string> config_data)
    : Actor(owner, in_actor_id, actor_class), stub_(std::move(stub)), config_data_(std::move(config_data)) {
  grpc_metadata trial_header;
  trial_header.key = grpc_slice_from_static_string("trial_id");
  trial_header.value = grpc_slice_from_copied_string(to_string(trial()->id()).c_str());

  grpc_metadata actor_header;
  actor_header.key = grpc_slice_from_static_string("actor_id");
  actor_header.value = grpc_slice_from_copied_string(std::to_string(actor_id()).c_str());

  headers_ = {trial_header, actor_header};

  AGENT_DEBUG_LOG("Agent(): {} {}", to_string(trial()->id()), actor_id());
}

Agent::~Agent() { AGENT_DEBUG_LOG("~Agent(): {} {}", to_string(trial()->id()), actor_id()); }

Future<void> Agent::init() {
  AGENT_DEBUG_LOG("Agent::init(): {} {}", to_string(trial()->id()), actor_id());

  cogment::AgentStartRequest req;

  if (config_data_) {
    req.mutable_config()->set_content(*config_data_);
  }

  const auto& trial_params = trial()->params();
  if (trial_params.has_trial_config()) {
    *req.mutable_trial_config() = trial_params.trial_config();
  }

  for (const auto& actor : trial()->actors()) {
    req.add_actor_class_idx(actor->actor_class()->index);
  }

  return stub_->stub.Start(req).then([this](auto rep) {
    (void)rep;
    (void)this;
    AGENT_DEBUG_LOG("Agent init start complete: {} {}", to_string(trial()->id()), actor_id());
  });
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

Future<cogment::Action> Agent::request_decision(cogment::Observation&& obs) {
  cogment::AgentDecideRequest req;
  *req.mutable_observation() = std::move(obs);

  easy_grpc::client::Call_options options;
  options.headers = &headers_;

  return stub_->stub.Decide(req, options).then([](auto rep) { return rep.action(); });
}
}  // namespace cogment