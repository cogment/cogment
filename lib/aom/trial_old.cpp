#include "aom/trial.h"

#include "aom/agent.h"
#include "aom/human.h"
#include "aom/orchestrator.h"

#include "spdlog/spdlog.h"

namespace cogment {
uuids::uuid_system_generator Trial::id_generator_;

void recompute_reward(cogment::Reward* reward) {
  float value = 0.0f;

  auto feedback_count = reward->feedbacks_size();
  if (feedback_count > 0) {
    float total_c = 0.0f;
    for (int i = 0; i < feedback_count; ++i) {
      auto v = reward->feedbacks(i).value();
      auto c = reward->feedbacks(i).confidence();

      value += v * c;
      total_c += c;
    }

    value /= total_c;
  }

  reward->set_value(value);
  reward->set_confidence(1.0f);
}

Trial::Trial(Orchestrator* owner, std::string user_id)
    : id_(id_generator_()), user_id_(std::move(user_id)), owner_(owner) {
  spdlog::info("creating trial: {}", to_string(id_));
  log_interface_ = owner->get_storage()->begin_trial(this);
  refresh_activity();
}

Trial::~Trial() {
  spdlog::info("tearing down trial: {}", to_string(id_));

  if (!data_.empty()) {
    log_interface_->add_samples(std::move(data_));
  }
}

::easy_grpc::Future<void> Trial::configure(cogment::TrialParams cfg) {
  auto trial_id = to_string(id_);
  params_ = std::move(cfg);

  grpc_metadata trial_header;
  trial_header.key = grpc_slice_from_static_string("trial_id");
  trial_header.value = grpc_slice_from_copied_string(trial_id.c_str());
  headers_.push_back(trial_header);

  call_options_.headers = &headers_;

  // Launch the environment
  env_stub_ = owner_->env_stubs_.get_stub(params_.environment().endpoint());

  // Launch the environment, in parallell with the agent launches
  std::vector<aom::Future<void>> agents_ready;
  ::cogment::EnvStartRequest env_start_req;

  env_start_req.set_trial_id(trial_id);

  if (params_.environment().has_config()) {
    *env_start_req.mutable_config() = params_.environment().config();
  }

  if (params_.has_trial_config()) {
    *env_start_req.mutable_trial_config() = params_.trial_config();
  }

  // List of actors, organized per actor class
  std::vector<std::vector<std::unique_ptr<Actor>>> actors_per_class(owner_->trial_spec_.actor_classes.size());
  actor_counts_.resize(owner_->trial_spec_.actor_classes.size(), 0);

  // Launch the actors, both AI and human
  for (const auto& actor_info : params_.actors()) {
    auto class_id = owner_->trial_spec_.get_class_id(actor_info.actor_class());
    auto url = actor_info.endpoint();
    actor_counts_[class_id] += 1;

    if (url == "human") {
      auto human_actor = std::make_unique<Human>(trial_id);
      human_actor->actor_class = &owner_->trial_spec_.actor_classes[class_id];
      actors_per_class.at(class_id).push_back(std::move(human_actor));
    } else {
      std::optional<std::string> config;
      if (actor_info.has_config()) {
        config = actor_info.config().content();
      }
      auto stub_entry = owner_->agent_stubs_.get_stub(url);
      auto actor = std::make_unique<Agent>(this, stub_entry, config);
      actor->actor_class = &owner_->trial_spec_.actor_classes[class_id];
      actors_per_class.at(class_id).push_back(std::move(actor));
    }
  }

  // Assign actor ids, and perform initialization
  int aid = 0;
  for (auto& a_class : actors_per_class) {
    for (auto& actor : a_class) {
      actor->set_actor_id(aid++);
      agents_ready.push_back(actor->init());

      actors_.push_back(std::move(actor));
    }
  }

  for (auto i : actor_counts_) {
    env_start_req.add_actor_counts(i);
  }

  auto env_ready = (*env_stub_)->Start(std::move(env_start_req), call_options_);

  return join(env_ready, concat(agents_ready.begin(), agents_ready.end())).then([this](cogment::EnvStartReply env_rep) {
    // Everyone is ready,
    latest_observation_ = std::move(*env_rep.mutable_observation_set());
    begin();
  });
}

void Trial::begin() { gather_actions(); }

void Trial::terminate() {
  mark_terminating();

  auto self_lock = shared_from_this();

  cogment::EnvEndRequest env_req;
  env_req.set_trial_id(to_string(id()));
  (*env_stub_)->End(env_req, call_options_).finally([self_lock](auto) {});

  for (auto& actor : actors_) {
    actor->terminate();
  }
}

int Trial::human_actor_id() const {
  int result = 0;
  for (auto& actor : actors_) {
    if (actor->is_human()) {
      return result;
    }
    ++result;
  }
  return -1;
}

void Trial::populate_observation(int actor_id, ::cogment::Observation* obs) {
  auto obs_index = latest_observation_.actors_map(actor_id);
  *obs->mutable_data() = latest_observation_.observations(obs_index);

  obs->set_tick_id(latest_observation_.tick_id());
  *obs->mutable_timestamp() = latest_observation_.timestamp();
}

// Removes from an observation set all the fields that were requested
// to be cleared from it.
void Trial::strip_observation(cogment::ObservationSet& observation_set) {
  std::set<int> processed;

  for (std::size_t i = 0; i < actors_.size(); ++i) {
    int index = observation_set.actors_map(i);
    if (processed.count(index)) {
      continue;
    }
    processed.insert(index);

    auto actor_class = actors_[i]->actor_class;

    auto& data = *observation_set.mutable_observations(index);
    if (data.snapshot()) {
      if (actor_class->cleared_observation_fields.size() > 0) {
        assert(actors_[i]->actor_class->observation_space_prototype);
        auto msg = actors_[i]->actor_class->observation_space_prototype->New();
        msg->ParseFromString(data.content());
        auto refl = msg->GetReflection();
        for (auto f : actor_class->cleared_observation_fields) {
          refl->ClearField(msg, f);
        }
        msg->SerializeToString(data.mutable_content());
      }
    } else {
      if (actor_class->cleared_delta_fields.size() > 0) {
        assert(actors_[i]->actor_class->observation_delta_prototype);
        auto msg = actors_[i]->actor_class->observation_delta_prototype->New();
        msg->ParseFromString(data.content());
        auto refl = msg->GetReflection();
        for (auto f : actor_class->cleared_delta_fields) {
          refl->ClearField(msg, f);
        }
        msg->SerializeToString(data.mutable_content());
      }
    }
  }
}

void Trial::prepare_actions() {
  data_.push_back({});
  data_.back().mutable_observations()->CopyFrom(latest_observation_);

  strip_observation(*data_.back().mutable_observations());

  data_.back().set_trial_id(to_string(id_));
  data_.back().set_user_id(user_id_);

  data_.back().mutable_actions()->Reserve(actors_.size());
  data_.back().mutable_rewards()->Reserve(actors_.size());

  for (unsigned int i = 0; i < actors_.size(); ++i) {
    data_.back().mutable_actions()->Add();
    data_.back().mutable_rewards()->Add();
  }

  mark_ready();

  int actor_count = int(actors_.size());
  if (latest_observation_.actors_map_size() != actor_count) {
    spdlog::error("Trial {}, env generated observations for {} actors, but expected {}", to_string(id_),
                  latest_observation_.actors_map_size(), actor_count);
    throw std::runtime_error("actor count mismatch");
  }

  pending_decisions_ = actor_count;
  actions_.resize(actor_count);
}

void Trial::gather_actions() {
  prepare_actions();
  auto self = shared_from_this();

  for (int i = 0; i < latest_observation_.actors_map_size(); ++i) {
    cogment::Observation observation;

    populate_observation(i, &observation);

    auto actor_decision_fut = actors_[i]->request_decision(std::move(observation));

    actor_decision_fut.finally([self, this, i](auto rep) {
      if (rep.has_value()) {
        data_.back().mutable_actions(i)->CopyFrom(*rep);
      }

      actions_[i] = std::move(rep);

      if (--pending_decisions_ == 0) {
        // Send the actions
        dispatch_update();
      }
    });
  }
}

void Trial::send_final_observation() {
  prepare_actions();
  auto self = shared_from_this();

  for (int i = 0; i < latest_observation_.actors_map_size(); ++i) {
    cogment::Observation observation;

    populate_observation(i, &observation);
    actors_[i]->send_final_observation(std::move(observation));
  }
}

void Trial::consume_feedback(const ::google::protobuf::RepeatedPtrField<::cogment::Feedback>& feedbacks) {
  refresh_activity();

  for (const auto& feedback : feedbacks) {
    auto time = feedback.tick_id();

    if (time < 0) {
      time = trial_steps_ - 1;
    }

    auto* reward = data_.at(time).mutable_rewards(feedback.actor_id());
    auto new_fb = reward->add_feedbacks();
    new_fb->CopyFrom(feedback);
    new_fb->set_tick_id(time);

    // TODO: this is perhaps a bit trigger happy...
    recompute_reward(reward);
  }
}

void Trial::dispatch_update() {
  if (state_ == Trial_state::Terminating) return;

  auto self = shared_from_this();

  // Send the previously acquired rewards...
  if (data_.size() > 1) {
    const auto& time_sample = data_[data_.size() - 2];
    for (int i = 0; i < time_sample.rewards_size(); ++i) {
      actors_[i]->dispatch_reward(data_.size() - 2, time_sample.rewards(i));
    }
  }

  cogment::EnvUpdateRequest req;
  req.set_trial_id(to_string(id_));
  req.set_reply_with_snapshot(false);
  for (const auto& act : actions_) {
    if (act.has_value()) {
      req.mutable_action_set()->add_actions(act->content());
    } else {
      req.mutable_action_set()->add_actions("");
    }
  }

  auto env_update_reply = (*env_stub_)->Update(std::move(req), call_options_);
  env_update_reply.finally([self, this](auto rep) {
    try {
      if (rep.has_value()) {
        latest_observation_ = std::move(*rep->mutable_observation_set());
        ++trial_steps_;
        consume_feedback(rep->feedbacks());

        if ((params_.max_steps() != 0 && trial_steps_ >= params_.max_steps())) {
          ::cogment::TrialEndRequest req;
          req.set_trial_id(to_string(id()));
          owner_->End(req);

        } else {
          const bool is_going_to_end = rep->end_trial();
          if (is_going_to_end) {
            send_final_observation();

            ::cogment::TrialEndRequest req;
            req.set_trial_id(to_string(id()));
            owner_->End(req);
          } else {
            gather_actions();
          }
        }
      } else {
        spdlog::error("TODO: handle this");
      }
    } catch (const std::exception& e) {
      spdlog::error(e.what());
    }
  });
}

::easy_grpc::Future<::cogment::TrialActionReply> Trial::user_acted(cogment::TrialActionRequest user_req) {
  refresh_activity();

  int aid = user_req.actor_id();

  auto self = this->shared_from_this();
  auto& actor = actors_.at(aid);

  data_.back().mutable_actions(human_actor_id())->CopyFrom(user_req.action());

  return actor->user_acted(std::move(user_req));
}

void Trial::heartbeat() { refresh_activity(); }

void Trial::refresh_activity() { last_activity_ = std::chrono::steady_clock::now(); }

bool Trial::is_stale() {
  auto dt = std::chrono::duration_cast<std::chrono::seconds>(std::chrono::steady_clock::now() - last_activity_);
  bool stale = std::chrono::steady_clock::now() - last_activity_ > std::chrono::seconds(params_.max_inactivity());
  spdlog::info("stale check: {} vs {}, {}", params_.max_inactivity(), dt.count(), stale);
  return params_.max_inactivity() > 0 && stale;
}

}  // namespace cogment