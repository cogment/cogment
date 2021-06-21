// Copyright 2021 AI Redefined Inc. <dev+cogment@ai-r.com>
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
public:
  enum class InternalState { unknown, initializing, pending, running, terminating, ended };

  Trial(Orchestrator* orch, const std::string& user_id);
  ~Trial();

  Trial(Trial&&) = delete;
  Trial& operator=(Trial&&) = delete;
  Trial(const Trial&) = delete;
  Trial& operator=(const Trial&) = delete;

  InternalState state() const { return m_state; }
  const char* state_char() const;
  uint64_t tick_id() const { return m_tick_id; }
  uint64_t start_timestamp() const { return m_start_timestamp; }

  // Trial identification
  const std::string& id() const { return m_id; }
  const std::string& user_id() const { return m_user_id; }

  // Actors present in the trial
  const std::vector<std::unique_ptr<Actor>>& actors() const { return m_actors; }
  const std::unique_ptr<Actor>& actor(const std::string& name) const;

  // Initializes the trial
  void start(cogment::TrialParams params);
  const cogment::TrialParams& params() const { return m_params; }

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

  Orchestrator* m_orchestrator;

  std::mutex m_state_lock;
  std::mutex m_actor_lock;
  std::mutex m_sample_lock;
  std::mutex m_reward_lock;
  std::mutex m_sample_message_lock;
  std::mutex m_env_message_lock;
  std::shared_mutex m_terminating_lock;

  const std::string m_id;
  const std::string m_user_id;

  cogment::TrialParams m_params;

  std::shared_ptr<cogment::Stub_pool<cogment::EnvironmentEndpoint>::Entry> m_env_entry;

  InternalState m_state;
  uint64_t m_tick_id;
  const uint64_t m_start_timestamp;
  uint64_t m_end_timestamp;

  std::vector<std::unique_ptr<Actor>> m_actors;
  std::vector<Message> m_env_message_accumulator;
  std::unordered_map<std::string, uint32_t> m_actor_indexes;
  std::chrono::time_point<std::chrono::steady_clock> m_last_activity;

  std::vector<grpc_metadata> m_headers;
  easy_grpc::client::Call_options m_call_options;

  std::optional<::easy_grpc::Stream_promise<::cogment::EnvActionRequest>> m_outgoing_actions;

  std::uint32_t m_gathered_actions_count = 0;

  std::unique_ptr<TrialLogInterface> m_datalog_interface;
  std::deque<cogment::DatalogSample> m_step_data;
};

const char* get_trial_state_string(Trial::InternalState);
cogment::TrialState get_trial_api_state(Trial::InternalState);

}  // namespace cogment

#endif
