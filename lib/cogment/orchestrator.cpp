#include "cogment/orchestrator.h"
#include "slt/settings.h"

namespace settings {

constexpr std::uint32_t default_garbage_collection_frequency = 10;

slt::Setting garbage_collection_frequency =
    slt::Setting_builder<std::uint32_t>()
        .with_default(default_garbage_collection_frequency)
        .with_description("Number of trials between garbage collection runs")
        .with_arg("gb_freq");
}  // namespace settings

namespace cogment {
Orchestrator::Orchestrator(Trial_spec trial_spec,
                           cogment::TrialParams default_trial_params)
    : trial_spec_(std::move(trial_spec)),
      default_trial_params_(std::move(default_trial_params)),
      env_stubs_(&channel_pool_, &client_queue_),
      agent_stubs_(&channel_pool_, &client_queue_),
      actor_service_(this),
      trial_lifecycle_service_(this) {}

Orchestrator::~Orchestrator() {}

Future<std::shared_ptr<Trial>> Orchestrator::start_trial(
    cogment::TrialParams params, std::string user_id) {
  check_garbage_collection_();

  auto new_trial = std::make_shared<Trial>(this, user_id);

  // Register the trial immediately.
  {
    std::lock_guard l(trials_mutex_);
    trials_[new_trial->id()] = new_trial;
  }

  cogment::TrialContext init_ctx;
  *init_ctx.mutable_params() = std::move(params);
  init_ctx.set_trial_id(to_string(new_trial->id()));
  init_ctx.set_user_id(user_id);

  auto final_ctx_fut = perform_pre_hooks_(std::move(init_ctx));

  return final_ctx_fut
      .then([new_trial](auto final_ctx) {
        return new_trial->configure(std::move(*final_ctx.mutable_params()));
      })
      .then([new_trial]() {
        spdlog::info("trial {} successfully initialized",
                     to_string(new_trial->id()));
        return new_trial;
      });
}

void Orchestrator::end_trial(const uuids::uuid& trial_id) {
  std::shared_ptr<Trial> trial;
  {
    std::lock_guard l(trials_mutex_);

    auto trial_ite = trials_.find(trial_id);
    if (trial_ite == trials_.end()) {
      throw std::out_of_range("unknown trial id");
    }

    trial = trial_ite->second;
    trials_.erase(trial_ite);
  }

  trial->terminate();
}

void Orchestrator::add_prehook(cogment::TrialHooks::Stub_interface* hook) {
  prehooks_.push_back(hook);
}

Future<cogment::TrialContext> Orchestrator::perform_pre_hooks_(
    cogment::TrialContext ctx) {
  aom::Promise<cogment::TrialContext> prom;
  auto result = prom.get_future();
  prom.set_value(std::move(ctx));

  // Run prehooks.
  for (auto& hook : prehooks_) {
    result =
        result.then([hook](auto p) { return hook->PreTrial(std::move(p)); });
  }

  return result;
}

void Orchestrator::set_log_exporter(
    std::unique_ptr<Datalog_storage_interface> le) {
  log_exporter_ = std::move(le);
}

void Orchestrator::check_garbage_collection_() {
  if (--garbage_collection_countdown_ <= 0) {
    garbage_collection_countdown_.store(
        settings::garbage_collection_frequency.get());
    perform_garbage_collection_();
  }
}

void Orchestrator::perform_garbage_collection_() {
  auto trials = all_trials();
  for (auto& trial : trials) {
    if (trial->is_stale()) {
      end_trial(trial->id());
    }
  }
}

std::shared_ptr<Trial> Orchestrator::get_trial(
    const uuids::uuid& trial_id) const {
  std::lock_guard l(trials_mutex_);
  return trials_.at(trial_id);
}

std::vector<std::shared_ptr<Trial>> Orchestrator::all_trials() const {
  std::lock_guard l(trials_mutex_);

  std::vector<std::shared_ptr<Trial>> result;
  result.reserve(trials_.size());

  for (const auto& t : trials_) {
    result.push_back(t.second);
  }

  return result;
}

}  // namespace cogment