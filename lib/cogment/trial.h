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

#include "cogment/api/environment.grpc.pb.h"
#include "cogment/api/orchestrator.pb.h"
#include "cogment/api/datalog.pb.h"

#include <prometheus/summary.h>

#include "cogment/utils.h"

#include <atomic>
#include <memory>
#include <chrono>
#include <deque>
#include <mutex>
#include <shared_mutex>
#include <string>
#include <vector>

namespace cogment {
class Orchestrator;
class Environment;
class Actor;
class ClientActor;
class DatalogService;

// TODO: Make Trial independent of orchestrator (to remove any chance of circular reference)
class Trial : public std::enable_shared_from_this<Trial> {
public:
  enum class InternalState {
     unknown,
     initializing,
     pending,
     running,
     terminating,
     ended
  };

  struct Metrics {
    prometheus::Summary* trial_duration = nullptr;
    prometheus::Summary* tick_duration = nullptr;
  };

  Trial(Orchestrator* orch, std::unique_ptr<DatalogService> log, const std::string& user_id, const Metrics& met);
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

  ThreadPool& thread_pool();

  // Actors present in the trial
  const std::vector<std::unique_ptr<Actor>>& actors() const { return m_actors; }
  const std::unique_ptr<Actor>& actor(const std::string& name) const;

  // Initializes the trial
  void start(cogmentAPI::TrialParams&& params);
  const cogmentAPI::TrialParams& params() const { return m_params; }

  ClientActor* get_join_candidate(const std::string& actor_name, const std::string& actor_class) const;

  void request_end();
  void terminate(const std::string& details);

  // Primarily used to determine garbage collection elligibility.
  bool is_stale() const;

  // Marks the trial as being active.
  void refresh_activity();

  void env_observed(const std::string& env_name, cogmentAPI::ObservationSet&& obs, bool last);
  void actor_acted(const std::string& actor_name, const cogmentAPI::Action& action);
  void reward_received(const cogmentAPI::Reward& reward, const std::string& source);
  void message_received(const cogmentAPI::Message& message, const std::string& source);
  std::shared_ptr<Trial> get_shared() { return shared_from_this(); }

  void set_info(cogmentAPI::TrialInfo* info, bool with_observations, bool with_actors);

private:
  void prepare_actors();
  void prepare_environment();
  cogmentAPI::DatalogSample& make_new_sample();
  cogmentAPI::DatalogSample* get_last_sample();
  void flush_samples();
  void set_state(InternalState state);
  void advance_tick();
  void new_obs(cogmentAPI::ObservationSet&& new_obs);
  void dispatch_observations(bool last);
  void cycle_buffer();
  cogmentAPI::ActionSet make_action_set();
  void dispatch_env_messages();
  bool finalize_env();
  void finalize_actors();
  void finish();
  std::vector<Actor*> get_all_actors(const std::string& name);
  bool for_actors(const std::string& pattern, const std::function<void(Actor*)>& func);

  Orchestrator* m_orchestrator;
  Metrics m_metrics;
  uint64_t m_tick_start_timestamp;

  std::mutex m_state_lock;
  std::mutex m_sample_lock;
  std::mutex m_reward_lock;
  std::mutex m_sample_message_lock;
  std::shared_mutex m_terminating_lock;

  const std::string m_id;
  const std::string m_user_id;

  cogmentAPI::TrialParams m_params;

  InternalState m_state;
  bool m_env_last_obs;
  bool m_end_requested;
  uint64_t m_tick_id;
  const uint64_t m_start_timestamp;
  uint64_t m_end_timestamp;

  std::unique_ptr<Environment> m_env;
  std::vector<std::unique_ptr<Actor>> m_actors;
  std::unordered_map<std::string, uint32_t> m_actor_indexes;
  std::chrono::time_point<std::chrono::steady_clock> m_last_activity;

  std::atomic_uint m_gathered_actions_count;

  std::deque<cogmentAPI::DatalogSample> m_step_data;
  std::unique_ptr<DatalogService> m_datalog;
};

const char* get_trial_state_string(Trial::InternalState);
cogmentAPI::TrialState get_trial_api_state(Trial::InternalState);

}  // namespace cogment

#endif
