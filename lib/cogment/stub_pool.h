#ifndef AOM_ORCHESTRATOR_STUB_POOL_H
#define AOM_ORCHESTRATOR_STUB_POOL_H

#include <set>
#include <typeinfo>
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
  // Gets an easy-grpc channel to the target url, recycling an existing one if
  // present.
  std::shared_ptr<::easy_grpc::client::Channel> get_channel(const std::string& url) {
    std::lock_guard l(mtx_);

    auto& found = channels_[url];
    auto result = found.lock();

    if (!result) {
      result = std::make_shared<::easy_grpc::client::Unsecure_channel>(url, nullptr);
      spdlog::info("opening channel to {}", url);
      found = result;
    }

    return result;
  }

  public:
  std::mutex mtx_;
  std::unordered_map<std::string, std::weak_ptr<::easy_grpc::client::Channel>> channels_;
};

// A thread-safe pool of easy-grpc connection stubs
template <typename Service_T>
class Stub_pool {
  public:
  using stub_type = typename Service_T::Stub;

  // Constructor
  Stub_pool(Channel_pool* channel_pool, easy_grpc::Completion_queue* queue)
      : channel_pool_(channel_pool), queue_(queue) {}

  struct Entry {
    // Prevents the channel from being destroyed.
    std::shared_ptr<::easy_grpc::client::Channel> channel;
    stub_type stub;

    stub_type* operator->() { return &stub; }
  };

  // Gets an easy-grpc channel to the target service at thetarget url,
  // recycling an existing one if present.
  std::shared_ptr<Entry> get_stub(const std::string& url) {
    if (url.find("grpc://") != 0) {
      spdlog::error("bad grpc url: \"{}\", should start with \"grpc://\"", url);
      throw std::runtime_error("bad grpc url");
    }
    std::lock_guard l(mtx_);

    auto real_url = url.substr(7);
    auto& found = entries_[real_url];
    auto result = found.lock();

    if (!result) {
      spdlog::info("opening stub for {} at {}", typeid(Service_T).name(), real_url);
      auto channel = channel_pool_->get_channel(real_url);

      result = std::make_shared<Entry>(Entry{channel, stub_type(channel.get(), queue_)});
      found = result;
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