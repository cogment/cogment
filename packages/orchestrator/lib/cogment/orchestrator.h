// Copyright 2023 AI Redefined Inc. <dev+cogment@ai-r.com>
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

#include "cogment/client_actor.h"
#include "cogment/stub_pool.h"
#include "cogment/trial.h"
#include "cogment/trial_params.h"
#include "cogment/utils.h"
#include "cogment/directory.h"

#include "cogment/api/hooks.grpc.pb.h"
#include "cogment/api/datalog.grpc.pb.h"
#include "cogment/api/agent.grpc.pb.h"
#include "cogment/api/environment.grpc.pb.h"

#include "prometheus/registry.h"
#include "prometheus/summary.h"

#include <atomic>
#include <unordered_map>
#include <thread>

namespace cogment {

class Orchestrator {
public:
  using HandlerFunction = std::function<bool(const Trial& trial)>;

  Orchestrator(cogmentAPI::TrialParams default_trial_params, uint32_t gc_frequency,
               std::shared_ptr<grpc::ChannelCredentials> creds, prometheus::Registry* metrics_registry);
  ~Orchestrator();

  void Version(cogmentAPI::VersionInfo* out);

  void add_directory(std::string_view url, std::string_view auth_token);
  void add_prehook(const std::string& url);
  void register_to_directory(std::string_view host, uint16_t actor_port, uint16_t lifecycle_port,
                             const std::string& props);

  std::shared_ptr<Trial> start_trial(cogmentAPI::TrialParams&& params, const std::string& user_id,
                                     std::string trial_id_req, bool final_params);
  std::shared_ptr<Trial> get_trial(const std::string& trial_id) const;
  std::vector<std::shared_ptr<Trial>> all_trials() const;

  const Directory& directory() { return m_directory; }
  bool use_ssl() const { return m_channel_pool.is_ssl(); }
  StubPool<cogmentAPI::DatalogSP>* log_pool() { return &m_log_stubs; }
  StubPool<cogmentAPI::EnvironmentSP>* env_pool() { return &m_env_stubs; }
  StubPool<cogmentAPI::ServiceActorSP>* agent_pool() { return &m_agent_stubs; }
  ThreadPool& thread_pool() { return m_thread_pool; }
  Watchdog& watchdog() { return m_watchdog; }

  const cogmentAPI::TrialParams& default_trial_params() const { return m_default_trial_params; }

  std::future<void> watch_trials(HandlerFunction func);
  void notify_watchers(const Trial& trial);

private:
  struct Watcher {
    Watcher(HandlerFunction&& func) : handler(std::move(func)) {}
    HandlerFunction handler;
    std::promise<void> prom;
  };
  void m_perform_trial_gc();  // garbage collection
  cogmentAPI::TrialParams m_perform_pre_hooks(cogmentAPI::TrialParams&& params, const std::string& trial_id,
                                              const std::string& user_id);

  cogmentAPI::TrialParams m_default_trial_params;
  uint32_t m_gc_frequency;
  prometheus::Summary* m_trials_metrics;
  prometheus::Summary* m_ticks_metrics;
  prometheus::Summary* m_gc_metrics;

  mutable std::mutex m_trials_mutex;
  std::unordered_map<std::string, std::shared_ptr<Trial>> m_trials;

  Directory m_directory;

  // List of trial pre-hooks to invoke before actually launching trials
  using HookEntryType = std::shared_ptr<StubPool<cogmentAPI::TrialHooksSP>::Entry>;
  std::vector<HookEntryType> m_prehooks;

  ChannelPool m_channel_pool;

  StubPool<cogmentAPI::DirectorySP> m_directory_stubs;
  StubPool<cogmentAPI::TrialHooksSP> m_hook_stubs;
  StubPool<cogmentAPI::DatalogSP> m_log_stubs;
  StubPool<cogmentAPI::EnvironmentSP> m_env_stubs;
  StubPool<cogmentAPI::ServiceActorSP> m_agent_stubs;

  mutable std::mutex m_notification_lock;
  std::vector<Watcher> m_trial_watchers;

  std::atomic<int> m_gc_countdown;
  ThrQueue<std::shared_ptr<Trial>> m_trials_to_delete;
  std::future<void> m_delete_thread_fut;

  ThreadPool m_thread_pool;
  Watchdog m_watchdog;

  std::vector<Directory::RegisteredService> m_directory_registrations;
};

}  // namespace cogment

#endif
