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

#ifndef COGMENT_ORCHESTRATOR_STUB_POOL_H
#define COGMENT_ORCHESTRATOR_STUB_POOL_H

#include "cogment/utils.h"

#include "spdlog/spdlog.h"
#include "grpc++/grpc++.h"

#include <mutex>
#include <set>
#include <typeinfo>

namespace cogment {

// At the application level, stubs are used to represent a connection to a gRPC
// service. However, the Orchestrator's trials can connect to various agent
// and environment services in a somewhat unpredictable manner. We want to
// recycle stubs and channels as efficently as possible, while not keeping
// conncetions open longer than they need.

//
// - A Channel must stay open as long as there is at least one active stub on it
// - A Channel must close as soon as the last stub disconnects from it.
// - A stub must stay open as long as at least one trial is making use of it
// - A stub must close as soon as no trial is making use of it.

// A thread-safe pool of easy-grpc Communication channel
class ChannelPool {
public:
  ChannelPool(std::shared_ptr<grpc::ChannelCredentials> creds) {
    if (creds.get() == nullptr) {
      spdlog::warn("Implicitly using unsecured channels");
      m_creds = grpc::InsecureChannelCredentials();
    }
    else {
      m_creds = std::move(creds);
    }
  }

  std::shared_ptr<grpc::Channel> get_channel(const std::string& url) {
    const std::lock_guard lg(m_map_lock);

    auto& found = m_channels[url];
    auto result = found.lock();

    if (!result) {
      result = grpc::CreateChannel(url, m_creds);
      found = result;
    }
    // TODO: manage the "else" for nullptr

    return result;
  }

  std::mutex m_map_lock;
  std::unordered_map<std::string, std::weak_ptr<grpc::Channel>> m_channels;

private:
  std::shared_ptr<grpc::ChannelCredentials> m_creds;
};

template <typename Service_T>
class StubPool {
public:
  using StubType = typename Service_T::Stub;

  // Constructor
  StubPool(ChannelPool* channel_pool) : m_channel_pool(channel_pool) {}

  class Entry {
  public:
    using ChannelType = std::shared_ptr<grpc::Channel>;
    Entry(ChannelType&& chan, StubType&& stb) : m_channel(chan), m_stub(stb) {}

    StubType& get_stub() { return m_stub; }

  private:
    // Prevents the channel from being destroyed.
    ChannelType m_channel;

    StubType m_stub;
  };

  std::shared_ptr<Entry> get_stub_entry(const std::string& url) {
    const std::lock_guard lg(m_map_lock);

    if (url.find("grpc://") != 0) {
      throw MakeException("Bad grpc url (must start with 'grpc://'): [{}]", url);
    }

    auto real_url = url.substr(7);
    auto& found = m_entries[real_url];
    auto result = found.lock();

    if (!result) {
      spdlog::info("Opening channel for [{}] at [{}]", Service_T::service_full_name(), real_url);
      auto channel = m_channel_pool->get_channel(real_url);

      result = std::make_shared<Entry>(std::move(channel), StubType(channel));
      found = result;
      spdlog::debug("Stub [{}] at [{}] ready for use", Service_T::service_full_name(), real_url);
    }

    return result;
  }

private:
  std::mutex m_map_lock;
  ChannelPool* m_channel_pool;
  std::unordered_map<std::string, std::weak_ptr<Entry>> m_entries;
};

}  // namespace cogment
#endif