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

#ifndef COGMENT_ORCHESTRATOR_CLIENT_ACTOR_H
#define COGMENT_ORCHESTRATOR_CLIENT_ACTOR_H

#include "cogment/actor.h"

#include "cogment/api/orchestrator.grpc.pb.h"

namespace cogment {

class ServerStream : public ActorStream {
public:
  using StreamType = grpc::ServerReaderWriter<InputType, OutputType>;

  ServerStream(StreamType* stream) : m_stream(stream) {}

  bool read(OutputType* data) override { return m_stream->Read(data); }
  bool write(const InputType& data) override { return m_stream->Write(data); }
  bool write_last(const InputType& data) override {
    // We could decide to do nothing special here!
    grpc::WriteOptions options;
    options.set_last_message();
    return m_stream->Write(data, options);
  }
  bool finish() override { return true; }

private:
  StreamType* m_stream;
};

class Trial;

class ClientActor : public Actor {
public:
  ClientActor(Trial* owner, const cogmentAPI::ActorParams& params);

  static void run_an_actor(std::shared_ptr<Trial>&& trial_requested, ServerStream::StreamType* stream);
};

}  // namespace cogment
#endif
