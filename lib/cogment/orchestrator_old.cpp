#include <memory>
#include <mutex>
#include "aom/base64.h"
#include "aom/orch_config.h"
#include "aom/orchestrator.h"
#include "spdlog/spdlog.h"

namespace cogment {

namespace {
const int garbage_collection_frequency = 10;
::cogment::TrialStartReply buildTrialStartReply(Trial& trial, int actor_id);
}  // namespace

using cogment::AgentStartRequest;
using cogment::EnvEndRequest;
using cogment::EnvStartRequest;
using cogment::TrialStartRequest;

using cogment::AgentStartReply;
using cogment::EnvStartReply;
using cogment::TrialStartReply;

Orchestrator::Orchestrator(Trial_spec trial_spec, cogment::TrialParams default_trial_params,
                           std::unique_ptr<Datalog_storage_interface> datalog_iface)
    : trial_spec_(std::move(trial_spec)),
      storage_(std::move(datalog_iface)),
      default_trial_params_(std::move(default_trial_params)),
      env_stubs_(&channel_pool_, &client_queue_),
      agent_stubs_(&channel_pool_, &client_queue_) {
  if (!storage_) {
    storage_ = Datalog_storage_interface::create("none", {});
  }
}

Orchestrator::~Orchestrator() {
  std::lock_guard l(trials_mutex_);
  for (auto& t : trials_) {
    t.second->terminate();
  }
}

void Orchestrator::add_prehook(cogment::TrialHooks::Stub_interface* hook) { prehooks_.push_back(hook); }

::easy_grpc::Future<TrialStartReply> Orchestrator::Start(TrialStartRequest req) {
  SPDLOG_TRACE("Orchestrator::Start");

  if (++started_trials_ == garbage_collection_frequency) {
    started_trials_.store(0);
    perform_trial_garbage_collection();
  }

  auto new_trial = std::make_shared<Trial>(this, req.user_id());

  aom::Future<cogment::TrialContext> trial_params;
  {
    aom::Promise<cogment::TrialContext> prom;
    trial_params = prom.get_future();

    cogment::TrialContext ctx;
    *ctx.mutable_params() = default_trial_params_;
    ctx.set_trial_id(to_string(new_trial->id()));
    ctx.set_user_id(req.user_id());
    if (req.has_config()) {
      ctx.mutable_params()->mutable_trial_config()->set_content(req.config().content());
    }
    prom.set_value(ctx);
  }

  // Run prehooks.
  for (auto& hook : prehooks_) {
    trial_params = trial_params.then([this, hook](auto p) { return hook->PreTrial(std::move(p)); });
  }

  return trial_params
      .then([new_trial](auto params) { return new_trial->configure(std::move(*params.mutable_params())); })
      .then([this, new_trial]() {
        // Make the trial persist
        spdlog::info("trial {} successfully initialized", to_string(new_trial->id()));

        {
          std::lock_guard l(trials_mutex_);
          trials_[new_trial->id()] = new_trial;
        }

        new_trial->mark_ready();

        return buildTrialStartReply(*new_trial, new_trial->human_actor_id());
      });
}

::cogment::TrialStartReply Orchestrator::Join(::cogment::TrialJoinRequest req) {
  SPDLOG_TRACE("Orchestrator::Join");

  auto trial_id = req.trial_id();
  auto actor_id = req.actor_id();

  std::shared_ptr<Trial> trial;
  try {
    std::lock_guard l(trials_mutex_);
    trial = get_trial(trial_id);
  } catch (...) {
    throw std::runtime_error("Trial not found");
  }
  return buildTrialStartReply(*trial, actor_id);
}

namespace {
::cogment::TrialStartReply buildTrialStartReply(Trial& trial, int actor_id) {
  TrialStartReply response;

  response.set_actor_id(actor_id);
  response.set_trial_id(to_string(trial.id()));

  for (auto c : trial.actor_counts()) {
    response.add_actor_counts(c);
  }

  // If we are running a human-less trial, then there is no point in
  // returning the initial observation.
  if (actor_id != -1) {
    SPDLOG_TRACE("With human: ", actor_id);
    trial.populate_observation(actor_id, response.mutable_observation());
  }

  return response;
}
}  // namespace

::cogment::TrialEndReply Orchestrator::End(::cogment::TrialEndRequest request) {
  SPDLOG_TRACE("Orchestrator::End");
  std::lock_guard l(trials_mutex_);

  auto trial_id = request.trial_id();
  auto trial = get_trial(trial_id);

  trial->terminate();

  // This won't delete the trial immediately, only once terminate has been
  // completed will the trial truly be over.
  trials_.erase(uuids::uuid::from_string(trial_id));

  return {};
}

::easy_grpc::Future<::cogment::TrialActionReply> Orchestrator::Action(::cogment::TrialActionRequest request) {
  SPDLOG_TRACE("Orchestrator::Action");
  std::unique_lock l(trials_mutex_);

  auto trial = get_trial(request.trial_id());

  if (trial->state() != Trial_state::Ready) {
    throw easy_grpc::error::unavailable("Trial is busy");
  }

  trial->mark_busy();
  l.unlock();
  return trial->user_acted(request);
}

::easy_grpc::Stream_future<::cogment::TrialActionReply> Orchestrator::ActionStream(
    ::easy_grpc::Stream_future<::cogment::TrialActionRequest> request) {
  // This is a bridge implementation of ActionStream, it will ostensibly
  // behave identically to sending a sequence of individual Action() messages.

  auto prom = std::make_shared<::easy_grpc::Stream_promise<::cogment::TrialActionReply>>();
  auto result = prom->get_future();

  // For each incoming message
  request
      .for_each([this, prom](auto req) mutable {
        // Delegate to the Action() method
        Action(std::move(req)).finally([prom](auto rep) {
          // And put the reply in the outgoing stream
          if (rep) {
            prom->push(std::move(*rep));
          }
          else {
            prom->set_exception(rep.error());
          }
        });
      })
      .finally([prom](auto status) {
        if (status) {
          prom->complete();
        }
        else {
          prom->set_exception(status.error());
        }
      });

  return result;
}

::cogment::TrialFeedbackReply Orchestrator::GiveFeedback(::cogment::TrialFeedbackRequest request) {
  SPDLOG_TRACE("Orchestrator::GiveFeedback");
  std::unique_lock l(trials_mutex_);

  auto trial = get_trial(request.trial_id());

  l.unlock();

  trial->consume_feedback(request.feedbacks());

  return {};
}

::cogment::TrialHeartbeatReply Orchestrator::Heartbeat(::cogment::TrialHeartbeatRequest request) {
  SPDLOG_TRACE("Orchestrator::Heartbeat");
  std::unique_lock l(trials_mutex_);

  auto trial = get_trial(request.trial_id());

  l.unlock();

  trial->heartbeat();

  return {};
}

::cogment::VersionInfo Orchestrator::Version(::cogment::VersionRequest) {
  ::cogment::VersionInfo result;
  auto v = result.add_versions();

  v->set_name("orchestrator");
  v->set_version(AOM_ORCHESTRATOR_VERSION);

  return result;
}

void Orchestrator::register_trial(std::shared_ptr<Trial> t) {
  std::lock_guard l(trials_mutex_);
  trials_.emplace(t->id(), t);
}

std::shared_ptr<Trial> Orchestrator::get_trial(const std::string& key) {
  return trials_.at(uuids::uuid::from_string(key));
}

void Orchestrator::perform_trial_garbage_collection() {
  std::unique_lock l(trials_mutex_);
  auto trials = trials_;
  l.unlock();
  for (auto& trial : trials) {
    if (trial.second->is_stale()) {
      ::cogment::TrialEndRequest req;
      req.set_trial_id(to_string(trial.second->id()));

      End(req);
    }
  }
}

}  // namespace cogment
