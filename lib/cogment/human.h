#ifndef AOM_ORCHESTRATOR_HUMAN_H
#define AOM_ORCHESTRATOR_HUMAN_H

#include "aom/actor.h"

namespace cogment {

class Human : public Actor {
 public:
  Human(std::string tid);
  ~Human();

  Future<void> init() override;
  void terminate() override;
  void send_final_observation(cogment::Observation&& obs) override;

  bool is_human() const override { return true; }

  void dispatch_reward(int tick_id, const ::cogment::Reward& reward) override {
    latest_reward_ = reward;
  }

  Future<cogment::Action> request_decision(cogment::Observation&& obs) override;
  ::easy_grpc::Future<::cogment::TrialActionReply> user_acted(
      cogment::TrialActionRequest req) override;

 private:
  std::optional<::cogment::Reward> latest_reward_;
  Promise<cogment::Action> human_action_promise_;
  Promise<::cogment::TrialActionReply> human_observation_promise_;
};
}  // namespace cogment
#endif