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
    m_stream_end_fut(m_stream_end_prom.get_future()),
    m_impl(impl) {
  SPDLOG_TRACE("Agent(): [{}] [{}] [{}]", trial()->id(), actor_name(), impl);

  grpc_metadata trial_header;
  trial_header.key = grpc_slice_from_static_string("trial-id");
  trial_header.value = grpc_slice_from_copied_string(trial()->id().c_str());

  grpc_metadata actor_header;
  actor_header.key = grpc_slice_from_static_string("actor-name");
  actor_header.value = grpc_slice_from_copied_string(actor_name().c_str());

  m_headers = {trial_header, actor_header};
  m_options.headers = &m_headers;
}

Agent::~Agent() {
  SPDLOG_TRACE("~Agent(): [{}] [{}]", trial()->id(), actor_name());

  if (m_outgoing_observations) {
    m_outgoing_observations->complete();
    m_stream_end_fut.wait();
  }
}

aom::Future<void> Agent::init() {
  SPDLOG_TRACE("Agent::init(): [{}] [{}]", trial()->id(), actor_name());

  cogment::AgentStartRequest req;

  req.set_impl_name(m_impl);

  if (m_config_data) {
    req.mutable_config()->set_content(m_config_data.value());
  }

  for (const auto& actor : trial()->actors()) {
    auto actor_in_trial = req.add_actors_in_trial();
    actor_in_trial->set_actor_class(actor->actor_class()->name);
    actor_in_trial->set_name(actor->actor_name());
  }

  return m_stub_entry->get_stub().OnStart(req, m_options).then([this](auto rep) {
    (void)rep;
    (void)this;
    SPDLOG_DEBUG("Agent init start complete: [{}] [{}]", trial()->id(), actor_name());
  });
}

void Agent::dispatch_observation(cogment::Observation&& observation) {
  lazy_start_decision_stream();

  // We serialize the observations to prevent long lags where
  // rewards/messages arrive much later and cause errors.
  m_stub_entry->serialize([this, obs = std::move(observation)]() {
    cogment::AgentObservationRequest req;
    *req.mutable_observation() = std::move(obs);

    m_outgoing_observations->push(std::move(req));
  });
}

void Agent::dispatch_final_data(cogment::ActorPeriodData&& data) {
  ::cogment::AgentEndRequest req;
  *(req.mutable_final_data()) = std::move(data);

  m_stub_entry->get_stub().OnEnd(req, m_options);
}

void Agent::lazy_start_decision_stream() {
  if (!m_outgoing_observations) {
    auto stream = m_stub_entry->get_stub().OnObservation(m_options);

    m_outgoing_observations = std::move(std::get<0>(stream));
    auto incoming_actions = std::move(std::get<1>(stream));

    std::weak_ptr trial_weak = trial()->get_shared();
    incoming_actions
        .for_each([this, trial_weak](auto rep) {
          auto trial = trial_weak.lock();
          if (trial != nullptr && trial->state() != Trial::InternalState::ended) {
            trial->actor_acted(actor_name(), rep.action());
            for (auto& rew : rep.rewards()) {
              trial->reward_received(rew, actor_name());
            }
            for (auto& message : rep.messages()) {
              trial->message_received(message, actor_name());
            }
          }
        })
        .finally([this](auto) {
          SPDLOG_TRACE("Trial: Finalized service actor [{}] stream", actor_name());
          m_stream_end_prom.set_value();
        });
  }
}

bool Agent::is_active() const {
  // Actors driven by agent services are always active since
  // they are driven by the orchestrator itself.
  return true;
}

void Agent::dispatch_reward(cogment::Reward&& reward) {
  m_stub_entry->serialize([this, rew = std::move(reward)]() {
    cogment::AgentRewardRequest req;
    *req.mutable_reward() = std::move(rew);

    m_stub_entry->get_stub().OnReward(req, m_options).get();
  });
}

void Agent::dispatch_message(cogment::Message&& message) {
  m_stub_entry->serialize([this, msg = std::move(message)]() {
    cogment::AgentMessageRequest req;
    auto new_msg = req.add_messages();
    *new_msg = std::move(msg);

    m_stub_entry->get_stub().OnMessage(req, m_options).get();
  });
}

}  // namespace cogment
