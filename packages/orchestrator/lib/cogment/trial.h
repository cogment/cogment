// Copyright 2022 AI Redefined Inc. <dev+cogment@ai-r.com>
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

#include "cogment/utils.h"

#include "cogment/api/orchestrator.pb.h"
#include "cogment/api/common.pb.h"
#include "cogment/api/datalog.pb.h"

#include "prometheus/summary.h"

#include <atomic>
#include <memory>
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
  enum class InternalState { unknown, initializing, pending, running, terminating, ended };
  struct Metrics {
    prometheus::Summary* trial_duration = nullptr;
    prometheus::Summary* tick_duration = nullptr;
  };

  static std::shared_ptr<Trial> make(Orchestrator* orch, const std::string& user_id, const std::string& id,
                                     const Metrics& met) {
    return std::shared_ptr<Trial>(new Trial(orch, user_id, id, met));
  }
  ~Trial();

  Trial(Trial&&) = delete;
  Trial& operator=(Trial&&) = delete;
  Trial(const Trial&) = delete;
  Trial& operator=(const Trial&) = delete;

  const std::string& id() const { return m_id; }
  const std::string& user_id() const { return m_user_id; }
  const std::string& env_name() const;
  ThreadPool& thread_pool();
  const cogmentAPI::TrialParams& params() const { return m_params; }

  InternalState state() const { return m_state; }
  uint64_t tick_id() const { return m_tick_id; }

  const std::vector<std::unique_ptr<Actor>>& actors() const { return m_actors; }

  void start(cogmentAPI::TrialParams&& params);

  ClientActor* get_join_candidate(const std::string& actor_name, const std::string& actor_class) const;

  void request_end();
  void terminate(const std::string& details);

  bool is_stale();

  void env_observed(const std::string& env_name, cogmentAPI::ObservationSet&& obs, bool last);
  void actor_acted(const std::string& actor_name, cogmentAPI::Action&& action);
  void reward_received(const std::string& source, cogmentAPI::Reward&& reward);
  void message_received(const std::string& source, cogmentAPI::Message&& message);

  void set_info(cogmentAPI::TrialInfo* info, bool with_observations, bool with_actors);

private:
  Trial(Orchestrator* orch, const std::string& user_id, const std::string& id, const Metrics& met);
  void refresh_activity();
  void prepare_actors();
  void prepare_environment();
  void prepare_datalog();
  void wait_for_actors();
  cogmentAPI::DatalogSample& make_new_sample();
  cogmentAPI::DatalogSample* get_last_sample();
  void flush_samples();
  void set_state(InternalState state);
  void advance_tick();
  void new_obs(cogmentAPI::ObservationSet&& new_obs);
  void new_special_event(std::string_view desc);
  void dispatch_observations(bool last);
  void cycle_buffer();
  cogmentAPI::ActionSet make_action_set();
  void dispatch_env_messages();
  bool finalize_env();
  void finalize_actors();
  void finish();
  std::vector<Actor*> get_all_actors(const std::string& name);
  bool for_actors(const std::string& pattern, const std::function<void(Actor*)>& func);

  const std::string m_id;
  const std::string m_user_id;
  const uint64_t m_start_timestamp;
  uint64_t m_end_timestamp;
  cogmentAPI::TrialParams m_params;
  Orchestrator* m_orchestrator;
  Metrics m_metrics;

  std::mutex m_state_lock;
  std::mutex m_sample_lock;
  std::mutex m_reward_lock;
  std::mutex m_sample_message_lock;
  std::shared_mutex m_terminating_lock;

  InternalState m_state;
  bool m_env_last_obs;
  bool m_end_requested;
  uint64_t m_tick_id;
  uint64_t m_tick_start_timestamp;
  std::atomic_uint m_nb_actors_acted;
  size_t m_nb_available_actors;
  uint64_t m_max_steps;
  uint64_t m_max_inactivity;

  std::unique_ptr<Environment> m_env;
  std::vector<std::unique_ptr<Actor>> m_actors;
  std::unordered_map<std::string, uint32_t> m_actor_indexes;
  uint64_t m_last_activity;

  std::deque<cogmentAPI::DatalogSample> m_step_data;
  std::unique_ptr<DatalogService> m_datalog;
};

const char* get_trial_state_string(Trial::InternalState);
cogmentAPI::TrialState get_trial_api_state(Trial::InternalState);

}  // namespace cogment

#endif
