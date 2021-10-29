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

// This is simpler and much more efficient than the C++ "std::chrono" way
uint64_t Timestamp() {
  struct timespec ts;
  int res = clock_gettime(CLOCK_REALTIME, &ts);
  if (res != -1) {
    return (ts.tv_sec * NANOS + ts.tv_nsec);
  }
  else {
    throw MakeException("Could not get time stamp: {}", strerror(errno));
  }
}

#endif

std::vector<std::string> split(const std::string& in, char separator) {
  std::vector<std::string> result;

  size_t first = 0;
  for (size_t pos = 0; pos < in.size(); pos++) {
    if (in[pos] != separator) {
      continue;
    }

    result.emplace_back(in.substr(first, pos - first));
    first = pos + 1;
  }

  if (first < in.size()) {
    result.emplace_back(in.substr(first));
  }

  return result;
}

class ThreadPool::ThreadControl {
public:
  void stop() { m_active = false; }
  bool is_available() { return (!m_executing_func && m_active); }

  std::future<void> set_execution(FUNC_TYPE&& func, std::string_view desc) {
    m_executing_func = true;
    m_prom = std::promise<void>();
    m_description.assign(desc.data(), desc.size());
    m_funcs.push(std::move(func));
    return m_prom.get_future();
  }

  void run() {
    m_active = true;

    try {
      while (m_active) {
        auto func = m_funcs.pop();
        try {
          func();
        }
        catch (const std::exception& exc) {
          spdlog::error("Threaded function [{}] failed [{}]", m_description, exc.what());
        }
        catch (...) {
          spdlog::error("Threaded function [{}] failed", m_description);
        }
        m_prom.set_value();
        m_executing_func = false;
      }
    }
    catch (const std::exception& exc) {
      spdlog::error("Pooled thread failure: {}", exc.what());
    }
    catch (...) {
      spdlog::error("Pooled thread failure");
    }

    m_active = false;
    m_executing_func = false;
  }

private:
  ThrQueue<FUNC_TYPE> m_funcs;
  bool m_active = false;
  bool m_executing_func = false;
  std::promise<void> m_prom;
  std::string m_description;
};

ThreadPool::~ThreadPool() {
  for (auto& ctrl : m_thread_controls) {
    ctrl->stop();
  }
  for (auto& thr : m_thread_pool) {
    thr.detach();
  }
}

std::future<void> ThreadPool::push(std::string_view desc, FUNC_TYPE&& func) {
  if (!func) {
    std::string str_desc(desc);
    throw MakeException("Invalid function for thread pool [{}]", str_desc);
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
  m_thread_pool.emplace_back([thr_control]() {
    thr_control->run();
  });

  spdlog::debug("Nb of threads in pool: [{}]", m_thread_controls.size());
  return thr_control;
}
