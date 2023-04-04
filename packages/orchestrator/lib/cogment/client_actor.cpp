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

#ifndef NDEBUG
  #define SPDLOG_ACTIVE_LEVEL SPDLOG_LEVEL_TRACE
#endif

#include "cogment/client_actor.h"
#include "cogment/trial.h"
#include "cogment/utils.h"

namespace cogment {

// Static
void ClientActor::run_an_actor(std::shared_ptr<Trial>&& trial_requested, ServerStream::StreamType* grpc_stream) {
  SPDLOG_TRACE("Client actor run_an_actor");

  if (grpc_stream == nullptr) {
    throw MakeException("Stream is null!");
  }
  auto server_stream = std::make_unique<ServerStream>(grpc_stream);

  // The init can be long or even hang
  std::weak_ptr<Trial> trial_weak(trial_requested);
  trial_requested.reset();

  cogmentAPI::ActorInitialOutput init_data;
  bool init_valid = false;
  try {
    init_valid = read_init_data(server_stream.get(), &init_data);
  }
  catch (const std::exception& exc) {
    throw MakeException("Init data failure [{}]", exc.what());
  }
  catch (...) {
    throw MakeException("Init data failure");
  }

  auto trial = trial_weak.lock();
  if (init_valid && trial != nullptr) {
    std::string actor_name;
    const auto slot_case = init_data.slot_selection_case();
    if (slot_case == cogmentAPI::ActorInitialOutput::kActorName) {
      actor_name = init_data.actor_name();
    }
    std::string actor_class;
    if (slot_case == cogmentAPI::ActorInitialOutput::kActorClass) {
      actor_class = init_data.actor_class();
    }
    auto actor = trial->get_join_candidate(actor_name, actor_class);
    init_data.Clear();
    trial.reset();

    auto run_fut = actor->run([stream_ptr = server_stream.release()]() {
      return std::unique_ptr<ActorStream>(stream_ptr);
    });

    // Since the only clean way to close a stream on the server side (with the sync grpc API) is
    // to return: we need to be able to return without waiting for the stream.
    run_fut.wait();
  }
}

ClientActor::ClientActor(Trial* owner, const cogmentAPI::ActorParams& params) : Actor(owner, params, false) {}

std::future<bool> ClientActor::init() {
  SPDLOG_TRACE("ClientActor::init(): [{}] [{}]", trial()->id(), actor_name());
  return get_run_init_fut();
}

}  // namespace cogment