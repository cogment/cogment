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

#include "cogment/datalog/grpc_exporter.h"
#include "cogment/trial.h"

#include "spdlog/spdlog.h"

#include <stdexcept>

namespace cogment {

GrpcDatalogExporterBase::Trial_log::Trial_log(GrpcDatalogExporterBase* owner, const Trial* trial) :
    m_owner(owner), m_trial(trial) {}

GrpcDatalogExporterBase::Trial_log::~Trial_log() {
  if (m_output_promise) {
    m_output_promise->complete();
  }
}

void GrpcDatalogExporterBase::Trial_log::m_lazy_start_stream() {
  if (!m_output_promise) {
    grpc_metadata trial_header;
    trial_header.key = grpc_slice_from_static_string("trial-id");
    trial_header.value = grpc_slice_from_copied_string(m_trial->id().c_str());
    m_headers = {trial_header};
    m_options.headers = &m_headers;

    auto stream_reply = m_owner->m_stub->OnLogSample(m_options);
    auto stream = std::move(std::get<0>(stream_reply));
    m_reply = std::move(std::get<1>(stream_reply));

    // We'll just ignore whatever comes back from the log exporter service.
    // TODO: It is required to bypass a probable bug in easy_grpc that expects a stream to be "used".
    m_reply.for_each([](auto) {}).finally([](auto) {});

    cogment::LogExporterSampleRequest msg;
    *msg.mutable_trial_params() = m_trial->params();
    stream.push(std::move(msg));

    m_output_promise = std::move(stream);
  }
}

void GrpcDatalogExporterBase::Trial_log::add_sample(cogment::DatalogSample&& data) {
  m_lazy_start_stream();

  cogment::LogExporterSampleRequest msg;
  *msg.mutable_sample() = std::move(data);
  m_output_promise->push(std::move(msg));
}

std::unique_ptr<TrialLogInterface> GrpcDatalogExporterBase::start_log(const Trial* trial) {
  return std::make_unique<GrpcDatalogExporter::Trial_log>(this, trial);
}

GrpcDatalogExporter::GrpcDatalogExporter(const std::string& url) :
    m_channel(url, &m_work_thread), m_stub_impl(&m_channel) {
  set_stub(&m_stub_impl);
  spdlog::info("Sending datalog to service running at: {}", url);
}

}  // namespace cogment