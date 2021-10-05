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

#include "cogment/agent_actor.h"
#include "cogment/trial.h"

#include "spdlog/spdlog.h"

namespace cogment {

ServiceActor::ServiceActor(Trial* owner, const std::string& in_actor_name, const ActorClass* actor_class, const std::string& impl,
             StubEntryType stub_entry, std::optional<std::string> config_data) :
    Actor(owner, in_actor_name, actor_class, impl, config_data),
    m_stub_entry(std::move(stub_entry)),
    m_stream_valid(false),
    m_init_completed(false) {
  SPDLOG_TRACE("ServiceActor(): [{}] [{}] [{}]", trial()->id(), actor_name(), impl);

  m_context.AddMetadata("trial-id", trial()->id());

  // TODO: Move this out of the constructor (maybe in init) and in its own thread to wait
  //       for the stream init without blocking. But then we'll need to synchonize with other parts.
  m_stream = m_stub_entry->get_stub().RunTrial(&m_context);
  m_stream_valid = true;

  m_incoming_thread = std::thread([this]() {
    try {
      cogmentAPI::ActorRunTrialOutput data;
      while(m_stream->Read(&data)) {
        process_incoming_data(std::move(data));
        if (!m_stream_valid) {
          break;
        }
      }
      SPDLOG_TRACE("Trial [{}] - Service actor [{}] finished reading stream (valid [{}])", trial()->id(), actor_name(), m_stream_valid);
    }
    // TODO: Look into a way to cancel the stream immediately here (in case of exception in process_incoming_data)
    catch(const std::exception& exc) {
      spdlog::error("Trial [{}] - Service actor [{}]: Error reading stream [{}]", trial()->id(), actor_name(), exc.what());
    }
    catch(...) {
      spdlog::error("Trial [{}] - Service actor [{}]: Unknown exception reading stream", trial()->id(), actor_name());
    }
  });
}

ServiceActor::~ServiceActor() {
  SPDLOG_TRACE("~ServiceActor(): [{}] [{}]", trial()->id(), actor_name());

  if (m_stream_valid) {
    m_stream->Finish();
  }

  if (!m_init_completed) {
    m_init_prom.set_value();
  }

  m_incoming_thread.join();
}

void ServiceActor::write_to_stream(cogmentAPI::ActorRunTrialInput&& data) {
  const std::lock_guard<std::mutex> lg(m_writing);
  if (m_stream_valid) {
    m_stream->Write(std::move(data));
  }
  else {
    throw MakeException("Trial [%s] - Actor [%s] stream has closed", trial()->id().c_str(), actor_name().c_str());
  }
}

void ServiceActor::trial_ended(std::string_view details) {
  const std::lock_guard<std::mutex> lg(m_writing);

  cogmentAPI::ActorRunTrialInput data;
  data.set_state(cogmentAPI::CommunicationState::END);
  if (details.size() > 0) {
    data.set_details(details.data(), details.size());
  }
  m_stream->Write(std::move(data));
  SPDLOG_DEBUG("Trial [{}] - Actor [{}] 'END' sent", trial()->id(), actor_name());

  m_stream->WritesDone();
  m_stream->Finish();
  m_stream_valid = false;
}

void ServiceActor::process_incoming_data(cogmentAPI::ActorRunTrialOutput&& data) {
  SPDLOG_TRACE("Trial [{}] - Service actor [{}]: Processing incoming data", trial()->id(), actor_name());

  const auto state = data.state();
  const auto data_case = data.data_case();
  switch (data_case) {
  case cogmentAPI::ActorRunTrialOutput::kInitOutput: {
    if (state == cogmentAPI::CommunicationState::NORMAL) {
      // TODO: Should we have a timer for this reply?
      m_init_prom.set_value();
      m_init_completed = true;
    }
    else {
      throw MakeException<std::invalid_argument>("'init_output' received from service actor on non-normal communication");
    }
    SPDLOG_DEBUG("Trial [{}] - Service actor [{}] init complete", trial()->id(), actor_name());
    break;
  }

  case cogmentAPI::ActorRunTrialOutput::kAction: {
    if (state == cogmentAPI::CommunicationState::NORMAL) {
      trial()->actor_acted(actor_name(), data.action());
    }
    else {
      throw MakeException<std::invalid_argument>("'action' received on from service actor on non-normal communication");
    }
    break;
  }

  case cogmentAPI::ActorRunTrialOutput::kReward: {
    if (state == cogmentAPI::CommunicationState::NORMAL) {
      trial()->reward_received(data.reward(), actor_name());
    }
    else {
      throw MakeException<std::invalid_argument>("'reward' received from service actor on non-normal communication");
    }
    break;
  }

  case cogmentAPI::ActorRunTrialOutput::kMessage: {
    if (state == cogmentAPI::CommunicationState::NORMAL) {
      trial()->message_received(data.message(), actor_name());
    }
    else {
      throw MakeException<std::invalid_argument>("'message' received from service actor on non-normal communication");
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

std::future<void> ServiceActor::init() {
  SPDLOG_TRACE("Trial [{}] - ServiceActor::init(): [{}]", trial()->id(), actor_name());

  cogmentAPI::ActorRunTrialInput msg;
  msg.set_state(cogmentAPI::CommunicationState::NORMAL);

  cogmentAPI::ActorInitialInput init_input;
  init_input.set_actor_name(actor_name());
  init_input.set_actor_class(actor_class()->name);
  init_input.set_impl_name(impl());
  if (config()) {
    init_input.mutable_config()->set_content(config().value());
  }
  *(msg.mutable_init_input()) = std::move(init_input);
  write_to_stream(std::move(msg));

  return m_init_prom.get_future();
}

void ServiceActor::dispatch_observation(cogmentAPI::Observation&& observation) {
  cogmentAPI::ActorRunTrialInput msg;
  msg.set_state(cogmentAPI::CommunicationState::NORMAL);
  *(msg.mutable_observation()) = std::move(observation);
  write_to_stream(std::move(msg));
}

void ServiceActor::dispatch_reward(cogmentAPI::Reward&& reward) {
  cogmentAPI::ActorRunTrialInput msg;
  msg.set_state(cogmentAPI::CommunicationState::NORMAL);
  *(msg.mutable_reward()) = std::move(reward);
  write_to_stream(std::move(msg));
}

void ServiceActor::dispatch_message(cogmentAPI::Message&& message) {
  cogmentAPI::ActorRunTrialInput msg;
  msg.set_state(cogmentAPI::CommunicationState::NORMAL);
  *(msg.mutable_message()) = std::move(message);
  write_to_stream(std::move(msg));
}

void ServiceActor::dispatch_final_data(cogmentAPI::ActorPeriodData&& data) {
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

bool ServiceActor::is_active() const {
  // Actors driven by agent services are always active since
  // they are driven by the orchestrator itself.
  return true;
}

}  // namespace cogment
