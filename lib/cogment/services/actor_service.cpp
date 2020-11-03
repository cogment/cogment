// Copyright 2020 Artificial Intelligence Redefined <dev+cogment@ai-r.com>
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

#include "cogment/services/actor_service.h"

#include "cogment/orch_config.h"
#include "cogment/orchestrator.h"

namespace cogment {
ActorService::ActorService(Orchestrator* orch) : orchestrator_(orch) {}

::easy_grpc::Future<::cogment::TrialJoinReply> ActorService::JoinTrial(::cogment::TrialJoinRequest,
                                                                       easy_grpc::Context ctx) {
  (void)ctx;
  return {};
}

::easy_grpc::Stream_future<::cogment::TrialActionReply> ActorService::ActionStream(
    ::easy_grpc::Stream_future<::cogment::TrialActionRequest>, easy_grpc::Context ctx) {
  (void)ctx;
  return {};
}

::easy_grpc::Future<::cogment::TrialActionReply> ActorService::Action(::cogment::TrialActionRequest,
                                                                      easy_grpc::Context ctx) {
  (void)ctx;
  return {};
}

::easy_grpc::Future<::cogment::TrialHeartbeatReply> ActorService::Heartbeat(::cogment::TrialHeartbeatRequest,
                                                                            easy_grpc::Context ctx) {
  (void)ctx;
  return {};
}

::easy_grpc::Future<::cogment::TrialFeedbackReply> ActorService::GiveFeedback(::cogment::TrialFeedbackRequest,
                                                                              easy_grpc::Context ctx) {
  (void)ctx;
  return {};
}

::easy_grpc::Future<::cogment::TrialMessageReply> ActorService::SendChanMessage(::cogment::TrialMessageRequest,
                                                                                easy_grpc::Context ctx) {
  (void)ctx;
  return {};
}

::cogment::VersionInfo ActorService::Version(::cogment::VersionRequest, easy_grpc::Context ctx) {
  (void)ctx;
  ::cogment::VersionInfo result;
  auto v = result.add_versions();

  v->set_name("orchestrator");
  v->set_version(COGMENT_ORCHESTRATOR_VERSION);

  v = result.add_versions();
  v->set_name("cogment-api");
  v->set_version(COGMENT_API_VERSION);

  return result;
}
}  // namespace cogment