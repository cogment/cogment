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

#ifndef COGMENT_ORCHESTRATOR_ACTOR_H
#define COGMENT_ORCHESTRATOR_ACTOR_H

#include "grpc++/grpc++.h"
#include "spdlog/spdlog.h"

#include "cogment/api/common.pb.h"

#include <mutex>
#include <string>
#include <future>

namespace cogment {

class Trial;

// Bare minimum to allow a common stream to represent client and server
class ActorStream {
public:
  using InputType = cogmentAPI::ActorRunTrialInput;
  using OutputType = cogmentAPI::ActorRunTrialOutput;

  ActorStream() {}
  virtual ~ActorStream() {}

  virtual bool read(OutputType* data) = 0;
  virtual bool write(const InputType& data) = 0;
  virtual bool write_last(const InputType& data) = 0;
  virtual bool finish() = 0;
};

// This class is to try to compensate/workaround limitations and bugs in gRPC
class ManagedStream {
public:
  ManagedStream() : m_stream_valid(false), m_last_writen(false) {}
  void operator=(std::unique_ptr<ActorStream> stream);

  ActorStream* actor_stream_ptr() { return m_stream.get(); }
  bool has_stream() const { return (m_stream != nullptr); }
  bool is_valid() const { return m_stream_valid; }

  bool read(ActorStream::OutputType* data);
  bool write(const ActorStream::InputType& data);
  bool write_last(const ActorStream::InputType& data);
  void finish();

private:
  std::unique_ptr<ActorStream> m_stream;
  std::mutex m_writing;
  std::mutex m_reading;
  std::atomic_bool m_stream_valid;
  std::atomic_bool m_last_writen;
};

class Actor {
  using TickIdType = uint64_t;
  using RewardAccumulator = std::map<TickIdType, cogmentAPI::Reward>;

public:
  Actor(Trial* owner, const cogmentAPI::ActorParams& params, bool read_init);
  virtual ~Actor();

  virtual std::future<void> init();

  bool has_joined() const { return m_stream.has_stream(); }
  std::future<void> last_ack() { return m_last_ack_prom.get_future(); }

  Trial* trial() const { return m_trial; }
  const std::string& actor_name() const { return m_name; }
  const std::string& actor_class() const { return m_actor_class; }

  void add_reward_src(const cogmentAPI::RewardSource& source, TickIdType tick_id);
  void send_message(const cogmentAPI::Message& message, TickIdType tick_id);

  void dispatch_tick(cogmentAPI::Observation&& obs, bool final_tick);
  void trial_ended(std::string_view details);

protected:
  static bool read_init_data(ActorStream* stream, cogmentAPI::ActorInitialOutput* out);
  std::future<void> run(std::unique_ptr<ActorStream> stream);

private:
  void write_to_stream(ActorStream::InputType&& data);
  void dispatch_init_data();
  void dispatch_observation(cogmentAPI::Observation&& obs, bool last);
  void dispatch_reward(cogmentAPI::Reward&& reward);
  void dispatch_message(cogmentAPI::Message&& message);
  void process_incoming_state(cogmentAPI::CommunicationState in_state, const std::string* details);
  void process_incoming_data(ActorStream::OutputType&& data);
  void process_incoming_stream();
  void finish_stream();

  bool m_wait_for_init_data;

  ManagedStream m_stream;

  Trial* const m_trial;
  const std::string m_name;
  const std::string m_actor_class;
  const std::string m_impl;
  std::string m_config_data;
  bool m_has_config;

  std::mutex m_reward_lock;
  RewardAccumulator m_reward_accumulator;

  std::future<void> m_incoming_thread;

  bool m_init_completed;
  std::promise<void> m_init_prom;

  bool m_last_sent;
  bool m_last_ack_received;
  std::promise<void> m_last_ack_prom;

  std::promise<void> m_finished_prom;
  std::atomic_bool m_finished;
};

}  // namespace cogment
#endif
