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

#ifndef NDEBUG
  #define SPDLOG_ACTIVE_LEVEL SPDLOG_LEVEL_TRACE
#endif

#include "cogment/environment.h"
#include "cogment/actor.h"
#include "cogment/trial.h"
#include "cogment/utils.h"

#include "spdlog/spdlog.h"

namespace cogment {

Environment::Environment(Trial* owner, const cogmentAPI::EnvironmentParams& params, StubEntryType stub_entry) :
    m_stub_entry(std::move(stub_entry)),
    m_stream_valid(false),
    m_trial(owner),
    m_name(params.name()),
    m_impl(params.implementation()),
    m_has_config(params.has_config()),
    m_init_completed(false),
    m_last_sent_received(false),
    m_last_ack_received(false) {
  SPDLOG_TRACE("Environment(): [{}] [{}] [{}]", m_trial->id(), m_name, m_impl);

  if (m_has_config) {
    m_config_data = params.config().content();
  }

  m_context.AddMetadata("trial-id", m_trial->id());
}

Environment::~Environment() {
  SPDLOG_TRACE("~Environment(): [{}]", m_trial->id());

  finish_stream();

  if (!m_init_completed) {
    m_init_prom.set_value();
  }

  if (!m_last_ack_received) {
    m_last_ack_prom.set_value();
  }

  if (m_incoming_thread.valid()) {
    m_incoming_thread.wait();
  }
}

void Environment::write_to_stream(cogmentAPI::EnvRunTrialInput&& data) {
  const std::lock_guard lg(m_writing);
  if (m_stream_valid) {
    try {
      m_stream_valid = m_stream->Write(std::move(data));
    }
    catch (...) {
      m_stream_valid = false;
      throw;
    }
  }
  else {
    throw MakeException("Environment stream has closed");
  }
}

void Environment::send_message(const cogmentAPI::Message& message, uint64_t tick_id) {
  cogmentAPI::Message msg(message);
  msg.set_tick_id(tick_id);
  msg.set_receiver_name(m_name);  // Because of possible wildcards in message receiver

  dispatch_message(std::move(msg));
}

void Environment::read_init_data() {
  SPDLOG_TRACE("Environment read_init_data");

  for (cogmentAPI::EnvRunTrialOutput data; m_stream_valid && m_stream->Read(&data); data.Clear()) {
    const auto state = data.state();
    const auto data_case = data.data_case();

    switch (state) {
    case cogmentAPI::CommunicationState::NORMAL: {
      if (data_case == cogmentAPI::EnvRunTrialOutput::DataCase::kInitOutput) {
        return;
      }
      else {
        throw MakeException("Data [{}] received before init data", static_cast<int>(data_case));
      }
    }

    case cogmentAPI::CommunicationState::HEARTBEAT: {
      if (data_case == cogmentAPI::EnvRunTrialOutput::DataCase::kDetails) {
        spdlog::info("Heartbeat requested from environment: [{}]", data.details());
      }
      cogmentAPI::EnvRunTrialInput msg;
      msg.set_state(cogmentAPI::CommunicationState::HEARTBEAT);
      m_stream_valid = m_stream->Write(std::move(msg));
      break;
    }

    case cogmentAPI::CommunicationState::LAST: {
      throw MakeException("Unexpected reception of communication state (LAST) from environment");
    }

    case cogmentAPI::CommunicationState::LAST_ACK: {
      throw MakeException("Unexpected reception of communication state (LAST_ACK) from environment");
    }

    case cogmentAPI::CommunicationState::END: {
      if (data_case == cogmentAPI::EnvRunTrialOutput::DataCase::kDetails) {
        spdlog::error("Unexpected end of communication (END) from environment: [{}]", data.details());
      }
      else {
        spdlog::error("Unexpected end of communication (END) from environment");
      }
      m_stream_valid = false;
    }

    default:
      throw MakeException("Unknown communication state [{}] received from environment", static_cast<int>(state));
    }
  }

  m_stream_valid = false;
}

void Environment::process_incoming_state(cogmentAPI::CommunicationState in_state, const std::string* details) {
  switch (in_state) {
  case cogmentAPI::CommunicationState::UNKNOWN_COM_STATE:
    if (details != nullptr) {
      throw MakeException("Unknown communication state: [{}]", *details);
    }
    else {
      throw MakeException("Unknown communication state");
    }
    break;

  case cogmentAPI::CommunicationState::NORMAL:
    if (details != nullptr) {
      spdlog::info("Trial [{}] - Environment [{}] Communication details received [{}]", m_trial->id(), m_name,
                   *details);
    }
    else {
      spdlog::warn("Trial [{}] - Environment [{}] No data in normal communication received", m_trial->id(), m_name);
    }
    break;

  case cogmentAPI::CommunicationState::HEARTBEAT:
    SPDLOG_TRACE("Trial [{}] - Environment [{}] 'HEARTBEAT' received", m_trial->id(), m_name);
    if (details != nullptr) {
      spdlog::info("Trial [{}] - Environment [{}] Heartbeat requested [{}]", m_trial->id(), m_name, *details);
    }
    // TODO : manage heartbeats
    break;

  case cogmentAPI::CommunicationState::LAST:
    m_last_sent_received = true;
    if (details == nullptr) {
      SPDLOG_TRACE("Trial [{}] - Environment [{}] 'LAST' received", m_trial->id(), m_name);
    }
    else {
      spdlog::info("Trial [{}] - Environment [{}] 'LAST' received [{}]", m_trial->id(), m_name, *details);
    }
    break;

  case cogmentAPI::CommunicationState::LAST_ACK:
    SPDLOG_DEBUG("Trial [{}] - Environment [{}] 'LAST_ACK' received", m_trial->id(), m_name);
    if (!m_last_sent_received) {
      if (details != nullptr) {
        spdlog::error("Trial [{}] - Environment [{}] Unexpected communication state (LAST_ACK) received [{}]",
                      m_trial->id(), m_name, *details);
      }
      else {
        spdlog::error("Trial [{}] - Environment [{}] Unexpected communication state (LAST_ACK) received", m_trial->id(),
                      m_name);
      }
    }
    m_last_ack_received = true;
    m_last_ack_prom.set_value();
    break;

  case cogmentAPI::CommunicationState::END:
    // TODO: Decide what to do about "END" received from environment
    if (details != nullptr) {
      spdlog::warn("Trial [{}] - Environment [{}] Communication state (END) received [{}]", m_trial->id(), m_name,
                   *details);
    }
    else {
      spdlog::warn("Trial [{}] - Environment [{}] Communication state (END) received", m_trial->id(), m_name);
    }
    break;

  default:
    throw MakeException("Invalid communication state: [{}]", static_cast<int>(in_state));
    break;
  }
}

void Environment::process_incoming_data(cogmentAPI::EnvRunTrialOutput&& data) {
  const auto state = data.state();
  const auto data_case = data.data_case();
  SPDLOG_TRACE("Trial [{}] - Environment [{}]: Processing incoming data [{}] [{}]", m_trial->id(), m_name, state,
               data_case);

  switch (data_case) {
  case cogmentAPI::EnvRunTrialOutput::DataCase::kInitOutput: {
    throw MakeException("Repeated init data");
    break;
  }

  case cogmentAPI::EnvRunTrialOutput::DataCase::kObservationSet: {
    if (state == cogmentAPI::CommunicationState::NORMAL) {
      m_trial->env_observed(m_name, std::move(*data.mutable_observation_set()), m_last_sent_received);
    }
    else {
      throw MakeException("Observation received on non-normal communication");
    }
    break;
  }

  case cogmentAPI::EnvRunTrialOutput::DataCase::kReward: {
    if (state == cogmentAPI::CommunicationState::NORMAL) {
      m_trial->reward_received(m_name, std::move(*data.mutable_reward()));
    }
    else {
      throw MakeException("Reward received on non-normal communication");
    }
    break;
  }

  case cogmentAPI::EnvRunTrialOutput::DataCase::kMessage: {
    if (state == cogmentAPI::CommunicationState::NORMAL) {
      m_trial->message_received(m_name, std::move(*data.mutable_message()));
    }
    else {
      throw MakeException("Message received on non-normal communication");
    }
    break;
  }

  case cogmentAPI::EnvRunTrialOutput::DataCase::kDetails: {
    process_incoming_state(state, &data.details());
    break;
  }

  case cogmentAPI::EnvRunTrialOutput::DataCase::DATA_NOT_SET: {
    process_incoming_state(state, nullptr);
    break;
  }

  default: {
    throw MakeException("Unknown communication data [{}]", static_cast<int>(data_case));
    break;
  }
  }
}

void Environment::process_incoming_stream() {
  for (cogmentAPI::EnvRunTrialOutput data; m_stream_valid && m_stream->Read(&data); data.Clear()) {
    try {
      process_incoming_data(std::move(data));
    }
    catch (const std::exception& exc) {
      spdlog::error("Trial [{}] - Environment [{}] failed to process incoming data [{}]", m_trial->id(), m_name,
                    exc.what());
    }
    catch (...) {
      spdlog::error("Trial [{}] - Environment [{}] failed to process incoming data", m_trial->id(), m_name);
    }
  }
  SPDLOG_DEBUG("Trial [{}] - Environment [{}] finished reading stream (valid [{}])", m_trial->id(), m_name,
               m_stream_valid);
}

void Environment::run(std::unique_ptr<StreamType> stream) {
  SPDLOG_TRACE("Trial [{}] - Environment [{}] run", m_trial->id(), m_name);

  if (m_stream != nullptr) {
    throw MakeException("Environment already running");
  }
  m_stream = std::move(stream);
  m_stream_valid = true;

  dispatch_init_data();

  m_incoming_thread = m_trial->thread_pool().push("Environment incoming data", [this]() {
    try {
      read_init_data();

      if (m_stream_valid) {
        m_init_prom.set_value();
        m_init_completed = true;
        spdlog::debug("Trial [{}] - Environment [{}] init complete", m_trial->id(), m_name);
      }

      process_incoming_stream();
    }
    catch (const std::exception& exc) {
      spdlog::error("Trial [{}] - Environment [{}] failed to process stream [{}]", m_trial->id(), m_name, exc.what());
    }
    catch (...) {
      spdlog::error("Trial [{}] - Environment [{}] failed to process stream", m_trial->id(), m_name);
    }

    finish_stream();
  });
}

std::future<void> Environment::init() {
  SPDLOG_TRACE("Trial [{}] - Environment::init(): [{}]", m_trial->id(), m_name);

  run(m_stub_entry->get_stub().RunTrial(&m_context));

  return m_init_prom.get_future();
}

void Environment::dispatch_init_data() {
  cogmentAPI::EnvRunTrialInput msg;
  msg.set_state(cogmentAPI::CommunicationState::NORMAL);

  auto init_data = msg.mutable_init_input();
  init_data->set_name(m_name);
  init_data->set_impl_name(m_impl);
  init_data->set_tick_id(m_trial->tick_id());
  if (m_has_config) {
    init_data->mutable_config()->set_content(m_config_data);
  }
  for (const auto& actor : m_trial->actors()) {
    auto env_actor = init_data->add_actors_in_trial();
    env_actor->set_name(actor->actor_name());
    env_actor->set_actor_class(actor->actor_class());
  }

  write_to_stream(std::move(msg));
}

void Environment::dispatch_actions(cogmentAPI::ActionSet&& set, bool last) {
  if (last) {
    cogmentAPI::EnvRunTrialInput msg;
    msg.set_state(cogmentAPI::CommunicationState::LAST);
    write_to_stream(std::move(msg));
    SPDLOG_DEBUG("Trial [{}] - Environment [{}] 'LAST' sent", m_trial->id(), m_name);
    m_last_sent_received = true;
  }

  cogmentAPI::EnvRunTrialInput msg;
  msg.set_state(cogmentAPI::CommunicationState::NORMAL);
  *(msg.mutable_action_set()) = std::move(set);
  write_to_stream(std::move(msg));
}

void Environment::dispatch_message(cogmentAPI::Message&& message) {
  cogmentAPI::EnvRunTrialInput msg;
  msg.set_state(cogmentAPI::CommunicationState::NORMAL);
  *(msg.mutable_message()) = std::move(message);
  write_to_stream(std::move(msg));
}

void Environment::trial_ended(std::string_view details) {
  if (!m_stream_valid) {
    SPDLOG_DEBUG("Trial [{}] - Environment [{}] stream has ended: cannot end it again", m_trial->id(), m_name);
    return;
  }

  cogmentAPI::EnvRunTrialInput msg;
  msg.set_state(cogmentAPI::CommunicationState::END);
  if (!details.empty()) {
    msg.set_details(details.data(), details.size());
  }

  write_to_stream(std::move(msg));
  SPDLOG_DEBUG("Trial [{}] - Environment [{}] 'END' sent", m_trial->id(), m_name);

  const std::lock_guard lg(m_writing);
  if (m_stream_valid) {
    try {
      m_stream_valid = m_stream->WritesDone();
    }
    catch (...) {
      m_stream_valid = false;
      throw;
    }
  }

  finish_stream();
}

void Environment::finish_stream() {
  // m_stream->Finish();  // This seems to cause a crash in grpc

  m_stream_valid = false;
}

}  // namespace cogment
