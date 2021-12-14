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

constexpr uint64_t NANOS = 1'000'000'000;
constexpr double NANOS_INV = 1.0 / NANOS;
using COGMENT_ERROR_BASE_TYPE = std::runtime_error;

// Ignores (i.e. not added to the vector) the last empty string if there is a trailing separator
std::vector<std::string> split(const std::string& in, char separator);

// Unix epoch time in nanoseconds
uint64_t Timestamp();

class CogmentError : public COGMENT_ERROR_BASE_TYPE {
  using COGMENT_ERROR_BASE_TYPE::COGMENT_ERROR_BASE_TYPE;
};

// TODO: Update to use std::format (C++20)
template <class EXC = CogmentError, class... Args>
EXC MakeException(const char* format, Args&&... args) {
  try {
    std::string val = fmt::format(format, std::forward<Args>(args)...);
    spdlog::debug("**Exception generated**: {}", val);
    return EXC(val);
  }
  catch (...) {
    return EXC("Could not create exception message");
  }
}

// TODO: Update to use std::format (C++20)
template <class... Args>
grpc::Status MakeErrorStatus(const char* format, Args&&... args) {
  try {
    std::string val = fmt::format(format, std::forward<Args>(args)...);
    spdlog::error("gRPC failed status returned: {}", val);
    return grpc::Status(grpc::StatusCode::UNKNOWN, val);
  }
  catch (...) {
    return grpc::Status::CANCELLED;  // Not ideal, but the only predefined status other than OK
  }
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

#endif
