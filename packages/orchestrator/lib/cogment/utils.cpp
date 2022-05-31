// Copyright 2022 AI Redefined Inc. <dev+cogment@ai-r.com>
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

namespace cogment {

#if defined(__linux__) || defined(__APPLE__)
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
#else
  #include <chrono>

// Fallback to "std::chrono", inspired by https://stackoverflow.com/a/31258680
uint64_t Timestamp() {
  auto now_system = std::chrono::system_clock::now();
  auto now_nano = std::chrono::time_point_cast<std::chrono::nanoseconds>(now_system);
  auto ts_nano = now_nano.time_since_epoch();
  return ts_nano.count();
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

std::string_view trim(std::string_view in) {
  std::string_view result = in;

  for (auto val : in) {
    if (!std::isspace(static_cast<unsigned char>(val))) {
      break;
    }
    else {
      result.remove_prefix(1);
    }
  }

  for (auto itor = in.rbegin(); itor != in.rend(); itor++) {
    const auto val = static_cast<unsigned char>(*itor);
    if (!std::isspace(val)) {
      break;
    }
    else {
      result.remove_suffix(1);
    }
  }

  return result;
}

class ThreadPool::ThreadControl {
public:
  bool is_available() const { return (!m_executing_func && m_active); }

  void stop() {
    m_active = false;  // To stop once func is done
    m_funcs.push({});  // To stop if there is no func running
  }

  std::future<void> set_execution(FUNC_TYPE&& func, std::string_view desc) {
    if (!func) {
      throw MakeException("Non-callable function for thread pool [{}]", desc);
    }

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
        if (!func) {
          break;
        }

        try {
          func();
        }
        catch (const std::exception& exc) {
          spdlog::error("Threaded function [{}] failed [{}]", m_description, exc.what());
        }
        catch (...) {
          spdlog::error("Threaded function [{}] failed", m_description);
        }
        m_prom.set_value();  // No need for 'set_exception' for now
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
  const std::lock_guard lg(m_push_lock);

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
    spdlog::debug("Threadpool thread exiting");
  });

  spdlog::debug("Nb of threads in pool: [{}]", m_thread_controls.size());
  return thr_control;
}

Watchdog::Watchdog(ThreadPool* pool) :
    m_base_time(std::chrono::high_resolution_clock::now()), m_sec_count(0), m_running(true) {
  m_time_thr = pool->push("Watchdog", [this]() {
    while (m_running) {
      const auto next = m_base_time + std::chrono::seconds(m_sec_count + 1);
      std::this_thread::sleep_until(next);
      m_sec_count++;

      // Assuming it will take less than 1 sec to execute!
      process_timeouts(m_sec_count);
    }
  });

  m_execution_thr = pool->push("Watchdog callback execution", [this]() {
    while (m_running) {
      execute_functions();
    }
  });
}

Watchdog::~Watchdog() {
  m_running = false;
  m_execution_queue.push(FUNC_TYPE());
  m_time_thr.wait();
  m_execution_thr.wait();
}

void Watchdog::push(uint16_t timeout_sec, FUNC_TYPE&& func) {
  if (!func) {
    MakeException("Trying to set a watchdog function that is undefined");
  }

  if (timeout_sec > 0) {
    const std::lock_guard lg(m_timed_queue_lock);
    const uint64_t time_count = m_sec_count + timeout_sec;
    m_time_queue.emplace(time_count, std::move(func));
  }
  else {
    func();
  }
}

void Watchdog::process_timeouts(uint64_t at_sec_count) {
  const std::lock_guard lg(m_timed_queue_lock);

  while (!m_time_queue.empty()) {
    auto& top = const_cast<WatchEntry&>(m_time_queue.top());  // Const cast ok: we won't change the count

    if (top.sec_count > at_sec_count) {
      break;
    }
    else {
      m_execution_queue.push(std::move(top.func));
      m_time_queue.pop();
    }
  }
}

void Watchdog::execute_functions() {
  auto func = m_execution_queue.pop();
  if (!func) {
    return;
  }

  SPDLOG_TRACE("Watchdog task execution at [{}] sec",
               std::chrono::high_resolution_clock::now().time_since_epoch().count() * NANOS_INV);
  try {
    func();
  }
  catch (const std::exception& exc) {
    spdlog::error("Watchdog function failed [{}]", exc.what());
  }
  catch (...) {
    spdlog::error("Watchdog function failed");
  }
}

MultiWait::MultiWait() : m_waiting(false) {}

void MultiWait::push_back(float timeout_sec, FUT_TYPE&& fut) {
  if (m_waiting) {
    throw MakeException("Cannot add futures while waiting");
  }
  if (!fut.valid()) {
    throw MakeException("Cannot add invalid futures");
  }

  const int64_t timeout_ns = timeout_sec * NANOS;
  const size_t fut_index = m_futures.size();
  m_wait_queue.emplace(timeout_ns, fut_index);
  m_futures.emplace_back(std::move(fut));
}

std::vector<size_t> MultiWait::wait_for_all() {
  static const auto zero = std::chrono::nanoseconds::zero();

  m_waiting = true;
  const auto base_time = std::chrono::high_resolution_clock::now();
  std::vector<size_t> result;

  for (; !m_wait_queue.empty(); m_wait_queue.pop()) {
    auto& top = m_wait_queue.top();
    auto& fut = m_futures[top.fut_index];

    if (top.wait_time_ns < 0) {
      fut.wait();

      SPDLOG_TRACE("Required future seen initialized after [{}] sec",
                   (std::chrono::high_resolution_clock::now() - base_time).count() * NANOS_INV);
    }
    else {
      const auto wait_time = std::chrono::nanoseconds(top.wait_time_ns);
      const auto now = std::chrono::high_resolution_clock::now();
      auto more_wait = wait_time - (now - base_time);
      if (more_wait < zero) {
        more_wait = zero;
      }

      const auto fut_res = fut.wait_for(more_wait);

      switch (fut_res) {
      case std::future_status::deferred:
        throw MakeException("Cannot accept deferred action in MultiWait futures");
        break;
      case std::future_status::ready:
        SPDLOG_TRACE("Optional future (with timeout [{}]) seen initialized after [{}] sec", top.wait_time_ns,
                     (std::chrono::high_resolution_clock::now() - base_time).count() * NANOS_INV);
        break;
      case std::future_status::timeout:
        result.emplace_back(top.fut_index);
        SPDLOG_TRACE("Optional future (with timeout [{}]) seen timed out after [{}] sec", top.wait_time_ns,
                     (std::chrono::high_resolution_clock::now() - base_time).count() * NANOS_INV);
        break;
      }
    }
  }

  m_futures.clear();
  m_waiting = false;

  return result;
}

}  // namespace cogment
