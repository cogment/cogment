#ifndef AOM_ORCHESTRATOR_TRIAL_H
#define AOM_ORCHESTRATOR_TRIAL_H

#include "easy_grpc/easy_grpc.h"

#include "cogment/api/agent.pb.h"
#include "cogment/api/data.pb.h"
#include "cogment/api/environment.egrpc.pb.h"
#include "cogment/api/environment.pb.h"
#include "cogment/api/orchestrator.pb.h"

#include "aom/actor.h"
#include "aom/datalog/storage_interface.h"
#include "aom/stub_pool.h"
#include "aom/utils.h"

#include "uuid.h"

#include <chrono>
#include <deque>
#include <memory>
#include <optional>

namespace cogment {
class Trial_action_handler;
class Orchestrator;

enum class Trial_state {
  Initializing,
  Ready,
  Busy,
  Terminating,
};

class Trial : public std::enable_shared_from_this<Trial> {
  static uuids::uuid_system_generator id_generator_;

 public:
  Trial(Orchestrator* owner, std::string user_id);
  ~Trial();
  Trial(Trial&&) = default;
  Trial& operator=(Trial&&) = default;

  // The id of the trial itself
  const uuids::uuid& id() { return id_; }

  // The actors present in the trial
  const std::vector<std::unique_ptr<Actor>>& actors() const { return actors_; }

  // The actor id of the human actor.
  int human_actor_id() const;

  ::easy_grpc::Future<void> configure(cogment::TrialParams);

  ::easy_grpc::Future<::cogment::TrialActionReply> user_acted(cogment::TrialActionRequest req);

  void begin();
  void terminate();

  Trial_state state() const { return state_; }

  void mark_busy() { state_ = Trial_state::Busy; }
  void mark_ready() { state_ = Trial_state::Ready; }
  void mark_terminating() { state_ = Trial_state::Terminating; }

  void consume_feedback(const ::google::protobuf::RepeatedPtrField<::cogment::Feedback>&);

  void populate_observation(int actor_id, ::cogment::Observation* obs);

  void heartbeat();

  bool is_stale();

  const std::vector<int>& actor_counts() const { return actor_counts_; }

  const cogment::TrialParams& params() const { return params_; }

 private:
  void prepare_actions();
  void gather_actions();
  void send_final_observation();
  void dispatch_update();

  cogment::TrialParams params_;

  void refresh_activity();

  std::chrono::time_point<std::chrono::steady_clock> last_activity_;

  void strip_observation(cogment::ObservationSet& observation);

  uuids::uuid id_;
  std::string user_id_;

  Orchestrator* owner_;
  std::vector<std::unique_ptr<Actor>> actors_;
  std::vector<int> actor_counts_;
  std::atomic<Trial_state> state_ = Trial_state::Initializing;

  cogment::ObservationSet latest_observation_;
  std::atomic<int> pending_decisions_;
  std::vector<expected<cogment::Action>> actions_;

  std::uint32_t trial_steps_ = 0;

  std::shared_ptr<Stub_pool<cogment::Environment>::Entry> env_stub_;

  std::unique_ptr<Trial_log_interface> log_interface_;
  std::vector<cogment::DatalogSample> data_;
  Trial(const Trial&) = delete;
  Trial& operator=(const Trial&) = delete;

  std::vector<grpc_metadata> headers_;
  easy_grpc::client::Call_options call_options_;
};
}  // namespace cogment
#endif