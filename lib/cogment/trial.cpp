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

#include "cogment/trial.h"
#include "cogment/orchestrator.h"

#include "cogment/agent.h"

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
    : orchestrator_(orch), id_(id_generator_()), user_id_(std::move(user_id)), state_(Trial_state::initializing) {
  refresh_activity();
}

Trial::~Trial() {}

std::lock_guard<std::mutex> Trial::lock() { return std::lock_guard(lock_); }

const uuids::uuid& Trial::id() const { return id_; }

const std::string& Trial::user_id() const { return user_id_; }

const std::vector<std::unique_ptr<Actor>>& Trial::actors() const { return actors_; }

void Trial::fill_env_start_request(::cogment::EnvStartRequest* io_req) {
  if (params_.environment().has_config()) {
    *io_req->mutable_config() = params_.environment().config();
  }

  if (params_.has_trial_config()) {
    *io_req->mutable_trial_config() = params_.trial_config();
  }
}

Future<void> Trial::configure(cogment::TrialParams params) {
  params_ = std::move(params);

  grpc_metadata trial_header;
  trial_header.key = grpc_slice_from_static_string("trial_id");
  trial_header.value = grpc_slice_from_copied_string(to_string(id_).c_str());
  headers_.push_back(trial_header);
  call_options_.headers = &headers_;

  env_stub_ = orchestrator_->env_pool()->get_stub(params_.environment().endpoint());

  ::cogment::EnvStartRequest env_start_req;
  fill_env_start_request(&env_start_req);

  std::vector<aom::Future<void>> agents_ready;
  for (const auto& actor_info : params_.actors()) {
    auto url = actor_info.endpoint();
    const auto& actor_class = orchestrator_->get_trial_spec().get_actor_class(actor_info.actor_class());
    env_start_req.add_actor_class_idx(actor_class.index);

    if (url == "connect") {
      //      auto human_actor = std::make_unique<Human>(trial_id);
      //      human_actor->actor_class = &owner_->trial_spec_.actor_classes[class_id];
      //      actors_.push_back(std::move(human_actor));
    }
    else {
      std::optional<std::string> config;
      if (actor_info.has_config()) {
        config = actor_info.config().content();
      }
      auto stub_entry = orchestrator_->agent_pool()->get_stub(url);
      auto agent_actor = std::make_unique<Agent>(this, actors_.size(), &actor_class, stub_entry, config);

      agents_ready.push_back(agent_actor->init());
      actors_.push_back(std::move(agent_actor));
    }
  }

  auto env_ready = (*env_stub_)->Start(std::move(env_start_req), call_options_).then_expect([](auto rep) {
    if (!rep) {
      spdlog::error("failed to connect to environment");
      try {
        std::rethrow_exception(rep.error());
      } catch (std::exception& e) {
        spdlog::error(e.what());
      }
    }

    return rep;
  });

  return join(env_ready, concat(agents_ready.begin(), agents_ready.end())).then([](auto) {
    // everyone is ready...
  });
}

void Trial::terminate() { state_ = Trial_state::terminating; }

const cogment::TrialParams& Trial::params() const { return params_; }

Trial_state Trial::state() const { return state_; }

void Trial::refresh_activity() { last_activity_ = std::chrono::steady_clock::now(); }

bool Trial::is_stale() const {
  bool stale = std::chrono::steady_clock::now() - last_activity_ > std::chrono::seconds(params_.max_inactivity());
  return params_.max_inactivity() > 0 && stale;
}
}  // namespace cogment