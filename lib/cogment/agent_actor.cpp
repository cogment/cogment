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

Agent::Agent(Trial* owner, const std::string& in_actor_name, const ActorClass* actor_class, const std::string& impl,
             StubEntryType stub_entry, std::optional<std::string> config_data) :
    Actor(owner, in_actor_name, actor_class),
    m_stub_entry(std::move(stub_entry)),
    m_config_data(std::move(config_data)),
    m_impl(impl),
    m_last_sent(false),
    m_init_completed(false) {
  SPDLOG_TRACE("Agent(): [{}] [{}] [{}]", trial()->id(), actor_name(), impl);

  m_context.AddMetadata("trial-id", trial()->id());
  m_context.AddMetadata("actor-name", actor_name());

  // TODO: Move this out of the constructor and in its own thread to wait for the stream init without blocking.
  //       But then we'll need to synchonize with other parts.
  m_stream = m_stub_entry->get_stub().RunTrial(&m_context);

  m_incoming_thread = std::thread([this]() {
    try {
      cogmentAPI::ServiceActorRunOutput data;
      while(m_stream->Read(&data)) {
        process_incoming_data(std::move(data));
      }
      SPDLOG_TRACE("Trial [{}]: Service actor [{}] finished reading stream", trial()->id(), actor_name());
    }
    catch(const std::exception& exc) {
      spdlog::error("Trial [{}] - Service actor [{}]: Error reading stream [{}]", trial()->id(), actor_name(), exc.what());
    }
    catch(...) {
      spdlog::error("Trial [{}] - Service actor [{}]: Unknown exception reading stream", trial()->id(), actor_name());
    }
  });
}

Agent::~Agent() {
  SPDLOG_TRACE("~Agent(): [{}] [{}]", trial()->id(), actor_name());

  m_stream->Finish();
  if (!m_init_completed) {
    m_init_prom.set_value();
  }

  m_incoming_thread.join();
}

void Agent::trial_ended(std::string_view details) {
  cogmentAPI::ServiceActorRunInput data;
  data.set_state(cogmentAPI::CommunicationState::END);
  if (details.size() > 0) {
    data.set_details(details.data(), details.size());
  }
  m_stream->Write(data);
  m_stream->WritesDone();
}

void Agent::process_communication_state(cogmentAPI::CommunicationState in_state, const std::string* details) {
  switch(in_state) {
    case cogmentAPI::UNKNOWN_COM_STATE:
      if (details != nullptr) {
        throw MakeException<std::invalid_argument>("Unknown communication state: [%s]", details->c_str());
      } else {
        throw MakeException<std::invalid_argument>("Unknown communication state");
      }
      break;
    case cogmentAPI::CommunicationState::NORMAL:
      if (details != nullptr) {
        spdlog::info("Communication details received from service actor: [{}]", *details);
      } else {
        spdlog::warn("No data in normal communication received from service actor");
      }
      break;
    case cogmentAPI::CommunicationState::HEARTBEAT:
      if (details != nullptr) {
        spdlog::info("Heartbeat requested from service actor: [{}]", *details);
      }
      // TODO : manage heartbeats
      break;
    case cogmentAPI::CommunicationState::LAST:
      if (details != nullptr) {
        spdlog::error("Unexpected communication state (LAST) received from service actor: [{}]", *details);
      } else {
        spdlog::error("Unexpected communication state (LAST) received from service actor");
      }
      break;
    case cogmentAPI::CommunicationState::LAST_ACK:
      if (!m_last_sent) {
        if (details != nullptr) {
          spdlog::error("Unexpected reception of communication state (LAST_ACK) from service actor: [{}]", *details);
        } else {
          spdlog::error("Unexpected reception of communication state (LAST_ACK) from service actor");
        }
      }
      ack_last();
      break;
    case cogmentAPI::CommunicationState::END:
      // TODO: Decide what to do about "END" received from actors
      if (details != nullptr) {
        spdlog::error("Unexpected communication state (END) received from service actor: [{}]", *details);
      } else {
        spdlog::error("Unexpected communication state (END) received from service actor");
      }
      break;

    default:
      throw MakeException<std::invalid_argument>("Invalid communication state: [%d]", 
                                                 static_cast<int>(in_state));
      break;
  }

}

