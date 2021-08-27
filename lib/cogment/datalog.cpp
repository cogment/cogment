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
#include "spdlog/spdlog.h"

namespace cogment {

DatalogServiceImpl::DatalogServiceImpl(StubEntryType stub_entry) : m_stub_entry(std::move(stub_entry)) {
  SPDLOG_TRACE("DatalogServiceImpl");
}

DatalogServiceImpl::~DatalogServiceImpl() {
  SPDLOG_TRACE("~DatalogServiceImpl()");

  m_stream->WritesDone();
  m_stream->Finish();
}

void DatalogServiceImpl::start(const std::string& trial_id, const cogmentAPI::TrialParams& params) {
  if (m_stream != nullptr) {
    throw MakeException("DatalogService already started");
  }

  m_context.AddMetadata("trial-id", trial_id);
  m_stream = m_stub_entry->get_stub().OnLogSample(&m_context);

  cogmentAPI::LogExporterSampleRequest msg;
  *msg.mutable_trial_params() = params;
  m_stream->Write(msg);
}

void DatalogServiceImpl::add_sample(cogmentAPI::DatalogSample&& data) {
  if (m_stream != nullptr) {
    cogmentAPI::LogExporterSampleRequest msg;
    *msg.mutable_sample() = std::move(data);
    m_stream->Write(msg);
  }
  else {
    throw MakeException("DatalogService is not started");
  }
}

} // namespace
