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

#ifndef COGMENT_ORCHESTRATOR_TRIAL_LIFECYCLE_SERVICE_H
#define COGMENT_ORCHESTRATOR_TRIAL_LIFECYCLE_SERVICE_H

#include "cogment/api/orchestrator.egrpc.pb.h"

namespace cogment {

class Orchestrator;
class TrialLifecycleService {
  Orchestrator* orchestrator_;

  using Trial_promise = ::easy_grpc::Stream_promise<::cogment::TrialListEntry>;
  using Trial_future = ::easy_grpc::Stream_future<::cogment::TrialListEntry>;

  public:
  using service_type = cogment::TrialLifecycle;

  TrialLifecycleService(Orchestrator* orch);

  ::easy_grpc::Future<::cogment::TrialStartReply> StartTrial(::cogment::TrialStartRequest, easy_grpc::Context ctx);
  ::cogment::TerminateTrialReply TerminateTrial(::cogment::TerminateTrialRequest, easy_grpc::Context ctx);
  ::cogment::TrialInfoReply GetTrialInfo(::cogment::TrialInfoRequest, easy_grpc::Context ctx);
  ::easy_grpc::Stream_future<::cogment::TrialListEntry> WatchTrials(::cogment::TrialListRequest,
                                                                    easy_grpc::Context ctx);
  ::cogment::VersionInfo Version(::cogment::VersionRequest, easy_grpc::Context ctx);
};

}  // namespace cogment

#endif