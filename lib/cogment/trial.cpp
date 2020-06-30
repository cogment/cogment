#include "cogment/trial.h"

namespace cogment {

const char* get_trial_state_string(Trial_state s) {
  switch (s) {
    case Trial_state::initializing:
      return "initializing";
    case Trial_state::pending:
      return "pending";
    case Trial_state::running:
      return "running";
    case Trial_state::terminating:
      return "terminating";
    case Trial_state::ended:
      return "ended";
  }

  throw std::out_of_range("unknown trial state");
}

cogment::TrialState get_trial_state_proto(Trial_state s) {
  switch (s) {
    case Trial_state::initializing:
      return cogment::INITIALIZING;
    case Trial_state::pending:
      return cogment::PENDING;
    case Trial_state::running:
      return cogment::RUNNING;
    case Trial_state::terminating:
      return cogment::TERMINATING;
    case Trial_state::ended:
      return cogment::ENDED;
  }

  throw std::out_of_range("unknown trial state");
}

uuids::uuid_system_generator Trial::id_generator_;

Trial::Trial(Orchestrator* orch, std::string user_id)
    : orchestrator_(orch),
      id_(id_generator_()),
      user_id_(std::move(user_id)),
      state_(Trial_state::initializing) {
  refresh_activity();
}

Trial::~Trial() {}

std::lock_guard<std::mutex> Trial::lock() { return std::lock_guard(lock_); }

const uuids::uuid& Trial::id() const { return id_; }

const std::string& Trial::user_id() const { return user_id_; }

const std::vector<std::unique_ptr<Actor>>& Trial::actors() const {
  return actors_;
}

Future<void> Trial::configure(cogment::TrialParams params) {
  params_ = std::move(params);

  // TODO: launch the on-demand agents and the environment.
  // This is not done as part of this MR in order to limit its scope.
  Promise<void> all_done;
  auto result = all_done.get_future();
  all_done.set_value();

  return result;
}

void Trial::terminate() { state_ = Trial_state::terminating; }

const cogment::TrialParams& Trial::params() const { return params_; }

Trial_state Trial::state() const { return state_; }

void Trial::refresh_activity() {
  last_activity_ = std::chrono::steady_clock::now();
}

bool Trial::is_stale() const {
  auto dt = std::chrono::duration_cast<std::chrono::seconds>(
      std::chrono::steady_clock::now() - last_activity_);
  bool stale = std::chrono::steady_clock::now() - last_activity_ >
               std::chrono::seconds(params_.max_inactivity());
  return params_.max_inactivity() > 0 && stale;
}
}  // namespace cogment