void Agent::process_incoming_data(cogmentAPI::ServiceActorRunOutput&& data) {
  SPDLOG_TRACE("Trial [{}] - Service actor [{}]: Processing incoming data", trial()->id(), actor_name());

  const auto state = data.state();
  const auto data_case = data.data_case();
  switch (data_case) {
  case cogmentAPI::ServiceActorRunOutput::kInitOutput: {
    if (state != cogmentAPI::CommunicationState::NORMAL) {
      throw MakeException<std::invalid_argument>("'init_output' received from service actor on non-normal communication");
    }
    SPDLOG_DEBUG("Trial [{}] - Service actor [{}] init complete", trial()->id(), actor_name());

    // TODO: Should we have a timer for this reply?
    m_init_prom.set_value();
    m_init_completed = true;
    break;
  }
  case cogmentAPI::ServiceActorRunOutput::kAction: {
    if (state != cogmentAPI::CommunicationState::NORMAL) {
      throw MakeException<std::invalid_argument>("'action' received on from service actor on non-normal communication");
    }
    trial()->actor_acted(actor_name(), data.action());
    break;
  }
  case cogmentAPI::ServiceActorRunOutput::kReward: {
    if (state != cogmentAPI::CommunicationState::NORMAL) {
      throw MakeException<std::invalid_argument>("'reward' received from service actor on non-normal communication");
    }
    trial()->reward_received(data.reward(), actor_name());
    break;
  }
  case cogmentAPI::ServiceActorRunOutput::kMessage: {
    if (state != cogmentAPI::CommunicationState::NORMAL) {
      throw MakeException<std::invalid_argument>("'message' received from service actor on non-normal communication");
    }
    trial()->message_received(data.message(), actor_name());
    break;
  }
  case cogmentAPI::ServiceActorRunOutput::kDetails: {
    process_communication_state(state, &data.details());
    break;
  }
  case cogmentAPI::ServiceActorRunOutput::DATA_NOT_SET: {
    process_communication_state(state, nullptr);
    break;
  }
  default: {
    throw MakeException<std::invalid_argument>("Unknown communication data [%d]", static_cast<int>(data_case));
    break;
  }
  }
}

std::future<void> Agent::init() {
  SPDLOG_TRACE("Trial [{}] - Agent::init(): [{}]", trial()->id(), actor_name());

  cogmentAPI::ServiceActorRunInput input;
  input.set_state(cogmentAPI::CommunicationState::NORMAL);

  cogmentAPI::ServiceActorInitialInput init_input;
  init_input.set_actor_class(actor_class()->name);
  init_input.set_impl_name(m_impl);
  if (m_config_data) {
    init_input.mutable_config()->set_content(m_config_data.value());
  }
  *(input.mutable_init_input()) = std::move(init_input);
  m_stream->Write(input);

  return m_init_prom.get_future();
}

void Agent::dispatch_observation(cogmentAPI::Observation&& observation) {
  cogmentAPI::ServiceActorRunInput input;
  input.set_state(cogmentAPI::CommunicationState::NORMAL);
  *(input.mutable_observation()) = std::move(observation);
  m_stream->Write(input);
}

void Agent::dispatch_reward(cogmentAPI::Reward&& reward) {
  cogmentAPI::ServiceActorRunInput input;
  input.set_state(cogmentAPI::CommunicationState::NORMAL);
  *(input.mutable_reward()) = std::move(reward);
  m_stream->Write(input);
}

void Agent::dispatch_message(cogmentAPI::Message&& message) {
  cogmentAPI::ServiceActorRunInput input;
  input.set_state(cogmentAPI::CommunicationState::NORMAL);
  *(input.mutable_message()) = std::move(message);
  m_stream->Write(input);
}

void Agent::dispatch_final_data(cogmentAPI::ActorPeriodData&& data) {
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
    cogmentAPI::ServiceActorRunInput input;
    input.set_state(cogmentAPI::CommunicationState::LAST);
    m_stream->Write(input);
    m_last_sent = true;

    dispatch_observation(std::move(*last_obs));
  }
}

bool Agent::is_active() const {
  // Actors driven by agent services are always active since
  // they are driven by the orchestrator itself.
  return true;
}

}  // namespace cogment
