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

#include "cogment/client_actor.h"
#include "cogment/trial.h"
#include "cogment/utils.h"

#include "spdlog/spdlog.h"

namespace cogment {
namespace {

bool get_init_data(ClientActor::StreamType* stream, cogmentAPI::ActorInitialOutput* out) {
  SPDLOG_TRACE("Client actor get_init_data");

  // TODO: Limit the time to wait for init data
  cogmentAPI::ActorRunTrialOutput data;
  while(stream->Read(&data)) {
    const auto state = data.state();
    const auto data_case = data.data_case();

    if (state == cogmentAPI::CommunicationState::NORMAL) {
      if (data_case == cogmentAPI::ActorRunTrialOutput::kInitOutput) {
        *out = std::move(data.init_output());
        return true;
      }
      else {
        throw MakeException("Data [%d] received from client actor before init data", static_cast<int>(data_case));
      }
    }
    else if (state == cogmentAPI::CommunicationState::HEARTBEAT) {
      if (data_case == cogmentAPI::ActorRunTrialOutput::kDetails) {
        spdlog::info("Heartbeat requested from client actor: [{}]", data.details());
      }
      cogmentAPI::ActorRunTrialInput msg;
      msg.set_state(cogmentAPI::CommunicationState::HEARTBEAT);
      stream->Write(std::move(msg));
    }
    else if (state == cogmentAPI::CommunicationState::LAST) {
      throw MakeException("Unexpected reception of communication state (LAST) from client actor");
    }
    else if (state == cogmentAPI::CommunicationState::LAST_ACK) {
      throw MakeException("Unexpected reception of communication state (LAST_ACK) from client actor");
    }
    else if (state == cogmentAPI::CommunicationState::END) {
      if (data_case == cogmentAPI::ActorRunTrialOutput::kDetails) {
        spdlog::error("Unexpected end of communication (END) from client actor: [{}]", data.details());
      }
      else {
        spdlog::error("Unexpected end of communication (END) from client actor");
      }
      break;
    }
    else {
      throw MakeException("Unknown communication state [%d] received from client actor", static_cast<int>(state));
    }
  }

  return false;
}

}  // namespace


grpc::Status ClientActor::run_an_actor(std::weak_ptr<Trial> trial_weak, StreamType* stream) {
  SPDLOG_TRACE("Client actor run_an_actor");

  cogmentAPI::ActorInitialOutput init_data;
  bool valid = false;
  try {
    valid = get_init_data(stream, &init_data);
  }
  catch(const std::exception& exc) {
    auto trial = trial_weak.lock();
    if (trial != nullptr) {
      throw MakeException("Trial [%s] - %s", trial->id().c_str(), exc.what());
    }
    else {
      throw;
    }
  }

  auto trial = trial_weak.lock();
  if (valid && trial != nullptr) {
    std::string actor_name;
    const auto slot_case = init_data.slot_selection_case();
    if (slot_case == cogmentAPI::ActorInitialOutput::kActorName) {
      actor_name = init_data.actor_name();
    }
    std::string actor_class;
    if (slot_case == cogmentAPI::ActorInitialOutput::kActorClass) {
      actor_class = init_data.actor_class();
    }
    auto actor = trial->get_join_candidate(actor_name, actor_class);
    init_data.Clear();
    trial.reset();

    return actor->run(stream);  // Blocking
  }
  else {
    return grpc::Status::OK;
  }
}


ClientActor::ClientActor(Trial* owner, const std::string& actor_name, const ActorClass* actor_class, const std::string& impl,
                           std::optional<std::string> config_data) :
    Actor(owner, actor_name, actor_class, impl, config_data),
    m_stream(nullptr),
    m_stream_valid(false),
    m_finished_fut(m_finished_prom.get_future()),
    m_incoming_stream_status(grpc::Status::OK) {
  SPDLOG_TRACE("ClientActor(): [{}] [{}]", trial()->id(), actor_name);
}

ClientActor::~ClientActor() {
  SPDLOG_TRACE("~ClientActor(): [{}] [{}]", trial()->id(), actor_name());

  finish_stream();

  if (m_incoming_thread.joinable()) {
    m_incoming_thread.join();
  }
}

void ClientActor::write_to_stream(cogmentAPI::ActorRunTrialInput&& data) {
  const std::lock_guard<std::mutex> lg(m_writing);
  if (m_stream_valid) {
    m_stream->Write(std::move(data));
  }
  else {
    throw MakeException("Trial [%s] - Actor [%s] stream has closed", trial()->id().c_str(), actor_name().c_str());
  }
}

void ClientActor::trial_ended(std::string_view details) {
  const std::lock_guard<std::mutex> lg(m_writing);

  cogmentAPI::ActorRunTrialInput data;
  data.set_state(cogmentAPI::CommunicationState::END);
  if (!details.empty()) {
    data.set_details(details.data(), details.size());
  }
  m_stream->Write(std::move(data));
  SPDLOG_DEBUG("Trial [{}] - Actor [{}] 'END' sent", trial()->id(), actor_name());

  finish_stream();
}

std::future<void> ClientActor::init() {
  SPDLOG_TRACE("ClientActor::init(): [{}] [{}]", trial()->id(), actor_name());
  return m_ready_prom.get_future();
}

bool ClientActor::is_active() const {
  const std::lock_guard<std::mutex> lg(m_active_mutex);
  return (m_stream != nullptr);
}

grpc::Status ClientActor::run(StreamType* stream) {
  SPDLOG_TRACE("Trial [{}] - Actor [{}] run", trial()->id(), actor_name());

  {
    const std::lock_guard<std::mutex> lg(m_active_mutex);
    if (m_stream != nullptr) {
      throw MakeException("Trial [%s] - Actor [%s] already running", trial()->id().c_str(), actor_name().c_str());
    }
    m_stream = stream;
    m_stream_valid = true;
  }

  // Init response
  cogmentAPI::ActorRunTrialInput init_response;
  init_response.set_state(cogmentAPI::CommunicationState::NORMAL);
  cogmentAPI::ActorInitialInput init_data;
  init_data.set_actor_name(actor_name());
  init_data.set_actor_class(actor_class()->name);
  init_data.set_impl_name(impl());
  if (config()) {
    init_data.mutable_config()->set_content(config().value());
  }
  *(init_response.mutable_init_input()) = std::move(init_data);

  write_to_stream(std::move(init_response));

  // Run
  m_incoming_thread = std::thread([this]() {
    try {
      m_ready_prom.set_value();
      spdlog::debug("Trial [{}] - Actor [{}] is active", trial()->id(), actor_name());

      cogmentAPI::ActorRunTrialOutput data;
      while(m_stream->Read(&data)) {
        process_incoming_data(std::move(data));
        if (!m_stream_valid) {
          break;
        }
      }
      SPDLOG_TRACE("Trial [{}] - Client actor [{}] finished reading stream (valid [{}])", trial()->id(), actor_name(), m_stream_valid);
    }
    catch(const std::exception& exc) {
      m_incoming_stream_status = MakeErrorStatus("Trial [%s] - Client actor [%s] error reading run stream [%s]", 
                                                 trial()->id().c_str(), actor_name().c_str(), exc.what());
    }
    catch(...) {
      m_incoming_stream_status = MakeErrorStatus("Trial [%s] - Client actor [%s] unknown exception reading run stream", 
                                                 trial()->id().c_str(), actor_name().c_str());
    }
    finish_stream();
  });

  m_finished_fut.get();  // Cannot be "wait()" because of "finish_stream()"
  return m_incoming_stream_status;
}

// TODO: This is a kludge to not delay error report in incoming stream error (or termination).
//       It needs to be looked at when we have time (or when we update the grpc use to async?).
void ClientActor::finish_stream() {
  const std::lock_guard<std::mutex> lg(m_finishing_mutex);

  m_stream_valid = false;
  try {
    if (m_finished_fut.valid()) {  // To try to prevent exceptions most of the time
      m_finished_prom.set_value();
    }
  }
  catch(...) {}
}

void ClientActor::process_incoming_data(cogmentAPI::ActorRunTrialOutput&& data) {
  SPDLOG_TRACE("Trial [{}] - Client actor [{}]: Processing incoming data", trial()->id(), actor_name());

  const auto state = data.state();
  const auto data_case = data.data_case();
  switch (data_case) {
  case cogmentAPI::ActorRunTrialOutput::kInitOutput: {
    throw MakeException("Trial [%s] - Actor [%s] unexpectedly repeated init data");
    break;
  }

  case cogmentAPI::ActorRunTrialOutput::kAction: {
    if (state == cogmentAPI::CommunicationState::NORMAL) {
      trial()->actor_acted(actor_name(), data.action());
    }
    else {
      throw MakeException<std::invalid_argument>("'action' received from client actor on non-normal communication");
    }
    break;
  }

  case cogmentAPI::ActorRunTrialOutput::kReward: {
    if (state == cogmentAPI::CommunicationState::NORMAL) {
      trial()->reward_received(data.reward(), actor_name());
    }
    else {
      throw MakeException<std::invalid_argument>("'reward' received from client actor on non-normal communication");
    }
    break;
  }

  case cogmentAPI::ActorRunTrialOutput::kMessage: {
    if (state == cogmentAPI::CommunicationState::NORMAL) {
      trial()->message_received(data.message(), actor_name());
    }
    else {
      throw MakeException<std::invalid_argument>("'message' received from client actor on non-normal communication");
    }
    break;
  }

  case cogmentAPI::ActorRunTrialOutput::kDetails: {
    process_incoming_state(state, &data.details());
    break;
  }

  case cogmentAPI::ActorRunTrialOutput::DATA_NOT_SET: {
    process_incoming_state(state, nullptr);
    break;
  }

  default: {
    throw MakeException<std::invalid_argument>("Unknown communication data [%d]", static_cast<int>(data_case));
    break;
  }
  }
}

void ClientActor::dispatch_observation(cogmentAPI::Observation&& observation) {
  cogmentAPI::ActorRunTrialInput msg;
  msg.set_state(cogmentAPI::CommunicationState::NORMAL);
  *(msg.mutable_observation()) = std::move(observation);
  write_to_stream(std::move(msg));
}

void ClientActor::dispatch_reward(cogmentAPI::Reward&& reward) {
  cogmentAPI::ActorRunTrialInput msg;
  msg.set_state(cogmentAPI::CommunicationState::NORMAL);
  *(msg.mutable_reward()) = std::move(reward);
  write_to_stream(std::move(msg));
}

void ClientActor::dispatch_message(cogmentAPI::Message&& message) {
  cogmentAPI::ActorRunTrialInput msg;
  msg.set_state(cogmentAPI::CommunicationState::NORMAL);
  *(msg.mutable_message()) = std::move(message);
  write_to_stream(std::move(msg));
}

void ClientActor::dispatch_final_data(cogmentAPI::ActorPeriodData&& data) {
  for (auto& rew : *data.mutable_rewards()) {
    dispatch_reward(std::move(rew));
  }
  for (auto& msg : *data.mutable_messages()) {
    dispatch_message(std::move(msg));
  }

  // Send 'LAST' only before the last observation to work with SDK auto_ack
  cogmentAPI::Observation* last_obs = nullptr;
  for (auto& obs : *data.mutable_observations()) {
    if (last_obs != nullptr) {
      dispatch_observation(std::move(*last_obs));
    }
    last_obs = &obs;
  }
  if (last_obs != nullptr) {
    cogmentAPI::ActorRunTrialInput msg;
    msg.set_state(cogmentAPI::CommunicationState::LAST);
    write_to_stream(std::move(msg));
    SPDLOG_DEBUG("Trial [{}] - Actor [{}] 'LAST' sent", trial()->id(), actor_name());
    last_sent();

    dispatch_observation(std::move(*last_obs));
  }
}

}  // namespace cogment
