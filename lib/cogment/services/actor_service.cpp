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