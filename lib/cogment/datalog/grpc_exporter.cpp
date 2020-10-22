#include "cogment/datalog/grpc_exporter.h"
#include "cogment/trial.h"

#include "spdlog/spdlog.h"

#include <stdexcept>

namespace cogment {

Grpc_datalog_exporter_base::Trial_log::Trial_log(Grpc_datalog_exporter_base* owner, Trial* trial)
    : owner_(owner), trial_(trial) {}

Grpc_datalog_exporter_base::Trial_log::~Trial_log() {
  if (output_promise_) {
    output_promise_->complete();
  }
}

void Grpc_datalog_exporter_base::Trial_log::lazy_start_stream_() {
  if (!output_promise_) {
    auto stream_reply = owner_->stub_->Log();
    auto& stream = std::get<0>(stream_reply);
    auto& reply = std::get<1>(stream_reply);

    // We'll just ignore whatever comes back from the log exporter service
    reply.finally([](auto) {});

    cogment::DatalogMsg msg;
    *msg.mutable_trial_params() = trial_->params();
    stream.push(std::move(msg));

    output_promise_ = std::move(stream);
  }
}

void Grpc_datalog_exporter_base::Trial_log::add_sample(cogment::DatalogSample data) {
  lazy_start_stream_();

  cogment::DatalogMsg msg;
  *msg.mutable_sample() = std::move(data);
  output_promise_->push(std::move(msg));
}

std::unique_ptr<Trial_log_interface> Grpc_datalog_exporter_base::begin_trial(Trial* trial) {
  return std::make_unique<Grpc_datalog_exporter::Trial_log>(this, trial);
}

Grpc_datalog_exporter::Grpc_datalog_exporter(const std::string& url) : channel(url, &work_thread), stub_impl(&channel) {
  set_stub(&stub_impl);

  spdlog::info("sending datalog to service running at: {}", url);
}

}  // namespace cogment