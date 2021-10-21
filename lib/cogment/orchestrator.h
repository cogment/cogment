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

#include "cogment/api/hooks.grpc.pb.h"
#include "cogment/api/datalog.grpc.pb.h"
#include "cogment/client_actor.h"
#include "cogment/services/actor_service.h"
#include "cogment/services/trial_lifecycle_service.h"
#include "cogment/stub_pool.h"
#include "cogment/trial.h"
#include "cogment/trial_params.h"
#include "cogment/utils.h"

#include <prometheus/registry.h>
#include <prometheus/summary.h>

#include <atomic>
#include <unordered_map>
#include <thread>

namespace cogment {

class Orchestrator {
public:
  using HandlerFunction = std::function<void(const Trial& trial)>;

  Orchestrator(cogmentAPI::TrialParams default_trial_params, std::shared_ptr<grpc::ChannelCredentials> creds,
               prometheus::Registry* metrics_registry);
  ~Orchestrator();

  void Version(cogmentAPI::VersionInfo* out);

  // Initialization
  using HookEntryType = std::shared_ptr<StubPool<cogmentAPI::TrialHooksSP>::Entry>;
  void add_prehook(const HookEntryType& prehook);

  std::shared_ptr<Trial> start_trial(cogmentAPI::TrialParams params, const std::string& user_id,
                                     std::string trial_id_req);
  std::shared_ptr<Trial> get_trial(const std::string& trial_id) const;
  std::vector<std::shared_ptr<Trial>> all_trials() const;

  ActorService* actor_service() { return &m_actor_service; }
  TrialLifecycleService* trial_lifecycle_service() { return &m_trial_lifecycle_service; }

  ChannelPool* channel_pool() { return &m_channel_pool; }
  StubPool<cogmentAPI::LogExporterSP>* log_pool() { return &m_log_stubs; }
  StubPool<cogmentAPI::EnvironmentSP>* env_pool() { return &m_env_stubs; }
  StubPool<cogmentAPI::ServiceActorSP>* agent_pool() { return &m_agent_stubs; }
  ThreadPool& thread_pool() { return m_thread_pool; }

  const cogmentAPI::TrialParams& default_trial_params() const { return m_default_trial_params; }

  std::shared_future<void> watch_trials(HandlerFunction func);

  void notify_watchers(const Trial& trial);

private:
  void m_perform_garbage_collection();
  cogmentAPI::TrialParams m_perform_pre_hooks(cogmentAPI::TrialParams&& params, const std::string& trial_id,
                                              const std::string& user_id);

  // Configuration
  cogmentAPI::TrialParams m_default_trial_params;
  prometheus::Summary* m_trials_metrics;
  prometheus::Summary* m_ticks_metrics;
  prometheus::Summary* m_gc_metrics;

  // Currently existing Trials
  mutable std::mutex m_trials_mutex;
  std::unordered_map<std::string, std::shared_ptr<Trial>> m_trials;

  // List of trial pre-hooks to invoke before actually launching trials
  std::vector<HookEntryType> m_prehooks;

  ChannelPool m_channel_pool;

  StubPool<cogmentAPI::LogExporterSP> m_log_stubs;
  StubPool<cogmentAPI::EnvironmentSP> m_env_stubs;
  StubPool<cogmentAPI::ServiceActorSP> m_agent_stubs;

  ActorService m_actor_service;
  TrialLifecycleService m_trial_lifecycle_service;

  mutable std::mutex m_notification_lock;
  std::vector<HandlerFunction> m_trial_watchers;

  std::atomic<int> m_garbage_collection_countdown;
  ThrQueue<std::shared_ptr<Trial>> m_trials_to_delete;

  std::promise<void> m_watchtrial_prom;
  std::shared_future<void> m_watchtrial_fut;

  ThreadPool m_thread_pool;
};

}  // namespace cogment

#endif
