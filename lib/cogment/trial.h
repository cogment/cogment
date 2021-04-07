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

#ifndef COGMENT_ORCHESTRATOR_TRIAL_H
#define COGMENT_ORCHESTRATOR_TRIAL_H

#include "cogment/actor.h"
#include "cogment/utils.h"

#include "cogment/api/datalog.pb.h"
#include "cogment/api/environment.egrpc.pb.h"
#include "cogment/api/orchestrator.pb.h"

#include "cogment/stub_pool.h"

#include "uuid.h"

#include <chrono>
#include <deque>
#include <mutex>
#include <string>
#include <vector>

namespace cogment {
class Orchestrator;
class Client_actor;
class TrialLogInterface;

enum class Trial_state { initializing, pending, running, terminating, ended };

const char* get_trial_state_string(Trial_state s);
cogment::TrialState get_trial_api_state(Trial_state s);

// TODO: Make Trial independent of orchestrator (to remove any chance of circular reference)
class Trial : public std::enable_shared_from_this<Trial> {
  static uuids::uuid_system_generator id_generator_;

  public:
  Trial(Orchestrator* orch, std::string user_id);
  ~Trial();

  Trial(Trial&&) = delete;
  Trial& operator=(Trial&&) = delete;
  Trial(const Trial&) = delete;
  Trial& operator=(const Trial&) = delete;

  std::lock_guard<std::mutex> lock() { return std::lock_guard(lock_); }

  Trial_state state() const { return state_; }
  uint64_t tick_id() const { return tick_id_; }

  // Trial identification
  const uuids::uuid& id() const { return id_; }
  const std::string& user_id() const { return user_id_; }

  // Actors present in the trial
  const std::vector<std::unique_ptr<Actor>>& actors() const { return actors_; }
  const std::unique_ptr<Actor>& actor(const std::string& name) const;

  // Initializes the trial
  void start(cogment::TrialParams params);
  const cogment::TrialParams& params() const { return params_; }

  // returns either a valid actor for the given request, or nullptr if none can be found
  Client_actor* get_join_candidate(const TrialJoinRequest& req);

  // Ends the trial. terminate() will hold a shared_ptr to the trial until
  // termination is complete, so it's safe to let go of the trial once
  // this has been called.
  void terminate();

  // Primarily used to determine garbage collection elligibility.
  bool is_stale() const;

  // Marks the trial as being active.
  void refresh_activity();

  void actor_acted(const std::string& actor_name, const cogment::Action& action);
  void reward_received(const cogment::Reward& reward, const std::string& source);
  void message_received(const cogment::Message& message, const std::string& source);
  std::shared_ptr<Trial> get_shared() { return shared_from_this(); }

  private:
  cogment::DatalogSample& make_new_sample();

  Orchestrator* orchestrator_;

  std::mutex lock_;
  std::mutex state_lock_;

  // Identity
  uuids::uuid id_;  // TODO: Store as string (since we convert everywhere back and forth)
  std::string user_id_;

  // Configuration
  cogment::TrialParams params_;

  // Connections
  std::shared_ptr<cogment::Stub_pool<cogment::EnvironmentEndpoint>::Entry> env_stub_;

  // State
  Trial_state state_;
  uint64_t tick_id_;
  ObservationSet observations_;
  void set_state(Trial_state state);
  void new_tick(ObservationSet&& new_obs, bool first_set = false);
  void next_step(EnvActionReply&& reply);

  std::vector<std::unique_ptr<Actor>> actors_;
  std::unordered_map<std::string, uint32_t> actor_indexes_;
  std::chrono::time_point<std::chrono::steady_clock> last_activity_;

  std::vector<grpc_metadata> headers_;
  easy_grpc::client::Call_options call_options_;

  void dispatch_observations();
  void cycle_buffer();
  void run_environment();
  cogment::EnvActionRequest make_action_request();

  std::optional<::easy_grpc::Stream_promise<::cogment::EnvActionRequest>> outgoing_actions_;

  std::vector<std::optional<cogment::Action>> actions_;
  std::uint32_t gathered_actions_count_ = 0;

  std::unique_ptr<TrialLogInterface> log_interface_;
  std::deque<cogment::DatalogSample> step_data_;
};

}  // namespace cogment

#endif
