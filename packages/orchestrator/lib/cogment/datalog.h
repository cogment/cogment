// Copyright 2023 AI Redefined Inc. <dev+cogment@ai-r.com>
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

#ifndef COGMENT_ORCHESTRATOR_DATALOG_H
#define COGMENT_ORCHESTRATOR_DATALOG_H

#include "cogment/actor.h"
#include "cogment/stub_pool.h"

#include "cogment/api/datalog.grpc.pb.h"
#include "cogment/api/common.pb.h"

#include <bitset>

namespace cogment {

class Trial;

class DatalogService {
public:
  virtual ~DatalogService() {}
  virtual void start(Trial* trial) = 0;
  virtual void add_sample(cogmentAPI::DatalogSample&& data) = 0;
};

class DatalogServiceNull : public DatalogService {
public:
  void start(Trial* trial) override {}
  void add_sample(cogmentAPI::DatalogSample&& data) override {}
};

class DatalogServiceImpl : public DatalogService {
  using StubEntryType = std::shared_ptr<StubPool<cogmentAPI::DatalogSP>::Entry>;

public:
  DatalogServiceImpl(StubEntryType stub_entry);
  ~DatalogServiceImpl();

  void start(Trial* trial) override;
  void add_sample(cogmentAPI::DatalogSample&& data) override;

private:
  static constexpr size_t NB_BITS = 7;
  void dispatch_sample(cogmentAPI::DatalogSample&& data);

  StubEntryType m_stub_entry;
  std::unique_ptr<grpc::ClientReaderWriter<cogmentAPI::RunTrialDatalogInput, cogmentAPI::RunTrialDatalogOutput>>
      m_stream;
  grpc::ClientContext m_context;
  bool m_stream_valid;
  bool m_error_reported;
  Trial* m_trial;
  std::future<void> m_incoming_thread;
  std::bitset<NB_BITS> m_exclude_fields;
};

}  // namespace cogment

#endif
