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

#include "cogment/agent_actor.h"
#include "cogment/trial.h"
#include "cogment/utils.h"

#include "spdlog/spdlog.h"

namespace cogment {

ServiceActor::ServiceActor(Trial* owner, const cogmentAPI::ActorParams& params, StubEntryType stub_entry) :
    Actor(owner, params, true), m_stub_entry(std::move(stub_entry)) {
  m_context.AddMetadata("trial-id", trial()->id());
}

std::future<void> ServiceActor::init() {
  SPDLOG_TRACE("ServiceActor::init(): [{}] [{}]", trial()->id(), actor_name());

  auto grpc_stream = m_stub_entry->get_stub().RunTrial(&m_context);
  auto stream = std::make_unique<ClientStream>(std::move(grpc_stream));

  // TODO: The problem here is that this stream will outlive the context. And less importantly
  //       (because it is kept in the orchestrator) it will outlive the stub entry.
  //       We could move them in the ClientStream!
  run(std::move(stream));

  return Actor::init();
}

}  // namespace cogment
