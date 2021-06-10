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

#ifndef AOM_ORCHESTRATOR_STUB_POOL_H
#define AOM_ORCHESTRATOR_STUB_POOL_H

#include <condition_variable>
#include <future>
#include <mutex>
#include <set>
#include <typeinfo>
#include "cogment/utils.h"
#include "easy_grpc/easy_grpc.h"
#include "spdlog/spdlog.h"

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
class Channel_pool {
  public:
  Channel_pool(std::shared_ptr<easy_grpc::client::Credentials> creds) : creds_(creds) {}

  // Gets an easy-grpc channel to the target url, recycling an existing one if
  // present.
  std::shared_ptr<::easy_grpc::client::Channel> get_channel(const std::string& url) {
    std::lock_guard l(mtx_);

    auto& found = channels_[url];
    auto result = found.lock();

    if (!result) {
      if (creds_.get() != nullptr) {
        result = std::make_shared<::easy_grpc::client::Secure_channel>(url, nullptr, creds_.get());
        spdlog::info("Opening secured channel to {}", url);
      }
      else {
        result = std::make_shared<::easy_grpc::client::Unsecure_channel>(url, nullptr);
        spdlog::info("Opening unsecured channel to {}", url);
      }

      found = result;
    }

    return result;
  }

  std::mutex mtx_;
  std::unordered_map<std::string, std::weak_ptr<::easy_grpc::client::Channel>> channels_;

  private:
  std::shared_ptr<easy_grpc::client::Credentials> creds_;
};

// A minimal thread-safe queue
template <typename T>
class ThrQueue {
  public:
  T pop() {
    std::unique_lock ul(m_lock);

    if (m_data.empty()) {
      m_cond.wait(ul, [this]() { return !m_data.empty(); });
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

// A thread-safe pool of easy-grpc connection stubs
template <typename Service_T>
class Stub_pool {
  public:
  using stub_type = typename Service_T::Stub;

  // Constructor
  Stub_pool(Channel_pool* channel_pool, easy_grpc::Completion_queue* queue)
      : channel_pool_(channel_pool), queue_(queue) {}

  using SerializedFunc = std::function<void()>;
  struct SerialCall {
    SerialCall(SerializedFunc&& serial_func) : func(serial_func) {}
    SerialCall(SerialCall&& prev) : func(std::move(prev.func)), prom(std::move(prev.prom)) {}
    SerializedFunc func;
    std::promise<void> prom;
  };

  // TODO: Implement proper destruction, or remove serialization option.  It leaks as it is.
  //       In current use-cases, it is not a big problem because entries are semi-permanent, and are few.
  class Entry {
    public:
    using ChannelType = std::shared_ptr<::easy_grpc::client::Channel>;
    Entry(ChannelType&& chan, stub_type&& stb) : m_channel(chan), m_stub(stb) {}

    void enable_serialization(std::weak_ptr<Entry> this_weak) {
      auto this_entry = this_weak.lock();
      if (this_entry.get() != this) {
        throw MakeException("Entry serialization enabling must be done with own pointer.");
      }

      if (m_serializing_thread.joinable()) {
        spdlog::warn("Serialization already enabled");
        return;
      }

      m_serializing_thread = std::thread([this_weak]() {
        spdlog::debug("Stub serialization thread started");

        while (true) {
          auto this_entry = this_weak.lock();
          if (this_entry == nullptr) {
            break;
          }

          try {
            auto call = this_entry->m_serialized_calls.pop();
            SPDLOG_TRACE("Stub serialization thread popped a call");

            try {
              call.func();
              SPDLOG_TRACE("Serialized function call returned");
              call.prom.set_value();
            } catch (...) {
              try {
                call.prom.set_exception(std::current_exception());
              } catch (...) {
                spdlog::error("Exception trying to set promise exception");
              }
            }
          } catch (...) {
            spdlog::error("Could not pop serialized call");
          }
        }

        // A "warning" because there is no planned way to end up here
        spdlog::warn("Stub serialization thread ending");
      });
    }

    std::future<void> serialize(SerializedFunc&& func) {
      if (m_serializing_thread.joinable()) {
        SerialCall call(std::move(func));
        auto fut = call.prom.get_future();
        m_serialized_calls.push(std::move(call));
        return fut;
      }
      else {
        SPDLOG_WARN("Trying to serialize a call when serialization is disabled");
        func();

        std::promise<void> prom;
        auto fut = prom.get_future();
        prom.set_value();
        return fut;
      }
    }

    stub_type& get_stub() { return m_stub; }

    private:
    // Prevents the channel from being destroyed.
    ChannelType m_channel;

    stub_type m_stub;

    // TODO: Re-evaluate if this serialization is still needed
    // We use serialization because of a bug in gRPC 1.37.1 (or easygrpc) that causes
    // a crash when multiple non-streamning calls are made to the same service simultenously.
    // We need to implement the serialization this way becasue we cannot make the calls synchronous
    // (i.e. wait for call to be done with `future.get()`) because of the use of
    // completion queues that lead to deadlock when calls are embeded.
    ThrQueue<SerialCall> m_serialized_calls;
    std::thread m_serializing_thread;
  };

  // Gets an easy-grpc channel to the target service at the target url,
  // recycling an existing one if present.
  std::shared_ptr<Entry> get_stub_entry(const std::string& url) {
    if (url.find("grpc://") != 0) {
      throw MakeException("Bad grpc url: [%s]", url.c_str());
    }
    std::lock_guard l(mtx_);

    auto real_url = url.substr(7);
    auto& found = entries_[real_url];
    auto result = found.lock();

    if (!result) {
      spdlog::info("Opening stub for {} at {}", typeid(Service_T).name(), real_url);
      auto channel = channel_pool_->get_channel(real_url);

      result = std::make_shared<Entry>(std::move(channel), stub_type(channel.get(), queue_));
      found = result;
      result->enable_serialization(found);
      spdlog::debug("Stub {} at {} ready for use", typeid(Service_T).name(), real_url);
    }

    return result;
  }

  private:
  std::mutex mtx_;
  Channel_pool* channel_pool_;
  easy_grpc::Completion_queue* queue_;
  std::unordered_map<std::string, std::weak_ptr<Entry>> entries_;
};

}  // namespace cogment
#endif