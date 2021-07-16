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

#ifndef COGMENT_ORCHESTRATOR_TRIAL_LIFECYCLE_SERVICE_H
#define COGMENT_ORCHESTRATOR_TRIAL_LIFECYCLE_SERVICE_H

#include "cogment/api/orchestrator.egrpc.pb.h"

namespace cogment {

class Orchestrator;
class TrialLifecycleService {
  Orchestrator* m_orchestrator;

  using Trial_promise = ::easy_grpc::Stream_promise<cogmentAPI::TrialListEntry>;
  using Trial_future = ::easy_grpc::Stream_future<cogmentAPI::TrialListEntry>;

public:
  using service_type = cogmentAPI::TrialLifecycleSP;

  TrialLifecycleService(Orchestrator* orch);

  ::easy_grpc::Future<cogmentAPI::TrialStartReply> StartTrial(::cogmentAPI::TrialStartRequest, easy_grpc::Context ctx);
  ::cogmentAPI::TerminateTrialReply TerminateTrial(::cogmentAPI::TerminateTrialRequest, easy_grpc::Context ctx);
  ::cogmentAPI::TrialInfoReply GetTrialInfo(::cogmentAPI::TrialInfoRequest, easy_grpc::Context ctx);
  ::easy_grpc::Stream_future<cogmentAPI::TrialListEntry> WatchTrials(::cogmentAPI::TrialListRequest,
                                                                    easy_grpc::Context ctx);
  ::cogmentAPI::VersionInfo Version(::cogmentAPI::VersionRequest, easy_grpc::Context ctx);
};

}  // namespace cogment

#endif