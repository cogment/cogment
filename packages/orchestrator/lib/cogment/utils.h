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

#ifndef COGMENT_ORCHESTRATOR_UTILS_H
#define COGMENT_ORCHESTRATOR_UTILS_H

#include "grpc++/grpc++.h"
#include "spdlog/spdlog.h"

#include <cstdarg>
#include <cstdio>
#include <queue>
#include <mutex>
#include <thread>
#include <future>
#include <utility>
#include <condition_variable>
#include <algorithm>
#include <chrono>

namespace cogment {

constexpr uint64_t NANOS = 1'000'000'000;
constexpr double NANOS_INV = 1.0 / NANOS;
using COGMENT_ERROR_BASE_TYPE = std::runtime_error;

// Ignores (i.e. not added to the vector) the last empty string if there is a trailing separator
std::vector<std::string> split(const std::string& in, char separator);

// Trim white space from begining and end of input string
std::string_view trim(std::string_view in);

// Unix epoch time in nanoseconds
uint64_t Timestamp();

// TODO: Update (or replace) to use std::format (C++20)
template <class... Args>
std::string MakeString(const char* format, Args&&... args) {
  return fmt::format(format, std::forward<Args>(args)...);
}

class CogmentError : public COGMENT_ERROR_BASE_TYPE {
  using COGMENT_ERROR_BASE_TYPE::COGMENT_ERROR_BASE_TYPE;
};

template <class EXC = CogmentError, class... Args>
EXC MakeException(const char* format, Args&&... args) {
  try {
    std::string val = MakeString(format, std::forward<Args>(args)...);
    spdlog::debug("**Exception generated**: {}", val);
    return EXC(val);
  }
  catch (...) {
    return EXC("Could not create exception message");
  }
}

template <class... Args>
grpc::Status MakeErrorStatus(const char* format, Args&&... args) {
  try {
    std::string val = MakeString(format, std::forward<Args>(args)...);
    spdlog::error("gRPC failed status returned: {}", val);
    return grpc::Status(grpc::StatusCode::UNKNOWN, val);
  }
  catch (...) {
    return grpc::Status::CANCELLED;  // Not ideal, but the only predefined status other than OK
  }
}

inline std::string& to_lower_case(std::string&& str) {
  for (auto& val : str) {
    val = static_cast<char>(std::tolower(static_cast<unsigned char>(val)));
  }

  return str;
}

inline std::string to_lower_case(std::string_view str) {
  std::string result(str.size(), '\0');
  for (size_t index = 0; index < str.size(); index++) {
    result[index] = static_cast<char>(std::tolower(static_cast<unsigned char>(str[index])));
  }

  return result;
}

// Return only the first instance of the key if there are more
template <class Container>
std::string_view OneFromMetadata(const Container& metadata, std::string_view key) {
  auto itor = metadata.find(grpc::string_ref(key.data(), key.size()));
  if (itor == metadata.end()) {
    throw MakeException("No [{}] key in metadata", key);
  }
  return std::string_view(itor->second.data(), itor->second.size());
}

// Return all instances of the key
template <class Container>
std::vector<std::string_view> FromMetadata(const Container& metadata, std::string_view key) {
  std::vector<std::string_view> result;

  auto range = metadata.equal_range(grpc::string_ref(key.data(), key.size()));
  for (auto itor = range.first; itor != range.second; ++itor) {
    result.emplace_back(itor->second.data(), itor->second.size());
  }

  return result;
}

// Minimal thread-safe queue
template <typename T>
class ThrQueue {
public:
  T pop() {
    std::unique_lock ul(m_queue_lock);

    if (m_data.empty()) {
      m_cond.wait(ul, [this]() {
        return !m_data.empty();
      });
    }

    T val(std::move(m_data.front()));
    m_data.pop();
    return val;
  }

  void push(T&& val) {
    std::unique_lock ul(m_queue_lock);

    m_data.emplace(std::forward<T>(val));
    ul.unlock();
    m_cond.notify_one();
  }

