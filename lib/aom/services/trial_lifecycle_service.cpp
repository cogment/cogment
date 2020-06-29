#include "aom/services/trial_lifecycle_service.h"

#include "aom/orch_config.h"
#include "aom/orchestrator.h"

namespace cogment {

TrialLifecycleService::TrialLifecycleService(Orchestrator* orch) : orchestrator_(orch) {}

::easy_grpc::Future<::cogment::TrialStartReply> TrialLifecycleService::StartTrial(::cogment::TrialStartRequest req,
                                                                                  easy_grpc::Context ctx) {
  auto params = orchestrator_->default_trial_params();

  // Apply config override if provided
  if (req.has_config()) {
    params.mutable_trial_config()->set_content(req.config().content());
  }

  return orchestrator_->start_trial(std::move(params), req.user_id()).then([](std::shared_ptr<Trial> trial) {
    ::cogment::TrialStartReply reply;

    reply.set_trial_id(to_string(trial->id()));

    for (const auto& actor : trial->actors()) {
      reply.add_actor_class_idx(actor->actor_class->index);
    }
    return reply;
  });
}

::cogment::TerminateTrialReply TrialLifecycleService::TerminateTrial(::cogment::TerminateTrialRequest,
                                                                     easy_grpc::Context ctx) {
  orchestrator_->end_trial(uuids::uuid::from_string(ctx.get_client_header("trial-id")));
  return {};
}

::easy_grpc::Future<::cogment::MessageDispatchReply> TrialLifecycleService::SendMessage(
    ::cogment::MasterMessageDispatchRequest, easy_grpc::Context ctx) {
  // TODO.
  return {};
}

::cogment::TrialInfoReply TrialLifecycleService::TrialInfo(::cogment::TrialInfoRequest, easy_grpc::Context ctx) {
  ::cogment::TrialInfoReply result;
  auto add_trial = [&](Trial* trial) {
    auto trial_info = result.add_trial();

    auto trial_lock = trial->lock();
    trial_info->set_trial_id(to_string(trial->id()));
    trial_info->set_state(get_trial_state_proto(trial->state()));
  };

  auto trial_id_str = ctx.get_client_header("trial-id");

  if (trial_id_str == "all_trials") {
    auto trials = orchestrator_->all_trials();
    for (auto& trial : trials) {
      add_trial(trial.get());
    }
  } else {
    auto trial_id = uuids::uuid::from_string(trial_id_str);
    auto trial = orchestrator_->get_trial(trial_id);
    add_trial(trial.get());
  }

  return result;
}

::cogment::VersionInfo TrialLifecycleService::Version(::cogment::VersionRequest, easy_grpc::Context ctx) {
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
