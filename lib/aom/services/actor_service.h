#ifndef COGMENT_ORCHESTRATOR_ACTOR_SERVICE_H
#define COGMENT_ORCHESTRATOR_ACTOR_SERVICE_H

#include "cogment/api/orchestrator.egrpc.pb.h"

namespace cogment {
class Orchestrator;
class ActorService {
  Orchestrator* orchestrator_;

 public:
  using service_type = cogment::ActorEndpoint;

  ActorService(Orchestrator* orch);

  ::easy_grpc::Future<::cogment::TrialJoinReply> JoinTrial(::cogment::TrialJoinRequest, easy_grpc::Context ctx);
  ::easy_grpc::Stream_future<::cogment::TrialActionReply> ActionStream(
      ::easy_grpc::Stream_future<::cogment::TrialActionRequest>, easy_grpc::Context ctx);
  ::easy_grpc::Future<::cogment::TrialActionReply> Action(::cogment::TrialActionRequest, easy_grpc::Context ctx);
  ::easy_grpc::Future<::cogment::TrialHeartbeatReply> Heartbeat(::cogment::TrialHeartbeatRequest,
                                                                easy_grpc::Context ctx);
  ::easy_grpc::Future<::cogment::TrialFeedbackReply> GiveFeedback(::cogment::TrialFeedbackRequest,
                                                                  easy_grpc::Context ctx);
  ::cogment::VersionInfo Version(::cogment::VersionRequest, easy_grpc::Context ctx);
};
}  // namespace cogment

#endif