  size_t size() {
    const std::lock_guard lg(m_queue_lock);
    return m_data.size();
  }

private:
  std::queue<T> m_data;
  std::mutex m_queue_lock;
  std::condition_variable m_cond;
};

// Minimal thread pool
class ThreadPool {
  using FUNC_TYPE = std::function<void()>;

public:
  ThreadPool() = default;
  ~ThreadPool();

  ThreadPool(const ThreadPool&) = delete;
  ThreadPool(ThreadPool&& tpool) = delete;
  void operator=(const ThreadPool&) = delete;
  void operator=(ThreadPool&& tpool) = delete;

  // The future indicates that the execution of the function is finished
  std::future<void> push(std::string_view desc, FUNC_TYPE&& func);

private:
  class ThreadControl;
  std::shared_ptr<ThreadControl>& add_thread();

  std::vector<std::thread> m_thread_pool;
  std::vector<std::shared_ptr<ThreadControl>> m_thread_controls;
  std::mutex m_push_lock;
};

// Minimal: Hardcoded to 1 sec period and integer timeouts.
// Vulnerable to action function problems.
// Action functions should ideally be simple and fast.
class Watchdog {
public:
  using FUNC_TYPE = std::function<void()>;

  Watchdog(ThreadPool* pool);
  ~Watchdog();
  void push(uint16_t timeout_sec, FUNC_TYPE&& func);

private:
  struct WatchEntry {
    uint64_t sec_count;
    FUNC_TYPE func;

    WatchEntry(uint16_t cc, FUNC_TYPE&& ff) : sec_count(cc), func(std::move(ff)) {}
    bool operator>(const WatchEntry& right) const { return (sec_count > right.sec_count); }
  };

  void process_timeouts(uint64_t at_sec_count);
  void execute_functions();

  const std::chrono::time_point<std::chrono::high_resolution_clock> m_base_time;
  uint64_t m_sec_count;
  bool m_running;

  std::future<void> m_time_thr;
  std::mutex m_timed_queue_lock;
  std::priority_queue<WatchEntry, std::vector<WatchEntry>, std::greater<WatchEntry>> m_time_queue;

  std::future<void> m_execution_thr;
  ThrQueue<FUNC_TYPE> m_execution_queue;
};

// Minimal class to wait for multiple futures with varied timeouts.
class MultiWait {
  using FUT_TYPE = std::future<void>;

public:
  MultiWait();

  // timeout < 0 for no time out (i.e. infinite wait).
  // The waiting is serial and ordered from the smaller timeouts to the larger.
  void push_back(float timeout_sec, FUT_TYPE&& fut, std::string_view descr);
  void push_back(FUT_TYPE&& fut, std::string_view descr) { push_back(-1.0f, std::move(fut), descr); };

  // Return: List of indexes of futures that timed out (not in any particular order).
  std::vector<size_t> wait_for_all();

private:
  struct FutEntry {
    int64_t wait_time_ns;
    size_t fut_index;
    std::string_view description;  // Can be dangerous, but it works the way we use it.

    FutEntry(int64_t wt, size_t index, std::string_view descr) :
        wait_time_ns(wt), fut_index(index), description(descr) {}
    bool operator>(const FutEntry& right) const { return (wait_time_ns > right.wait_time_ns); }
  };

  std::priority_queue<FutEntry, std::vector<FutEntry>, std::greater<FutEntry>> m_wait_queue;
  std::vector<FUT_TYPE> m_futures;
  bool m_waiting;
};

class SimpleSignal {
public:
  void deactivate() {
    std::unique_lock ul(m_lock);
    m_active = false;
    m_signalled = true;
    m_cond.notify_all();
  }

  void signal() {
    std::unique_lock ul(m_lock);
    if (m_active && !m_signalled) {
      m_signalled = true;
      ul.unlock();
      m_cond.notify_one();
    }
  }

  bool wait() {
    std::unique_lock ul(m_lock);
    if (m_active) {
      m_cond.wait(ul, [this] {
        return m_signalled;
      });
      if (!m_signalled)
        throw MakeException("Spurious waking of condition variable!!!!");
      m_signalled = !m_active;
    }
    return m_active;
  }

private:
  std::mutex m_lock;
  std::condition_variable m_cond;
  bool m_signalled = false;
  bool m_active = true;
};

}  // namespace cogment

#endif
