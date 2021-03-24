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

#ifndef AOM_DATALOG_GRPC_EXPORTER_H
#define AOM_DATALOG_GRPC_EXPORTER_H

#include "cogment/api/datalog.egrpc.pb.h"
#include "cogment/datalog/storage_interface.h"
#include "slt/concur/work_pool.h"

#include "easy_grpc/easy_grpc.h"

#include <fstream>
#include <memory>

namespace cogment {

class Grpc_datalog_exporter_base : public Datalog_storage_interface {
  public:
  class Trial_log : public TrialLogInterface {
    public:
    Trial_log(Grpc_datalog_exporter_base* owner, const Trial* trial);
    ~Trial_log();

    void add_sample(cogment::DatalogSample data) override;

    private:
    Grpc_datalog_exporter_base* owner_ = nullptr;
    const Trial* trial_ = nullptr;
    ::easy_grpc::Stream_future<::cogment::LogExporterSampleReply> reply_;
    std::vector<grpc_metadata> headers_;
    easy_grpc::client::Call_options options_;

    void lazy_start_stream_();
    std::optional<::easy_grpc::Stream_promise<::cogment::LogExporterSampleRequest>> output_promise_;
  };

  std::unique_ptr<TrialLogInterface> start_log(const Trial* trial) final override;

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