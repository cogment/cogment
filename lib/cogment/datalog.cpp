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

#include "cogment/datalog.h"
#include "cogment/utils.h"

#include "spdlog/spdlog.h"

namespace cogment {
namespace {

// Sample fields
constexpr std::string_view OBSERVATION_FIELD_NAME("observation");
constexpr std::string_view ACTIONS_FIELD_NAME("actions");
constexpr std::string_view REWARDS_FIELD_NAME("rewards");
constexpr std::string_view MESSAGES_FIELD_NAME("messages");
constexpr std::string_view INFO_FIELD_NAME("info");

constexpr size_t OBSERVATIONS_FIELD = 0;
constexpr size_t ACTIONS_FIELD = 1;
constexpr size_t REWARDS_FIELD = 2;
constexpr size_t MESSAGES_FIELD = 3;
constexpr size_t INFO_FIELD = 4;

constexpr size_t NB_FIELDS = 5;

}  // namespace

DatalogServiceImpl::DatalogServiceImpl(StubEntryType stub_entry) :
    m_stub_entry(std::move(stub_entry)), m_stream_valid(false) {
  SPDLOG_TRACE("DatalogServiceImpl");
}

DatalogServiceImpl::~DatalogServiceImpl() {
  SPDLOG_TRACE("~DatalogServiceImpl()");

  if (m_stream_valid) {
    m_stream_valid = false;
    m_stream->WritesDone();
    m_stream->Finish();
  }
}

void DatalogServiceImpl::start(const std::string& trial_id, const std::string& user_id,
                               const cogmentAPI::TrialParams& params) {
  if (m_stream != nullptr) {
    throw MakeException("DatalogService already started for [{}] cannot start for [{}]", m_trial_id, trial_id);
  }
  m_trial_id = trial_id;

  static_assert(NB_BITS >= NB_FIELDS);
  const auto& exclude = params.datalog().exclude_fields();
  for (auto field : exclude) {
    to_lower_case(&field);

    if (field == OBSERVATION_FIELD_NAME) {
      m_exclude_fields.set(OBSERVATIONS_FIELD);
    }
    else if (field == ACTIONS_FIELD_NAME) {
      m_exclude_fields.set(ACTIONS_FIELD);
    }
    else if (field == REWARDS_FIELD_NAME) {
      m_exclude_fields.set(REWARDS_FIELD);
    }
    else if (field == MESSAGES_FIELD_NAME) {
      m_exclude_fields.set(MESSAGES_FIELD);
    }
    else if (field == INFO_FIELD_NAME) {
      m_exclude_fields.set(INFO_FIELD);
    }
    else {
      spdlog::warn("Trial [{}] - Datalog excluded field [{}] is not a sample log field", m_trial_id, field);
    }
  }
  spdlog::debug("Trial [{}] - Datalog excluded field [{}]", m_trial_id, m_exclude_fields.to_string());

  m_context.AddMetadata("trial-id", m_trial_id);
  m_context.AddMetadata("user-id", user_id);
  m_stream = m_stub_entry->get_stub().RunTrialDatalog(&m_context);

  if (m_stream != nullptr) {
    cogmentAPI::RunTrialDatalogInput msg;
    *msg.mutable_trial_params() = params;
    m_stream_valid = m_stream->Write(msg);
  }
}

void DatalogServiceImpl::dispatch_sample(cogmentAPI::DatalogSample&& data) {
  if (m_stream_valid) {
    cogmentAPI::RunTrialDatalogInput msg;
    *msg.mutable_sample() = std::move(data);
    m_stream_valid = m_stream->Write(msg);
  }
  else {
    if (m_stream != nullptr) {
      throw MakeException("DatalogService stream stopped");
    }
    else {
      throw MakeException("DatalogService is not started");
    }
  }
}

void DatalogServiceImpl::add_sample(cogmentAPI::DatalogSample&& sample) {
  if (m_exclude_fields.none()) {
    dispatch_sample(std::move(sample));
  }
  else {
    if (m_exclude_fields[OBSERVATIONS_FIELD]) {
      sample.clear_observations();
    }
    if (m_exclude_fields[ACTIONS_FIELD]) {
      sample.clear_actions();
    }
    if (m_exclude_fields[REWARDS_FIELD]) {
      sample.clear_rewards();
    }
    if (m_exclude_fields[MESSAGES_FIELD]) {
      sample.clear_messages();
    }
    if (m_exclude_fields[INFO_FIELD]) {
      sample.clear_info();
    }

    dispatch_sample(std::move(sample));
  }
}

}  // namespace cogment
