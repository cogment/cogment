#ifndef AOM_ORCHESTRATOR_AGENT_H
#define AOM_ORCHESTRATOR_AGENT_H

#include "cogment/actor.h"
#include "cogment/api/agent.egrpc.pb.h"
#include "cogment/stub_pool.h"

#include <optional>

namespace cogment {

class Trial;
class Agent : public Actor {
  public:
  using stub_type = std::shared_ptr<Stub_pool<cogment::AgentEndpoint>::Entry>;
  Agent(Trial* owner, std::uint32_t actor_id, const ActorClass* actor_class, stub_type stub,
        std::optional<std::string> config_data);

  ~Agent();

  Future<void> init() override;
  void terminate() override;

  void dispatch_reward(int tick_id, const ::cogment::Reward& reward) override;
  Future<cogment::Action> request_decision(cogment::Observation&& obs) override;

  ::easy_grpc::Future<::cogment::TrialActionReply> user_acted(cogment::TrialActionRequest req) override {
    throw std::runtime_error("agent is recieving human action.");
  }

  private:
  stub_type stub_;
  std::vector<grpc_metadata> headers_;

  cogment::Action latest_action_;
  std::optional<std::string> config_data_;
};
}  // namespace cogment
#endif