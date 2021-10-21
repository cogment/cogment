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

#ifndef NDEBUG
  #define SPDLOG_ACTIVE_LEVEL SPDLOG_LEVEL_TRACE
#endif

#include "cogment/environment.h"
#include "cogment/trial.h"
#include "cogment/actor.h"
#include "cogment/utils.h"
#include "cogment/config_file.h"

#include "spdlog/spdlog.h"

namespace cogment {

Environment::Environment(Trial* owner, const std::string& name, const std::string& impl, StubEntryType stub_entry,
                         const std::optional<std::string>& config_data) :
    m_stub_entry(std::move(stub_entry)),
    m_stream_valid(false),
    m_trial(owner),
    m_name(name),
    m_impl(impl),
    m_config_data(config_data),
    m_start_completed(false),
    m_init_received(false),
    m_last_enabled(false),
    m_last_ack_received(false) {
  SPDLOG_TRACE("Environment(): [{}] [{}] [{}]", m_trial->id(), m_name, impl);

  m_context.AddMetadata("trial-id", m_trial->id());

  // TODO: Move this out of the constructor (maybe in init) and in its own thread to wait
  //       for the stream init without blocking. But then we'll need to synchonize with other parts.
  m_stream = m_stub_entry->get_stub().RunTrial(&m_context);
  m_stream_valid = true;

  m_incoming_thread = m_trial->thread_pool().push("Environment incoming data", [this]() {
    try {
      cogmentAPI::EnvRunTrialOutput data;
      while (m_stream->Read(&data)) {
        try {
          if (!process_incoming_data(std::move(data))) {
            break;
          }
          if (!m_stream_valid) {
            break;
          }
        }
        catch (const std::exception& exc) {
          spdlog::error("Trial [{}] - Environment [{}]: Failed to process incoming data [{}]", m_trial->id(), m_name,
                        exc.what());
          break;
        }
        catch (...) {
          spdlog::error("Trial [{}] - Environment [{}]: Failed to process incoming data", m_trial->id(), m_name);
          break;
        }
      }
      SPDLOG_TRACE("Trial [{}] - Environment [{}] finished reading stream (valid [{}])", m_trial->id(), m_name,
                   m_stream_valid);
    }
    // TODO: Look into a way to cancel the stream immediately here (in case of exception in process_incoming_data)
    catch (const std::exception& exc) {
      spdlog::error("Trial [{}] - Environment [{}]: Error reading stream [{}]", m_trial->id(), m_name, exc.what());
    }
    catch (...) {
      spdlog::error("Trial [{}] - Environment [{}]: Error reading stream", m_trial->id(), m_name);
    }
  });
}

Environment::~Environment() {
  SPDLOG_TRACE("~Environment(): [{}]", m_trial->id());

  if (m_stream_valid) {
    m_stream->Finish();
  }

  if (!m_start_completed) {
    m_init_prom.set_value({});
  }

  if (!m_last_ack_received) {
    m_last_ack_prom.set_value();
  }

  m_incoming_thread.wait();
}

bool Environment::process_incoming_state(cogmentAPI::CommunicationState in_state, const std::string* details) {
  switch (in_state) {
  case cogmentAPI::CommunicationState::UNKNOWN_COM_STATE:
    if (details != nullptr) {
      throw MakeException<std::invalid_argument>("Unknown communication state: [%s]", details->c_str());
    }
    else {
      throw MakeException<std::invalid_argument>("Unknown communication state");
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
    if (!m_last_enabled) {
      m_last_enabled = true;
      if (details == nullptr) {
        SPDLOG_TRACE("Trial [{}] - Environment [{}] 'LAST' received", m_trial->id(), m_name);
      }
      else {
        spdlog::info("Trial [{}] - Environment [{}] 'LAST' received [{}]", m_trial->id(), m_name, *details);
      }
    }
    else {
      // This may happen normally if the Orchestrator sends LAST at the same time as the environment sends it
      if (details != nullptr) {
        spdlog::info("Trial [{}] - Environment [{}] Received redundant 'LAST' communication state [{}]", m_trial->id(),
                     m_name, *details);
      }
      else {
        spdlog::debug("Trial [{}] - Environment [{}] Received redundant 'LAST' communication state", m_trial->id(),
                      m_name);
      }
    }
    break;

  case cogmentAPI::CommunicationState::LAST_ACK:
    SPDLOG_DEBUG("Trial [{}] - Environment [{}] 'LAST_ACK' received", m_trial->id(), m_name);
    if (!m_last_enabled) {
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
    return false;
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
    return false;
    break;

  default:
    throw MakeException<std::invalid_argument>("Invalid communication state: [%d]", static_cast<int>(in_state));
    break;
  }

  return true;
}

// TODO: Split the init process from the normal run process (see client actor and python sdk)
bool Environment::process_incoming_data(cogmentAPI::EnvRunTrialOutput&& data) {
  const auto state = data.state();
  const auto data_case = data.data_case();
  switch (data_case) {
  case cogmentAPI::EnvRunTrialOutput::kInitOutput: {
    SPDLOG_TRACE("Trial [{}] - Environment [{}]: Received init_output", m_trial->id(), m_name);
    if (state == cogmentAPI::CommunicationState::NORMAL) {
      // TODO: Should we have a timer for this reply?
      if (!m_init_received) {
        SPDLOG_DEBUG("Trial [{}] - Environment [{}] init data received", m_trial->id(), m_name);
        m_init_received = true;
      }
      else {
        spdlog::warn("Trial [{}] - Environment [{}] Unexpected init data ignored", m_trial->id(), m_name);
      }
    }
    else {
      throw MakeException<std::invalid_argument>("'init_output' received from environment on non-normal communication");
    }
    SPDLOG_DEBUG("Trial [{}] - Environment [{}] init complete", m_trial->id(), m_name);
    break;
  }

  case cogmentAPI::EnvRunTrialOutput::kObservationSet: {
    SPDLOG_TRACE("Trial [{}] - Environment [{}]: Received observation set", m_trial->id(), m_name);
    if (state == cogmentAPI::CommunicationState::NORMAL) {
      if (m_start_completed) {
        m_trial->env_observed(m_name, std::move(*data.mutable_observation_set()), m_last_enabled);
      }
      else {
        SPDLOG_DEBUG("Trial [{}] - Environment [{}] first observations received", m_trial->id(), m_name);
        m_start_completed = true;
        m_init_prom.set_value(std::move(*data.mutable_observation_set()));
      }
    }
    else {
      throw MakeException<std::invalid_argument>(
          "'observation_set' received on from environment on non-normal communication");
    }
    break;
  }

  case cogmentAPI::EnvRunTrialOutput::kReward: {
    SPDLOG_TRACE("Trial [{}] - Environment [{}]: Received reward", m_trial->id(), m_name);
    if (state == cogmentAPI::CommunicationState::NORMAL) {
      m_trial->reward_received(data.reward(), m_name);
    }
    else {
      throw MakeException<std::invalid_argument>("'reward' received from environment on non-normal communication");
    }
    break;
  }

  case cogmentAPI::EnvRunTrialOutput::kMessage: {
    SPDLOG_TRACE("Trial [{}] - Environment [{}]: Received message", m_trial->id(), m_name);
    if (state == cogmentAPI::CommunicationState::NORMAL) {
      m_trial->message_received(data.message(), m_name);
    }
    else {
      throw MakeException<std::invalid_argument>("'message' received from environment on non-normal communication");
    }
    break;
  }

  case cogmentAPI::EnvRunTrialOutput::kDetails: {
    SPDLOG_TRACE("Trial [{}] - Environment [{}]: Received details", m_trial->id(), m_name);
    return process_incoming_state(state, &data.details());
    break;
  }

  case cogmentAPI::EnvRunTrialOutput::DATA_NOT_SET: {
    SPDLOG_TRACE("Trial [{}] - Environment [{}]: Received new communcation state [{}]", m_trial->id(), m_name, state);
    return process_incoming_state(state, nullptr);
    break;
  }

  default: {
    throw MakeException<std::invalid_argument>("Unknown communication data [%d]", static_cast<int>(data_case));
    break;
  }
  }

  return m_stream_valid;
}

void Environment::write_to_stream(cogmentAPI::EnvRunTrialInput&& data) {
  const std::lock_guard<std::mutex> lg(m_writing);
  if (m_stream_valid) {
    m_stream->Write(std::move(data));
  }
  else {
    spdlog::error("Trial [{}] - Environment [{}] stream has ended: cannot write", m_trial->id(), m_name);
  }
}

void Environment::trial_ended(std::string_view details) {
  const std::lock_guard<std::mutex> lg(m_writing);

  if (!m_stream_valid) {
    SPDLOG_DEBUG("Trial [{}] - Environment [{}] stream has ended: cannot end it again", m_trial->id(), m_name);
    return;
  }

  cogmentAPI::EnvRunTrialInput msg;
  msg.set_state(cogmentAPI::CommunicationState::END);
  if (!details.empty()) {
    msg.set_details(details.data(), details.size());
  }
  m_stream->Write(std::move(msg));
  SPDLOG_DEBUG("Trial [{}] - Environment [{}] 'END' sent", m_trial->id(), m_name);

  m_stream->WritesDone();
  m_stream->Finish();
  m_stream_valid = false;
}

std::future<cogmentAPI::ObservationSet> Environment::init() {
  SPDLOG_TRACE("Trial [{}] - Environment::init(): [{}]", m_trial->id(), m_name);

  cogmentAPI::EnvRunTrialInput msg;
  msg.set_state(cogmentAPI::CommunicationState::NORMAL);

  auto init_data = msg.mutable_init_input();
  init_data->set_name(m_name);
  init_data->set_impl_name(m_impl);
  init_data->set_tick_id(m_trial->tick_id());
  if (m_config_data) {
    init_data->mutable_config()->set_content(m_config_data.value());
  }
  for (const auto& actor : m_trial->actors()) {
    auto env_actor = init_data->add_actors_in_trial();
    env_actor->set_name(actor->actor_name());
    env_actor->set_actor_class(actor->actor_class());
  }

  write_to_stream(std::move(msg));

  return m_init_prom.get_future();
}

void Environment::send_last() {
  cogmentAPI::EnvRunTrialInput msg;
  msg.set_state(cogmentAPI::CommunicationState::LAST);
  write_to_stream(std::move(msg));

  m_last_enabled = true;
  SPDLOG_DEBUG("Trial [{}] - Environment [{}] 'LAST' sent", m_trial->id(), m_name);
}

void Environment::dispatch_actions(cogmentAPI::ActionSet&& set, bool final_tick) {
  if (!m_last_enabled) {
    if (final_tick) {
      send_last();
    }

    cogmentAPI::EnvRunTrialInput msg;
    msg.set_state(cogmentAPI::CommunicationState::NORMAL);
    *(msg.mutable_action_set()) = std::move(set);
    write_to_stream(std::move(msg));
  }
}

void Environment::send_message(const cogmentAPI::Message& message, const std::string& source, uint64_t tick_id) {
  cogmentAPI::EnvRunTrialInput in;
  in.set_state(cogmentAPI::CommunicationState::NORMAL);
  auto msg = in.mutable_message();
  msg->CopyFrom(message);
  msg->set_tick_id(tick_id);
  msg->set_sender_name(source);
  msg->set_receiver_name(m_name);  // Because of wildcard destination

  write_to_stream(std::move(in));
}

}  // namespace cogment
