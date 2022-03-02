// Copyright 2022 AI Redefined Inc. <dev+cogment@ai-r.com>
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

#ifndef COGMENT_ORCHESTRATOR_DIRECTORY_H
#define COGMENT_ORCHESTRATOR_DIRECTORY_H

#include "cogment/api/directory.grpc.pb.h"
#include "cogment/stub_pool.h"

#include <string>

namespace cogment {

// Invalid characters for property names and values.
// Not all necessary; we are being extra cautious.
constexpr std::string_view INVALID_CHARACTERS = "%?&=;:/+,";

// Address returned by "get_address" method for client actors registered in directory.
// This is different than a host named "client", because a host will have a port e.g. "client:8000".
// This is a hack.
constexpr std::string_view CLIENT_ACTOR_ADDRESS = "client";

// Endpoint parsing result
struct EndpointData {
  enum SchemeType { UNKNOWN_SCHEME, GRPC, COGMENT };
  enum HostType { UNKNOWN_HOST, CLIENT, DISCOVER };
  enum PathType { UNKNOWN_PATH, EMPTY_PATH, SERVICE, ACTOR, ENVIRONMENT, DATALOG, PREHOOK, LIFECYCLE, ACTSERVICE };

  EndpointData() : scheme(UNKNOWN_SCHEME), host(UNKNOWN_HOST), path(UNKNOWN_PATH) {}
  std::string debug_string() const;

  std::string original_endpoint;  // The original endpoint and underlying base for 'std::string_view'

  SchemeType scheme;
  std::string_view address;  // Address if a GRPC scheme

  HostType host;
  PathType path;
  std::vector<std::pair<std::string_view, std::string_view>> query;  // (property, value) pairs
};

void parse_endpoint(const std::string& endpoint, EndpointData* data);

class Directory {
  using StubEntryType = std::shared_ptr<StubPool<cogmentAPI::DirectorySP>::Entry>;

public:
  void add_stub(const StubEntryType& entry);
  bool is_context_endpoint(const EndpointData& data) const;
  std::string get_address(const EndpointData& data) const;

private:
  std::vector<StubEntryType> m_stubs;
};

}  // namespace cogment
#endif