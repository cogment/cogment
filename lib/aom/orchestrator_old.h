#ifndef AOM_ORCHESTRATOR_ORCHESTRATOR_H
#define AOM_ORCHESTRATOR_ORCHESTRATOR_H

#include <grpcpp/grpcpp.h>

#include "easy_grpc/completion_queue.h"

#include "cogment/api/environment.egrpc.pb.h"
#include "cogment/api/hooks.egrpc.pb.h"
#include "cogment/api/orchestrator.egrpc.pb.h"

#include "aom/datalog/storage_interface.h"
#include "aom/stub_pool.h"
#include "aom/trial.h"
#include "aom/trial_spec.h"

#include <atomic>

namespace cogment {

struct Trial_start_handler;

class Orchestrator {
  friend class Trial;

  std::vector<cogment::TrialHooks::Stub_interface*> prehooks_;
  Trial_spec trial_spec_;

  std::unique_ptr<Datalog_storage_interface> storage_ = nullptr;

  std::mutex trials_mutex_;
  std::unordered_map<uuids::uuid, std::shared_ptr<Trial>> trials_;

  cogment::TrialParams default_trial_params_;
  std::atomic<int> started_trials_ = 0;

  // Handles event completions for environments and agents
  easy_grpc::Completion_queue client_queue_;

  Channel_pool channel_pool_;
  Stub_pool<cogment::Environment> env_stubs_;
  Stub_pool<cogment::Agent> agent_stubs_;

 public:
  using service_type = cogment::Trial;

  Orchestrator(Trial_spec trial_spec, cogment::TrialParams default_trial_params,
               std::unique_ptr<Datalog_storage_interface> datalog_iface);
  ~Orchestrator();

  void add_prehook(cogment::TrialHooks::Stub_interface*);

  ::easy_grpc::Future<::cogment::TrialStartReply> Start(::cogment::TrialStartRequest);

  ::cogment::TrialStartReply Join(::cogment::TrialJoinRequest);

  ::cogment::TrialEndReply End(::cogment::TrialEndRequest);

  ::easy_grpc::Future<::cogment::TrialActionReply> Action(::cogment::TrialActionRequest);

  ::easy_grpc::Stream_future<::cogment::TrialActionReply> ActionStream(
      ::easy_grpc::Stream_future<::cogment::TrialActionRequest>);

  ::cogment::TrialFeedbackReply GiveFeedback(::cogment::TrialFeedbackRequest);

  ::cogment::TrialHeartbeatReply Heartbeat(::cogment::TrialHeartbeatRequest);

  ::cogment::VersionInfo Version(::cogment::VersionRequest);

  std::shared_ptr<Trial> get_trial(const std::string& key);

  Datalog_storage_interface* get_storage() { return storage_.get(); }

  easy_grpc::Completion_queue* client_queue() { return &client_queue_; }
  Channel_pool* channel_pool() { return &channel_pool_; }

  // private:
  Orchestrator(const Orchestrator&) = delete;
  Orchestrator(Orchestrator&&) = delete;
  Orchestrator& operator=(const Orchestrator&) = delete;
  Orchestrator& operator=(Orchestrator&&) = delete;

  void register_trial(std::shared_ptr<Trial>);

  void perform_trial_garbage_collection();
};

}  // namespace cogment

#endif