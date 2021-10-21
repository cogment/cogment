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

#ifndef COGMENT_ORCHESTRATOR_ACTOR_SERVICE_H
#define COGMENT_ORCHESTRATOR_ACTOR_SERVICE_H

#include "cogment/api/orchestrator.grpc.pb.h"

namespace cogment {
class Orchestrator;

class ActorService final : public cogmentAPI::ClientActorSP::Service {
public:
  ActorService(Orchestrator* orch);

  grpc::Status RunTrial(
      grpc::ServerContext* ctx,
      grpc::ServerReaderWriter<cogmentAPI::ActorRunTrialInput, cogmentAPI::ActorRunTrialOutput>* stream) override;
  grpc::Status Version(grpc::ServerContext* ctx, const cogmentAPI::VersionRequest* in,
                       cogmentAPI::VersionInfo* out) override;

private:
  Orchestrator* m_orchestrator;
};

}  // namespace cogment

#endif