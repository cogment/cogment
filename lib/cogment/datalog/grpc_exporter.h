#ifndef AOM_DATALOG_GRPC_EXPORTER_H
#define AOM_DATALOG_GRPC_EXPORTER_H

#include "cogment/api/data.egrpc.pb.h"
#include "cogment/datalog/storage_interface.h"
#include "slt/concur/work_pool.h"

#include "easy_grpc/easy_grpc.h"

#include <fstream>
#include <memory>

namespace cogment {

class Grpc_datalog_exporter_base : public Datalog_storage_interface {
 public:
  class Trial_log : public Trial_log_interface {
   public:
    Trial_log(Grpc_datalog_exporter_base* owner, Trial* trial);
    ~Trial_log();

    void add_sample(cogment::DatalogSample data) override;

   private:
    Grpc_datalog_exporter_base* owner_ = nullptr;
    Trial* trial_ = nullptr;

    void lazy_start_stream_();
    std::optional<::easy_grpc::Stream_promise<::cogment::DatalogMsg>>
        output_promise_;
  };

  std::unique_ptr<Trial_log_interface> begin_trial(Trial* trial) final override;

  void set_stub(cogment::LogExporter::Stub_interface* stub) { stub_ = stub; }

 private:
  cogment::LogExporter::Stub_interface* stub_ = nullptr;
};

// Stores Data samples to a local CVS file.
class Grpc_datalog_exporter : public Grpc_datalog_exporter_base {
 public:
  Grpc_datalog_exporter(const std::string& url);

 private:
  easy_grpc::Completion_queue work_thread;
  easy_grpc::client::Unsecure_channel channel;
  cogment::LogExporter::Stub stub_impl;
};
}  // namespace cogment
#endif