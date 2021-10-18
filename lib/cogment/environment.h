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

#ifndef AIR_ORCHESTRATOR_ENVIRONMENT_H
#define AIR_ORCHESTRATOR_ENVIRONMENT_H

#include "cogment/api/environment.grpc.pb.h"
#include "cogment/api/orchestrator.pb.h"

#include "cogment/stub_pool.h"

#include "grpc++/grpc++.h"

#include <vector>
#include <future>

namespace cogment {

class Trial;

class Environment {
public:
  using StubEntryType = std::shared_ptr<StubPool<cogmentAPI::EnvironmentSP>::Entry>;
  using StreamType = grpc::ClientReaderWriter<cogmentAPI::EnvRunTrialInput, cogmentAPI::EnvRunTrialOutput>;

  Environment(Trial* owner, const std::string& name, const std::string& impl,
        StubEntryType stub_entry, const std::optional<std::string>& config_data);
  ~Environment();

  std::future<cogmentAPI::ObservationSet> init();
  std::future<void> last_ack() { return m_last_ack_prom.get_future(); }
  void trial_ended(std::string_view details);

  const std::string& name() const { return m_name; }
  const std::string& impl() const { return m_impl; }
  bool started() const { return m_start_completed; }

  void dispatch_actions(cogmentAPI::ActionSet&& msg, bool final_tick);
  void send_message(const cogmentAPI::Message& message, const std::string& source, uint64_t tick_id);

private:
  void send_last();
  void write_to_stream(cogmentAPI::EnvRunTrialInput&& data);
  bool process_incoming_state(cogmentAPI::CommunicationState in_state, const std::string* details);
  bool process_incoming_data(cogmentAPI::EnvRunTrialOutput&& data);

  StubEntryType m_stub_entry;
  grpc::ClientContext m_context;
  std::unique_ptr<StreamType> m_stream;
  bool m_stream_valid;
  mutable std::mutex m_writing;

  Trial* const m_trial;
  const std::string m_name;
  const std::string m_impl;
  std::optional<std::string> m_config_data;
  bool m_start_completed;
  std::future<void> m_incoming_thread;

  bool m_init_completed;
  bool m_init_received;;
  std::promise<cogmentAPI::ObservationSet> m_init_prom;

  bool m_last_enabled;
  bool m_last_ack_received;
  std::promise<void> m_last_ack_prom;
};

}  // namespace cogment
#endif
