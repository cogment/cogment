#ifndef COGMENT_ORCHESTRATOR_TRIAL_LIFECYCLE_SERVICE_H
#define COGMENT_ORCHESTRATOR_TRIAL_LIFECYCLE_SERVICE_H

#include "cogment/api/orchestrator.egrpc.pb.h"

namespace cogment {

class Orchestrator;
class TrialLifecycleService {
  Orchestrator* orchestrator_;

  public:
  using service_type = cogment::TrialLifecycle;

  TrialLifecycleService(Orchestrator* orch);

  ::easy_grpc::Future<::cogment::TrialStartReply> StartTrial(::cogment::TrialStartRequest, easy_grpc::Context ctx);
  ::cogment::TerminateTrialReply TerminateTrial(::cogment::TerminateTrialRequest, easy_grpc::Context ctx);
  ::easy_grpc::Future<::cogment::MessageDispatchReply> SendMessage(::cogment::MasterMessageDispatchRequest,
                                                                   easy_grpc::Context ctx);
  ::cogment::TrialInfoReply TrialInfo(::cogment::TrialInfoRequest, easy_grpc::Context ctx);
  ::cogment::VersionInfo Version(::cogment::VersionRequest, easy_grpc::Context ctx);
};
}  // namespace cogment

#endif