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

#ifndef COGMENT_ORCHESTRATOR_ORCHESTRATOR_H
#define COGMENT_ORCHESTRATOR_ORCHESTRATOR_H

#include "cogment/api/hooks.egrpc.pb.h"
#include "cogment/datalog/storage_interface.h"
#include "cogment/services/actor_service.h"
#include "cogment/services/trial_lifecycle_service.h"
#include "cogment/stub_pool.h"
#include "cogment/trial.h"
#include "cogment/trial_params.h"
#include "cogment/trial_spec.h"
#include "cogment/utils.h"

#include <prometheus/registry.h>
#include <prometheus/summary.h>

#include <atomic>
#include <unordered_map>

namespace cogment {
class Orchestrator {
public:
  using HandlerFunction = std::function<void(const Trial& trial)>;

  Orchestrator(Trial_spec trial_spec, cogment::TrialParams default_trial_params,
               std::shared_ptr<easy_grpc::client::Credentials> creds, prometheus::Registry* metrics_registry);
  ~Orchestrator();

  // Initialization
  using HookEntryType = std::shared_ptr<Stub_pool<cogment::TrialHooks>::Entry>;
  void add_prehook(const HookEntryType& prehook);
  void set_log_exporter(std::unique_ptr<DatalogStorageInterface> le);

  // Lifecycle
  aom::Future<std::shared_ptr<Trial>> start_trial(cogment::TrialParams params, const std::string& user_id);

  // Client API
  TrialJoinReply client_joined(TrialJoinRequest);
  ::easy_grpc::Stream_future<::cogment::TrialActionReply> bind_client(
      const std::string& trial_id, const std::string& actor_name,
      ::easy_grpc::Stream_future<::cogment::TrialActionRequest> actions);

  // Services
  ActorService& actor_service() { return m_actor_service; }
  TrialLifecycleService& trial_lifecycle_service() { return m_trial_lifecycle_service; }

  // Lookups
  std::shared_ptr<Trial> get_trial(const std::string& trial_id) const;

  // Gets all running trials.
  std::vector<std::shared_ptr<Trial>> all_trials() const;

  // Semi-internal, rpc management related.
  easy_grpc::Completion_queue* client_queue() { return &m_client_queue; }
  Channel_pool* channel_pool() { return &m_channel_pool; }
  Stub_pool<cogment::EnvironmentEndpoint>* env_pool() { return &m_env_stubs; }
  Stub_pool<cogment::AgentEndpoint>* agent_pool() { return &m_agent_stubs; }

  const cogment::TrialParams& default_trial_params() const { return m_default_trial_params; }

  const Trial_spec& get_trial_spec() const { return m_trial_spec; }

  std::unique_ptr<TrialLogInterface> start_log(const Trial* trial) { return m_log_exporter->start_log(trial); }

  void watch_trials(HandlerFunction func);

  void notify_watchers(const Trial& trial);

private:
  // Configuration
  Trial_spec m_trial_spec;
  cogment::TrialParams m_default_trial_params;
  prometheus::Summary* m_trials_metrics;
  prometheus::Summary* m_ticks_metrics;
  prometheus::Summary* m_gc_metrics;

  // Currently existing Trials
  mutable std::mutex m_trials_mutex;
  std::unordered_map<std::string, std::shared_ptr<Trial>> m_trials;

  // List of trial pre-hooks to invoke before actually launching trials
  std::vector<HookEntryType> m_prehooks;

  // Send trial data to this destination.
  std::unique_ptr<DatalogStorageInterface> m_log_exporter;

  // Completion queue for handling requests returning from env/agent/hooks
  easy_grpc::Completion_queue m_client_queue;
  Channel_pool m_channel_pool;

  Stub_pool<cogment::EnvironmentEndpoint> m_env_stubs;
  Stub_pool<cogment::AgentEndpoint> m_agent_stubs;

  ActorService m_actor_service;
  TrialLifecycleService m_trial_lifecycle_service;

  mutable std::mutex m_notification_lock;
  std::vector<HandlerFunction> m_trial_watchers;

  std::atomic<int> m_garbage_collection_countdown;
  void m_perform_garbage_collection();

  aom::Future<cogment::PreTrialContext> m_perform_pre_hooks(cogment::PreTrialContext ctx, const std::string& trial_id);
};
}  // namespace cogment
#endif