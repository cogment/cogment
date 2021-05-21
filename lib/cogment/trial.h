// Copyright 2021 Artificial Intelligence Redefined <dev+cogment@ai-r.com>
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

#include <atomic>
#include <chrono>
#include <deque>
#include <mutex>
#include <shared_mutex>
#include <string>
#include <vector>

namespace cogment {
class Orchestrator;
class Client_actor;
class TrialLogInterface;

// TODO: Make Trial independent of orchestrator (to remove any chance of circular reference)
class Trial : public std::enable_shared_from_this<Trial> {
  static uuids::uuid_system_generator id_generator_;

  public:
  enum class InternalState { unknown, initializing, pending, running, terminating, ended };

  Trial(Orchestrator* orch, std::string user_id);
  ~Trial();

  Trial(Trial&&) = delete;
  Trial& operator=(Trial&&) = delete;
  Trial(const Trial&) = delete;
  Trial& operator=(const Trial&) = delete;

  InternalState state() const { return state_; }
  const char* state_char() const;
  uint64_t tick_id() const { return tick_id_; }
  uint64_t start_timestamp() const { return start_timestamp_; }

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

  void set_info(cogment::TrialInfo* info, bool with_observations, bool with_actors);

  private:
  void prepare_actors();
  cogment::EnvStartRequest prepare_environment();
  cogment::DatalogSample& make_new_sample();
  cogment::DatalogSample* get_last_sample();
  void flush_samples();
  void set_state(InternalState state);
  void advance_tick();
  void new_obs(ObservationSet&& new_obs);
  void next_step(EnvActionReply&& reply);
  void dispatch_observations();
  void cycle_buffer();
  void run_environment();
  cogment::EnvActionRequest make_action_request();
  void dispatch_env_messages();

  Orchestrator* orchestrator_;

  std::mutex state_lock_;
  std::mutex actor_lock_;
  std::mutex sample_lock_;
  std::mutex reward_lock_;
  std::mutex sample_message_lock_;
  std::mutex env_message_lock_;
  std::shared_mutex terminating_lock_;

  uuids::uuid id_;  // TODO: Store as string (since we convert everywhere back and forth)
  std::string user_id_;

  cogment::TrialParams params_;

  std::shared_ptr<cogment::Stub_pool<cogment::EnvironmentEndpoint>::Entry> env_stub_;

  InternalState state_;
  uint64_t tick_id_;
  const uint64_t start_timestamp_;
  uint64_t end_timestamp_;

  std::vector<std::unique_ptr<Actor>> actors_;
  std::vector<Message> env_message_accumulator_;
  std::unordered_map<std::string, uint32_t> actor_indexes_;
  std::chrono::time_point<std::chrono::steady_clock> last_activity_;

  std::vector<grpc_metadata> headers_;
  easy_grpc::client::Call_options call_options_;

  std::optional<::easy_grpc::Stream_promise<::cogment::EnvActionRequest>> outgoing_actions_;

  std::uint32_t gathered_actions_count_ = 0;

  std::unique_ptr<TrialLogInterface> datalog_interface_;
  std::deque<cogment::DatalogSample> step_data_;
};

const char* get_trial_state_string(Trial::InternalState);
cogment::TrialState get_trial_api_state(Trial::InternalState);

}  // namespace cogment

#endif
