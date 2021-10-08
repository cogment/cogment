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

#include "cogment/utils.h"

#ifdef __linux__
  #include <string.h>
  #include <time.h>

// This is simpler and much more efficient than the C++ "chrono" way
uint64_t Timestamp() {
  struct timespec ts;
  int res = clock_gettime(CLOCK_REALTIME, &ts);
  if (res == -1) {
    throw MakeException("Could not get time stamp: %s", strerror(errno));
  }

  static constexpr uint64_t NANOS = 1'000'000'000;
  return (ts.tv_sec * NANOS + ts.tv_nsec);
}

grpc::Status MakeErrorStatus(const char* format, ...) {
  static constexpr std::size_t BUF_SIZE = 256;

  try {
    char buf[BUF_SIZE];
    va_list args;
    va_start(args, format);
    std::vsnprintf(buf, BUF_SIZE, format, args);
    va_end(args);

    const char* const const_buf = buf;
    spdlog::error("gRPC error returned: {}", const_buf);
    return grpc::Status(grpc::StatusCode::UNKNOWN, const_buf);
  }
  catch(...) {
    return grpc::Status::CANCELLED;  // Not ideal, but the only predefined status other than OK
  }
}

class ThreadPool::ThreadControl {
public:
  bool is_available() { return !m_running; }

  void cancel() {
    m_cancelled = true;
    m_running = true;  // To prevent any more scheduling
  }

  std::future<void> set_execution(FUNC_TYPE&& func, std::string_view desc) {
    if (m_cancelled) {
      throw MakeException("Cannot run [%.*s] on cancelled thread", static_cast<int>(desc.size()), desc.data());
    }

    m_running = true;
    m_prom = std::promise<void>();
    m_description.assign(desc.data(), desc.size()); 
    m_funcs.push(std::move(func));
    return m_prom.get_future();
  }

  void run() {
    while(true) {
      auto func = m_funcs.pop();
      try {
        func();
      }
      catch(const std::exception& exc) {
        spdlog::error("Threaded function [{}] failed [{}]", m_description, exc.what());
      }
      catch(...) {
        spdlog::error("Threaded function [{}] failed", m_description);
      }
      m_prom.set_value();
      m_running = false;
    }
  }

private:
  ThrQueue<FUNC_TYPE> m_funcs;
  bool m_running = false;
  bool m_cancelled = false;
  std::promise<void> m_prom;
  std::string m_description;
};

ThreadPool::~ThreadPool() {
  for (auto& thr : m_pool) {
    thr.detach();
  }
}

std::future<void> ThreadPool::push(std::string_view desc, FUNC_TYPE&& func) {
  if (!func) {
    std::string str_desc(desc);
    throw MakeException("Invalid function for thread pool [%s]", str_desc.c_str());
  }

  const std::lock_guard<std::mutex> lg(m_lock);

  for (auto& control : m_thread_controls) {
    if (control->is_available()) {
      return control->set_execution(std::move(func), desc);
    }
  }

  auto& thr_control = add_thread();
  return thr_control->set_execution(std::move(func), desc);
}

std::shared_ptr<ThreadPool::ThreadControl>& ThreadPool::add_thread() {
  m_thread_controls.emplace_back(std::make_shared<ThreadControl>());
  auto& thr_control = m_thread_controls.back();
  m_pool.emplace_back([thr_control]() {
    try {
      thr_control->run();
    }
    catch(const std::exception& exc) {
      spdlog::debug("Error in pooled thread [{}]", exc.what());
    }
    catch(...) {
      spdlog::debug("Error in pooled thread");
    }
    thr_control->cancel();
  });

  spdlog::debug("Nb of threads in pool: [{}]", m_thread_controls.size());
  return thr_control;
}

#endif
