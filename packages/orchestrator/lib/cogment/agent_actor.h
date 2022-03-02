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

#ifndef COGMENT_ORCHESTRATOR_AGENT_ACTOR_H
#define COGMENT_ORCHESTRATOR_AGENT_ACTOR_H

#include "cogment/actor.h"
#include "cogment/stub_pool.h"

#include "cogment/api/agent.grpc.pb.h"
#include "cogment/api/common.pb.h"

namespace cogment {

class ClientStream : public ActorStream {
public:
  using StreamType = grpc::ClientReaderWriter<InputType, OutputType>;

  ClientStream(std::unique_ptr<StreamType> stream) : m_stream(std::move(stream)) {}

  bool read(OutputType* data) override { return m_stream->Read(data); }
  bool write(const InputType& data) override { return m_stream->Write(data); }
  bool write_last(const InputType& data) override {
    if (m_stream->Write(data)) {
      return m_stream->WritesDone();
    }
    return false;
  }
  bool finish() override { return m_stream->Finish().ok(); }

private:
  std::unique_ptr<StreamType> m_stream;
};

class Trial;

class ServiceActor : public Actor {
  using StubEntryType = std::shared_ptr<StubPool<cogmentAPI::ServiceActorSP>::Entry>;

public:
  ServiceActor(Trial* owner, const cogmentAPI::ActorParams& params, StubEntryType stub_entry);

  std::future<void> init() override;

private:
  StubEntryType m_stub_entry;
  grpc::ClientContext m_context;
};

}  // namespace cogment

#endif
