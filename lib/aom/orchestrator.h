#ifndef COGMENT_ORCHESTRATOR_ORCHESTRATOR_H
#define COGMENT_ORCHESTRATOR_ORCHESTRATOR_H

#include "aom/datalog/storage_interface.h"
#include "aom/services/actor_service.h"
#include "aom/services/trial_lifecycle_service.h"
#include "aom/stub_pool.h"
#include "aom/trial.h"
#include "aom/trial_params.h"
#include "aom/trial_spec.h"
#include "aom/utils.h"
#include "cogment/api/hooks.egrpc.pb.h"

#include <atomic>
#include <unordered_map>

namespace cogment {
class Orchestrator {
 public:
  Orchestrator(Trial_spec trial_spec, cogment::TrialParams default_trial_params);
  ~Orchestrator();

  // Initialization
  void add_prehook(cogment::TrialHooks::Stub_interface* prehook);
  void set_log_exporter(std::unique_ptr<Datalog_storage_interface> le);

  // Lifecycle
  Future<std::shared_ptr<Trial>> start_trial(cogment::TrialParams params, std::string user_id);
  void end_trial(const uuids::uuid& trial_id);

  // Services
  ActorService& actor_service() { return actor_service_; }
  TrialLifecycleService& trial_lifecycle_service() { return trial_lifecycle_service_; }

  // Lookups
  std::shared_ptr<Trial> get_trial(const uuids::uuid& trial_id) const;

  // Gets all running trials.
  std::vector<std::shared_ptr<Trial>> all_trials() const;

  // Semi-internal, rpc management related.
  easy_grpc::Completion_queue* client_queue() { return &client_queue_; }
  Channel_pool* channel_pool() { return &channel_pool_; }

  const cogment::TrialParams& default_trial_params() const { return default_trial_params_; };

 private:
  // Configuration
  Trial_spec trial_spec_;
  cogment::TrialParams default_trial_params_;

  // Currently existing Trials
  mutable std::mutex trials_mutex_;
  std::unordered_map<uuids::uuid, std::shared_ptr<Trial>> trials_;

  // List of trial pre-hooks to invoke before actually launching trials
  std::vector<cogment::TrialHooks::Stub_interface*> prehooks_;

  // Send trial data to this destination.
  std::unique_ptr<Datalog_storage_interface> log_exporter_;

  // Completion queue for handling requests returning from env/agent/hooks
  easy_grpc::Completion_queue client_queue_;
  Channel_pool channel_pool_;

  ActorService actor_service_;
  TrialLifecycleService trial_lifecycle_service_;

  std::atomic<int> garbage_collection_countdown_;
  void check_garbage_collection_();
  void perform_garbage_collection_();

  Future<cogment::TrialContext> perform_pre_hooks_(cogment::TrialContext ctx);
};
}  // namespace cogment
#endif