
#include "aom/agent.h"
#include "aom/trial.h"

#include "spdlog/spdlog.h"

namespace cogment {
Agent::Agent(Trial* owner, stub_type stub, std::optional<std::string> config_data)
    : Actor(to_string(owner->id())), owner_(owner), stub_(std::move(stub)), config_data_(std::move(config_data)) {}

Agent::~Agent() { SPDLOG_TRACE("Agent->~"); }

Future<void> Agent::init() {
  cogment::AgentStartRequest req;
  req.set_trial_id(trial_id());
  req.set_actor_id(actor_id());
  if (config_data_) {
    req.mutable_config()->set_content(*config_data_);
  }

  const auto& trial_params = owner_->params();
  if (trial_params.has_trial_config()) {
    *req.mutable_trial_config() = trial_params.trial_config();
  }

  for (auto i : owner_->actor_counts()) {
    req.add_actor_counts(i);
  }
  SPDLOG_TRACE("Agent->Start");
  return stub_->stub.Start(req).then([](auto rep) { SPDLOG_TRACE("Agent->Started"); });
}

void Agent::terminate() {
  cogment::AgentEndRequest req;
  req.set_trial_id(trial_id());
  req.set_actor_id(actor_id());

  SPDLOG_TRACE("Agent->End");
  stub_->stub.End(req).finally([](auto) { SPDLOG_TRACE("Agent->Ended"); });
}

void Agent::dispatch_reward(int tick_id, const ::cogment::Reward& reward) {
  cogment::AgentRewardRequest req;
  req.set_trial_id(trial_id());
  req.set_actor_id(actor_id());
  req.set_tick_id(tick_id);

  req.mutable_reward()->CopyFrom(reward);
  stub_->stub.Reward(req).finally([](auto) {});
}

Future<cogment::Action> Agent::request_decision(cogment::Observation&& obs) {
  cogment::AgentDecideRequest req;
  req.set_trial_id(trial_id());
  req.set_actor_id(actor_id());

  *req.mutable_observation() = std::move(obs);

  SPDLOG_TRACE("Agent->Decide");
  return stub_->stub.Decide(req).then([this](auto rep) {
    SPDLOG_TRACE("Agent->Decided");
    owner_->consume_feedback(rep.feedbacks());

    return rep.action();
  });
}
}  // namespace cogment