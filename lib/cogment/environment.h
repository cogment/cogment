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

#ifndef COGMENT_ORCHESTRATOR_ENVIRONMENT_H
#define COGMENT_ORCHESTRATOR_ENVIRONMENT_H

#include "cogment/stub_pool.h"

#include "grpc++/grpc++.h"

#include "cogment/api/environment.grpc.pb.h"
#include "cogment/api/common.pb.h"

#include <vector>
#include <future>

namespace cogment {

class Trial;

class Environment {
  using StubEntryType = std::shared_ptr<StubPool<cogmentAPI::EnvironmentSP>::Entry>;
  using StreamType = grpc::ClientReaderWriter<cogmentAPI::EnvRunTrialInput, cogmentAPI::EnvRunTrialOutput>;

public:
  Environment(Trial* owner, const cogmentAPI::EnvironmentParams& params, StubEntryType stub_entry);
  ~Environment();

  std::future<void> init();
  std::future<void> last_ack() { return m_last_ack_prom.get_future(); }
  void trial_ended(std::string_view details);

  const std::string& name() const { return m_name; }

  void dispatch_actions(cogmentAPI::ActionSet&& msg, bool last);
  void send_message(const cogmentAPI::Message& message, uint64_t tick_id);

private:
  void read_init_data();
  void dispatch_init_data();
  void write_to_stream(cogmentAPI::EnvRunTrialInput&& data);
  void process_incoming_state(cogmentAPI::CommunicationState in_state, const std::string* details);
  void process_incoming_data(cogmentAPI::EnvRunTrialOutput&& data);
  void process_incoming_stream();
  void run(std::unique_ptr<StreamType> stream);
  void dispatch_message(cogmentAPI::Message&& message);
  void finish_stream();

  StubEntryType m_stub_entry;
  std::unique_ptr<StreamType> m_stream;
  bool m_stream_valid;
  grpc::ClientContext m_context;
  std::mutex m_writing;

  Trial* const m_trial;
  const std::string m_name;
  const std::string m_impl;
  std::string m_config_data;
  bool m_has_config;

  bool m_init_completed;
  std::future<void> m_incoming_thread;

  std::promise<void> m_init_prom;

  bool m_last_sent_received;
  bool m_last_ack_received;
  std::promise<void> m_last_ack_prom;
};

}  // namespace cogment
#endif
