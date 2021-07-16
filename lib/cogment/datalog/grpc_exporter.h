// Copyright 2021 AI Redefined Inc. <dev+cogment@ai-r.com>
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

#include "easy_grpc/easy_grpc.h"

#include <vector>

namespace cogment {

class GrpcDatalogExporterBase : public DatalogStorageInterface {
public:
  class Trial_log : public TrialLogInterface {
  public:
    Trial_log(GrpcDatalogExporterBase* owner, const Trial* trial);
    ~Trial_log();

    void add_sample(cogmentAPI::DatalogSample&& data) override;

  private:
    void m_lazy_start_stream();

    GrpcDatalogExporterBase* m_owner = nullptr;
    const Trial* m_trial = nullptr;
    ::easy_grpc::Stream_future<cogmentAPI::LogExporterSampleReply> m_reply;
    std::vector<grpc_metadata> m_headers;
    easy_grpc::client::Call_options m_options;

    std::optional<easy_grpc::Stream_promise<cogmentAPI::LogExporterSampleRequest>> m_output_promise;
    std::promise<void> m_stream_end_prom;
    std::future<void> m_stream_end_fut;
  };

  std::unique_ptr<TrialLogInterface> start_log(const Trial* trial) final override;

  void set_stub(cogmentAPI::LogExporterSP::Stub_interface* stub) { m_stub = stub; }

private:
  cogmentAPI::LogExporterSP::Stub_interface* m_stub = nullptr;
};

// Stores Data samples to a local CVS file.
class GrpcDatalogExporter : public GrpcDatalogExporterBase {
public:
  GrpcDatalogExporter(const std::string& url);

private:
  easy_grpc::Completion_queue m_work_thread;
  easy_grpc::client::Unsecure_channel m_channel;
  cogmentAPI::LogExporterSP::Stub m_stub_impl;
};
}  // namespace cogment
#endif