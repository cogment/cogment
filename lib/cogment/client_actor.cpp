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

#include "spdlog/spdlog.h"

namespace cogment {
Client_actor::Client_actor(Trial* owner, const std::string& actor_name, const ActorClass* actor_class,
                           std::optional<std::string> config_data) :
    Actor(owner, actor_name, actor_class),
    m_joined(false),
    m_config_data(std::move(config_data)),
    m_stream(nullptr),
    m_finished_fut(m_finished_prom.get_future()),
    m_incoming_stream_status(grpc::Status::OK) {
  SPDLOG_TRACE("Client_actor(): [{}] [{}]", trial()->id(), actor_name);
}

Client_actor::~Client_actor() {
  SPDLOG_TRACE("~Client_actor(): [{}] [{}]", trial()->id(), actor_name());

  finish_stream();

  if (m_incoming_thread.joinable()) {
    m_incoming_thread.join();
  }
}

void Client_actor::trial_ended(std::string_view) {}

std::future<void> Client_actor::init() {
  SPDLOG_TRACE("Client_actor::init(): [{}] [{}]", trial()->id(), actor_name());
  // Client actors are ready once a client has connected to it.
  return m_ready_prom.get_future();
}

bool Client_actor::is_active() const { return m_joined; }

std::optional<std::string> Client_actor::join() {
  m_joined = true;
  m_ready_prom.set_value();

  spdlog::debug("Trial [{}] - Actor [{}] has joined the trial", trial()->id(), actor_name());
  return m_config_data;
}

grpc::Status Client_actor::bind(StreamType* stream) {
  SPDLOG_TRACE("Trial [{}] - Actor [{}] binding", trial()->id(), actor_name());

  if (m_stream != nullptr) {
    throw MakeException("Trial [%s] - Actor [%s] already bound", trial()->id().c_str(), actor_name().c_str());
  }

  m_stream = stream;
  m_incoming_thread = std::thread([this]() {
    try {
      cogmentAPI::TrialActionRequest data;
      while(m_stream->Read(&data)) {
        trial()->actor_acted(actor_name(), data.action());
      }
      SPDLOG_TRACE("Trial [{}]: Client actor [{}] stream finalized", trial()->id(), actor_name());
    }
    catch(const std::exception& exc) {
      m_incoming_stream_status = MakeErrorStatus("Trial [%s] - Client actor [%s] error reading stream [%s]", 
                                                 trial()->id().c_str(), actor_name().c_str(), exc.what());
    }
    catch(...) {
      m_incoming_stream_status = MakeErrorStatus("Trial [%s] - Client actor [%s] unknown exception reading stream", 
                                                 trial()->id().c_str(), actor_name().c_str());
    }
    finish_stream();
  });

  // TODO: Fix delayed error report --> If the reading of the stream has a problem, the stream will not be finalized
  //       (and report any error) until the actor is destroyed.

  m_finished_fut.get();  // Cannot be "wait()" because of "finish_stream()"
  return m_incoming_stream_status;
}

void Client_actor::dispatch_observation(cogmentAPI::Observation&& obs) {
  cogmentAPI::TrialActionReply req;
  req.set_final_data(false);
  auto new_obs = req.mutable_data()->add_observations();
  *new_obs = std::move(obs);

  m_stream->Write(req);
}

void Client_actor::dispatch_final_data(cogmentAPI::ActorPeriodData&& data) {
  cogmentAPI::TrialActionReply req;
  req.set_final_data(true);
  *(req.mutable_data()) = std::move(data);

  m_stream->Write(req);

  ack_last();  // Fake it until we switch client actors to api2.0
}

void Client_actor::dispatch_reward(cogmentAPI::Reward&& reward) {
  cogmentAPI::TrialActionReply req;
  auto new_reward = req.mutable_data()->add_rewards();
  *new_reward = std::move(reward);

  m_stream->Write(req);
}

void Client_actor::dispatch_message(cogmentAPI::Message&& message) {
  cogmentAPI::TrialActionReply req;
  auto new_mess = req.mutable_data()->add_messages();
  *new_mess = std::move(message);

  m_stream->Write(req);
}

// TODO: This is a horrible kludge to not delay error report in incoming stream error (or termination).
//       It needs to be looked at when we have time (or when we update the grpc use).
void Client_actor::finish_stream() {
  const std::lock_guard<std::mutex> lg(m_finishing_mutex);

  try {
    if (m_finished_fut.valid()) {  // To try to prevent exceptions most of the time
      m_finished_prom.set_value();
    }
  }
  catch(...) {}

}

}  // namespace cogment
