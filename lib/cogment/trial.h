#ifndef COGMENT_ORCHESTRATOR_TRIAL_H
#define COGMENT_ORCHESTRATOR_TRIAL_H

#include "cogment/actor.h"
#include "cogment/utils.h"

#include "cogment/api/orchestrator.pb.h"

#include "uuid.h"

#include <chrono>
#include <mutex>
#include <string>
#include <vector>

namespace cogment {
class Orchestrator;

enum class Trial_state { initializing, pending, running, terminating, ended };

const char* get_trial_state_string(Trial_state s);
cogment::TrialState get_trial_state_proto(Trial_state s);

class Trial : public std::enable_shared_from_this<Trial> {
  static uuids::uuid_system_generator id_generator_;

 public:
  Trial(Orchestrator* orch, std::string user_id);
  ~Trial();

  Trial(Trial&&) = delete;
  Trial& operator=(Trial&&) = delete;
  Trial(const Trial&) = delete;
  Trial& operator=(const Trial&) = delete;

  std::lock_guard<std::mutex> lock();

  Trial_state state() const;

  // Trial identification
  const uuids::uuid& id() const;
  const std::string& user_id() const;

  // Actors present in the trial
  const std::vector<std::unique_ptr<Actor>>& actors() const;

  // Initializes the trial
  Future<void> configure(cogment::TrialParams params);
  const cogment::TrialParams& params() const;

  // Ends the trial. terminatr() will hold a shared_ptr to the trial until
  // termination is complete, so it's safe to let go of the trial once
  // this has been called.
  void terminate();

  // Primarily used to determine garbage collection elligibility.
  bool is_stale() const;

  // Marks the trial as being active.
  void refresh_activity();

 private:
  Orchestrator* orchestrator_;

  std::mutex lock_;

  // Identity
  uuids::uuid id_;
  std::string user_id_;

  // Configuration
  cogment::TrialParams params_;

  // State
  Trial_state state_;
  std::vector<std::unique_ptr<Actor>> actors_;
  std::chrono::time_point<std::chrono::steady_clock> last_activity_;
};

}  // namespace cogment

#endif