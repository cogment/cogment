// Copyright 2020 Artificial Intelligence Redefined <dev+cogment@ai-r.com>
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

#include "cogment/datalog/grpc_exporter.h"
#include "cogment/trial.h"

#include "spdlog/spdlog.h"

#include <stdexcept>

namespace cogment {

Grpc_datalog_exporter_base::Trial_log::Trial_log(Grpc_datalog_exporter_base* owner, Trial* trial)
    : owner_(owner), trial_(trial) {}

Grpc_datalog_exporter_base::Trial_log::~Trial_log() {
  if (output_promise_) {
    output_promise_->complete();
  }
}

void Grpc_datalog_exporter_base::Trial_log::lazy_start_stream_() {
  if (!output_promise_) {
    auto stream_reply = owner_->stub_->OnLogSample();
    auto& stream = std::get<0>(stream_reply);
    auto& reply = std::get<1>(stream_reply);

    // We'll just ignore whatever comes back from the log exporter service
    reply.finally([](auto) {});

    cogment::LogExporterSampleRequest msg;
    *msg.mutable_trial_params() = trial_->params();
    stream.push(std::move(msg));

    output_promise_ = std::move(stream);
  }
}

void Grpc_datalog_exporter_base::Trial_log::add_sample(cogment::DatalogSample data) {
  lazy_start_stream_();

  cogment::LogExporterSampleRequest msg;
  *msg.mutable_sample() = std::move(data);
  output_promise_->push(std::move(msg));
}

std::unique_ptr<Trial_log_interface> Grpc_datalog_exporter_base::begin_trial(Trial* trial) {
  return std::make_unique<Grpc_datalog_exporter::Trial_log>(this, trial);
}

Grpc_datalog_exporter::Grpc_datalog_exporter(const std::string& url) : channel(url, &work_thread), stub_impl(&channel) {
  set_stub(&stub_impl);

  spdlog::info("Sending datalog to service running at: {}", url);
}

}  // namespace cogment