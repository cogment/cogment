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

#ifndef COGMENT_UTILS_H_INCLUDED
#define COGMENT_UTILS_H_INCLUDED

#include <cstdarg>
#include <cstdio>
#include <queue>
#include <mutex>
#include <condition_variable>
#include "grpc++/grpc++.h"
#include "spdlog/spdlog.h"

grpc::Status MakeErrorStatus(const char* format, ...);
uint64_t Timestamp();

// TODO: Make a specific cogment exception for all internally generated exceptions
template <class EXC = std::runtime_error>
EXC MakeException(const char* format, ...) {
  static constexpr std::size_t BUF_SIZE = 256;

  try {
    char buf[BUF_SIZE];
    va_list args;
    va_start(args, format);
    std::vsnprintf(buf, BUF_SIZE, format, args);
    va_end(args);

    const char* const const_buf = buf;
    spdlog::error("**Exception generated**: {}", const_buf);
    return EXC(const_buf);
  }
  catch(...) {
    return EXC("Error creating exception message");
  }
}

// Return only the first instance if there are more
template <class Container>
std::string_view OneFromMetadata(const Container& metadata, std::string_view key) {
  auto itor = metadata.find(grpc::string_ref(key.data(), key.size()));
  if (itor == metadata.end()) {
    throw MakeException("No [%.*s] key in metadata", static_cast<int>(key.size()), key.data());
  }
  return std::string_view(itor->second.data(), itor->second.size());
}

template <class Container>
std::vector<std::string_view> FromMetadata(const Container& metadata, std::string_view key) {
  std::vector<std::string_view> result;

  auto range = metadata.equal_range(grpc::string_ref(key.data(), key.size()));
  for (auto itor = range.first ; itor != range.second ; ++itor) {
    result.emplace_back(itor->second.data(), itor->second.size());
  }

  return result;
}

// A minimal thread-safe queue
template <typename T>
class ThrQueue {
public:
  T pop() {
    std::unique_lock ul(m_lock);

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
    std::unique_lock ul(m_lock);

    m_data.push(std::move(val));
    ul.unlock();
    m_cond.notify_one();
  }

  size_t size() const {
    std::unique_lock ul(m_lock);
    return m_data.size();
  }

private:
  std::queue<T> m_data;
  mutable std::mutex m_lock;
  std::condition_variable m_cond;
};

#endif
