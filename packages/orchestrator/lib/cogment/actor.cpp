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

#include "cogment/actor.h"
#include "cogment/utils.h"
#include "cogment/config_file.h"
#include "cogment/trial.h"

namespace {

const std::string EMPTY;

float compute_reward_value(const cogmentAPI::Reward& reward) {
  float value_accum = 0.0f;
  float confidence_accum = 0.0f;

  for (auto& src : reward.sources()) {
    float src_conf = src.confidence();

    if (src_conf > 0.0f) {
      value_accum += src.value() * src_conf;
      confidence_accum += src_conf;
    }
  }

  if (confidence_accum > 0.0f) {
    value_accum /= confidence_accum;
  }

  return value_accum;
}

}  // namespace

namespace cogment {

void ManagedStream::operator=(std::unique_ptr<ActorStream> stream) {
  // Testing the locks is very hard due to spurious false return of try_lock.
  // In our use case, this following test is good enough.
  if (m_stream != nullptr) {
    throw MakeException("Cannot overwrite an established managed actor stream");
  }
  if (stream == nullptr) {
    throw MakeException("Invalid actor stream");
  }

  m_stream = std::move(stream);
  m_stream_valid = true;
}

bool ManagedStream::read(ActorStream::OutputType* data) {
  const std::lock_guard lg(m_reading);
  if (m_stream_valid) {
    try {
      // We never want to set m_stream_valid to "true", so we use an "if" statement
      if (!m_stream->read(data)) {
        m_stream_valid = false;
      }
    }
    catch (...) {
      spdlog::error("gRPC failure reading");
      m_stream_valid = false;
    }
  }
  return m_stream_valid;
}

bool ManagedStream::write(const ActorStream::InputType& data) {
  const std::lock_guard lg(m_writing);
  if (m_stream_valid) {
    if (!m_last_writen) {
      try {
        // We never want to set m_stream_valid to "true", so we use an "if" statement
        if (!m_stream->write(data)) {
          m_stream_valid = false;
        }
      }
      catch (...) {
        spdlog::error("gRPC failure writing");
        m_stream_valid = false;
      }
    }
    else {
      spdlog::warn("Trying to write after last writen");
    }
  }

  return m_stream_valid;
}

bool ManagedStream::write_last(const ActorStream::InputType& data) {
  const std::lock_guard lg(m_writing);
  if (m_stream_valid) {
    if (!m_last_writen) {
      try {
        if (m_stream->write_last(data)) {
          m_last_writen = true;
        }
        else {
          m_stream_valid = false;
        }
      }
      catch (...) {
        spdlog::error("gRPC failure writing last data");
        m_stream_valid = false;
      }
    }
    else {
      spdlog::warn("Trying to write last data after last writen");
    }
  }

  return m_stream_valid;
}

void ManagedStream::finish() {
  const std::lock_guard lg(m_writing);
  if (m_stream_valid) {
    m_stream_valid = false;
    try {
      m_stream->finish();
    }
    catch (...) {
      spdlog::error("gRPC failure finishing");
    }
  }
}

// Static
bool Actor::read_init_data(ActorStream* stream, cogmentAPI::ActorInitialOutput* out) {
  SPDLOG_TRACE("Actor read_init_data");

  for (ActorStream::OutputType data; stream->read(&data); data.Clear()) {
    const auto state = data.state();
    const auto data_case = data.data_case();

    switch (state) {
    case cogmentAPI::CommunicationState::NORMAL: {
      if (data_case == ActorStream::OutputType::DataCase::kInitOutput) {
        if (out != nullptr) {
          *out = std::move(data.init_output());
        }
        return true;
      }
      else {
        throw MakeException("Data [{}] received before init data", static_cast<int>(data_case));
      }
    }

    case cogmentAPI::CommunicationState::HEARTBEAT: {
      if (data_case == ActorStream::OutputType::DataCase::kDetails) {
        spdlog::info("Heartbeat requested from actor: [{}]", data.details());
      }
      ActorStream::InputType msg;
      msg.set_state(cogmentAPI::CommunicationState::HEARTBEAT);
      if (!stream->write(std::move(msg))) {
        return false;
      }
      break;
    }

    case cogmentAPI::CommunicationState::LAST: {
      throw MakeException("Unexpected reception of communication state (LAST) from actor");
    }

    case cogmentAPI::CommunicationState::LAST_ACK: {
      throw MakeException("Unexpected reception of communication state (LAST_ACK) from actor");
    }

    case cogmentAPI::CommunicationState::END: {
      if (data_case == ActorStream::OutputType::DataCase::kDetails) {
        spdlog::error("Unexpected end of communication (END) from actor: [{}]", data.details());
      }
      else {
        spdlog::error("Unexpected end of communication (END) from actor");
      }
      return false;
    }

    default:
      throw MakeException("Unknown communication state [{}] received from actor", static_cast<int>(state));
    }
  }

  return false;
}

Actor::Actor(Trial* owner, const cogmentAPI::ActorParams& params, bool read_init) :
    m_wait_for_init_data(read_init),
    m_trial(owner),
    m_params(params),
    m_name(params.name()),
    m_actor_class(params.actor_class()),
    m_init_completed(false),
    m_disengaged(false),
    m_last_sent(false),
    m_last_ack_received(false),
    m_finished(false) {
  SPDLOG_TRACE("Actor(): [{}] [{}] [{}] [{}]", m_trial->id(), m_name, m_actor_class, params.implementation());
}

Actor::~Actor() {
  SPDLOG_TRACE("~Actor(): [{}] [{}]", m_trial->id(), m_name);

  m_disengaged = true;
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

void Actor::disengage() {
  if (!m_disengaged.exchange(true)) {
    m_trial->actor_disengaging(*this);
  }
}

void Actor::write_to_stream(ActorStream::InputType&& data) {
  if (!m_stream.write(std::move(data))) {
    throw MakeException("Actor stream failure");
  }
}

void Actor::add_reward_src(const cogmentAPI::RewardSource& source, TickIdType tick_id) {
  if (m_disengaged) {
    return;
  }

  const std::lock_guard lg(m_reward_lock);
  auto& rew = m_reward_accumulator[tick_id];
  auto new_src = rew.add_sources();
  *new_src = source;
}

void Actor::send_message(const cogmentAPI::Message& message, TickIdType tick_id) {
  if (m_disengaged) {
    return;
  }

  cogmentAPI::Message msg(message);
  msg.set_tick_id(tick_id);
  msg.set_receiver_name(m_name);  // Because of possible wildcards in message receiver

  try {
    dispatch_message(std::move(msg));
  }
  catch (const std::exception& exc) {
    connection_error("Trial [{}] - Actor [{}]: Failed to send message [{}]", m_trial->id(), m_name, exc.what());
  }
  catch (...) {
    connection_error("Trial [{}] - Actor [{}]: Failed to send message", m_trial->id(), m_name);
  }
}

void Actor::dispatch_tick(cogmentAPI::Observation&& obs, bool final_tick) {
  if (m_disengaged) {
    return;
  }

  RewardAccumulator reward_acc;
  {
    const std::lock_guard lg(m_reward_lock);
    reward_acc.swap(m_reward_accumulator);
  }

  try {
    for (auto& value : reward_acc) {
      const auto tick_id = value.first;
      auto& reward = value.second;

      auto val = compute_reward_value(reward);
      reward.set_value(val);
      reward.set_tick_id(tick_id);
      reward.set_receiver_name(m_name);
      dispatch_reward(std::move(reward));
    }

    dispatch_observation(std::move(obs), final_tick);
  }
  catch (const std::exception& exc) {
    connection_error("Trial [{}] - Actor [{}]: Failed to process outgoing data [{}]", m_trial->id(), m_name,
                     exc.what());
  }
  catch (...) {
    connection_error("Trial [{}] - Actor [{}]: Failed to process outgoing data", m_trial->id(), m_name);
  }
}

bool Actor::process_initialization() {
  bool result = false;

  dispatch_init_data();

  if (m_wait_for_init_data) {
    result = (m_stream.is_valid() && read_init_data(m_stream.actor_stream_ptr(), nullptr));
  }
  else {
    result = true;
  }

  if (result) {
    m_init_prom.set_value();
    m_init_completed = true;
    spdlog::debug("Trial [{}] - Actor [{}] init complete", m_trial->id(), m_name);
  }

  return result;
}

void Actor::process_incoming_state(cogmentAPI::CommunicationState in_state, const std::string* details) {
  switch (in_state) {
  case cogmentAPI::CommunicationState::UNKNOWN_COM_STATE:
    if (details == nullptr) {
      throw MakeException("Unknown communication state");
    }
    else {
      throw MakeException("Unknown communication state: [{}]", *details);
    }
    break;

  case cogmentAPI::CommunicationState::NORMAL:
    if (details == nullptr) {
      spdlog::warn("Trial [{}] - Actor [{}] No data in normal communication received", m_trial->id(), m_name);
    }
    else {
      spdlog::info("Trial [{}] - Actor [{}] Communication details received [{}]", m_trial->id(), m_name, *details);
    }
    break;

  case cogmentAPI::CommunicationState::HEARTBEAT:
    SPDLOG_TRACE("Trial [{}] - Actor [{}] 'HEARTBEAT' received", m_trial->id(), m_name);
    if (details != nullptr) {
      spdlog::info("Trial [{}] - Actor [{}] Heartbeat requested [{}]", m_trial->id(), m_name, *details);
    }
    // TODO : manage heartbeats
    break;

  case cogmentAPI::CommunicationState::LAST:
    if (details != nullptr) {
      spdlog::error("Trial [{}] - Actor [{}] Unexpected communication state (LAST) received [{}]", m_trial->id(),
                    m_name, *details);
    }
    else {
      spdlog::error("Trial [{}] - Actor [{}] Unexpected communication state (LAST) received", m_trial->id(), m_name);
    }
    break;

  case cogmentAPI::CommunicationState::LAST_ACK:
    SPDLOG_DEBUG("Trial [{}] - Actor [{}] 'LAST_ACK' received", m_trial->id(), m_name);
    if (!m_last_sent) {
      if (details != nullptr) {
        spdlog::error("Trial [{}] - Actor [{}] Unexpected communication state (LAST_ACK) received [{}]", m_trial->id(),
                      m_name, *details);
      }
      else {
        spdlog::error("Trial [{}] - Actor [{}] Unexpected communication state (LAST_ACK) received", m_trial->id(),
                      m_name);
      }
    }
    // TODO: Should we accept even if the "LAST" was not sent?
    //       This could be used to indicate that the actor has finished interacting with the trial.
    m_last_ack_received = true;
    m_last_ack_prom.set_value();
    break;

  case cogmentAPI::CommunicationState::END:
    // TODO: Decide what to do about "END" received from actors
    if (details != nullptr) {
      spdlog::error("Trial [{}] - Actor [{}] Unexpected communication state (END) received [{}]", m_trial->id(), m_name,
                    *details);
    }
    else {
      spdlog::error("Trial [{}] - Actor [{}] Unexpected communication state (END) received", m_trial->id(), m_name);
    }
    break;

  default:
    throw MakeException("Invalid communication state: [{}]", static_cast<int>(in_state));
    break;
  }
}

void Actor::process_incoming_data(ActorStream::OutputType&& data) {
  const auto state = data.state();
  const auto data_case = data.data_case();
  SPDLOG_TRACE("Trial [{}] - Actor [{}]: Processing incoming data [{}] [{}]", m_trial->id(), m_name, state, data_case);

  switch (data_case) {
  case ActorStream::OutputType::DataCase::kInitOutput: {
    throw MakeException("Repeated init data");
    break;
  }

  case ActorStream::OutputType::DataCase::kAction: {
    if (state == cogmentAPI::CommunicationState::NORMAL) {
      m_trial->actor_acted(m_name, std::move(*data.mutable_action()));
    }
    else {
      throw MakeException("Action received on non-normal communication");
    }
    break;
  }

  case ActorStream::OutputType::DataCase::kReward: {
    if (state == cogmentAPI::CommunicationState::NORMAL) {
      m_trial->reward_received(m_name, std::move(*data.mutable_reward()));
    }
    else {
      throw MakeException("Reward received on non-normal communication");
    }
    break;
  }

  case ActorStream::OutputType::DataCase::kMessage: {
    if (state == cogmentAPI::CommunicationState::NORMAL) {
      m_trial->message_received(m_name, std::move(*data.mutable_message()));
    }
    else {
      throw MakeException("Message received on non-normal communication");
    }
    break;
  }

  case ActorStream::OutputType::DataCase::kDetails: {
    process_incoming_state(state, &data.details());
    break;
  }

  case ActorStream::OutputType::DataCase::DATA_NOT_SET: {
    process_incoming_state(state, nullptr);
    break;
  }

  default: {
    throw MakeException("Unknown communication data [{}]", static_cast<int>(data_case));
    break;
  }
  }
}

void Actor::process_incoming_stream() {
  for (ActorStream::OutputType data; m_stream.read(&data); data.Clear()) {
    if (m_disengaged) {
      break;
    }

    process_incoming_data(std::move(data));
  }

  SPDLOG_DEBUG("Trial [{}] - Actor [{}] finished reading stream", m_trial->id(), m_name);
}

// We use a lambda so we can call it inside the thread since it can be slow to
// get the stream in case of errors.
std::future<void> Actor::run(std::function<std::unique_ptr<ActorStream>()> stream_func) {
  SPDLOG_TRACE("Trial [{}] - Actor [{}] run", m_trial->id(), m_name);

  if (m_stream.has_stream()) {
    throw MakeException("Actor already running");
  }

  m_incoming_thread =
      m_trial->thread_pool().push("Actor incoming data", [this, stream_func = std::move(stream_func)]() {
        bool init_success = false;
        try {
          m_stream = stream_func();
          init_success = process_initialization();
        }
        catch (const std::exception& exc) {
          connection_error("Trial [{}] - Actor [{}] failed to initialize [{}]", m_trial->id(), m_name, exc.what());
        }
        catch (...) {
          connection_error("Trial [{}] - Actor [{}] failed to initialize", m_trial->id(), m_name);
        }

        // This is separate to have a better error output in case of exception.
        try {
          if (init_success) {
            process_incoming_stream();
          }
        }
        catch (const std::exception& exc) {
          connection_error("Trial [{}] - Actor [{}] failed to process stream [{}]", m_trial->id(), m_name, exc.what());
        }
        catch (...) {
          connection_error("Trial [{}] - Actor [{}] failed to process stream", m_trial->id(), m_name);
        }

        disengage();
      });

  return m_finished_prom.get_future();
}

void Actor::dispatch_observation(cogmentAPI::Observation&& observation, bool last) {
  if (last) {
    ActorStream::InputType msg;
    msg.set_state(cogmentAPI::CommunicationState::LAST);
    write_to_stream(std::move(msg));
    SPDLOG_DEBUG("Trial [{}] - Actor [{}] 'LAST' sent", m_trial->id(), m_name);
    m_last_sent = true;
  }

  ActorStream::InputType msg;
  msg.set_state(cogmentAPI::CommunicationState::NORMAL);
  *(msg.mutable_observation()) = std::move(observation);
  write_to_stream(std::move(msg));
}

void Actor::dispatch_reward(cogmentAPI::Reward&& reward) {
  ActorStream::InputType msg;
  msg.set_state(cogmentAPI::CommunicationState::NORMAL);
  *(msg.mutable_reward()) = std::move(reward);
  write_to_stream(std::move(msg));
}

void Actor::dispatch_message(cogmentAPI::Message&& message) {
  ActorStream::InputType msg;
  msg.set_state(cogmentAPI::CommunicationState::NORMAL);
  *(msg.mutable_message()) = std::move(message);
  write_to_stream(std::move(msg));
}

void Actor::dispatch_init_data() {
  ActorStream::InputType msg;
  msg.set_state(cogmentAPI::CommunicationState::NORMAL);
  auto init_data = msg.mutable_init_input();

  init_data->set_actor_name(m_name);
  init_data->set_actor_class(m_actor_class);
  init_data->set_impl_name(m_params.implementation());
  init_data->set_env_name(m_trial->env_name());
  if (m_params.has_config()) {
    init_data->mutable_config()->set_content(m_params.config().content());
  }

  write_to_stream(std::move(msg));
}

void Actor::trial_ended(std::string_view details) {
  if (!m_stream.is_valid()) {
    SPDLOG_DEBUG("Trial [{}] - Actor [{}] stream has ended: cannot end it again", m_trial->id(), m_name);
    return;
  }

  ActorStream::InputType data;
  data.set_state(cogmentAPI::CommunicationState::END);
  if (!details.empty()) {
    data.set_details(details.data(), details.size());
  }

  m_stream.write_last(std::move(data));
  SPDLOG_DEBUG("Trial [{}] - Actor [{}] 'END' sent", m_trial->id(), m_name);

  finish_stream();
}

void Actor::finish_stream() {
  if (!m_finished.exchange(true)) {
    try {
      m_stream.finish();
    }
    catch (...) {
    }
    m_finished_prom.set_value();
  }
}

}  // namespace cogment
