// Copyright 2023 AI Redefined Inc. <dev+cogment@ai-r.com>
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

#ifndef COGMENT_ORCHESTRATOR_CALLBACK_SPDLOG_SINK_H
#define COGMENT_ORCHESTRATOR_CALLBACK_SPDLOG_SINK_H

#include "orchestrator.h"

#include "spdlog/sinks/base_sink.h"

#include <chrono>
#include <vector>

namespace cogment {

template <typename Mutex>
class CallbackSpdLogSink : public spdlog::sinks::base_sink<Mutex> {
public:
  CallbackSpdLogSink(void* ctx, CogmentOrchestratorLogger logger) :
      spdlog::sinks::base_sink<Mutex>(), m_ctx(ctx), m_logger(logger) {}

protected:
  void sink_it_(const spdlog::details::log_msg& msg) override {
    std::string logger_name_str = fmt::to_string(msg.logger_name);
    std::string payload_str = fmt::to_string(msg.payload);
    m_logger(m_ctx, logger_name_str.c_str(), msg.level, std::chrono::system_clock::to_time_t(msg.time), msg.thread_id,
             msg.source.filename, msg.source.line, msg.source.funcname, payload_str.c_str());
  }

  void flush_() override {}

private:
  void* m_ctx;
  CogmentOrchestratorLogger m_logger;
};
}  // namespace cogment

#endif
