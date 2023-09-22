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

#ifndef COGMENT_ORCHESTRATOR_DIRECTORY_H
#define COGMENT_ORCHESTRATOR_DIRECTORY_H

#include "cogment/api/directory.grpc.pb.h"
#include "cogment/stub_pool.h"
#include "cogment/endpoint.h"

#include <string>

namespace cogment {

// Invalid characters for property names and values.
// Not all necessary; we are being extra cautious.
constexpr std::string_view INVALID_CHARACTERS = "%?&=;:/+,#@[]";

// Address returned by "inquire_address" method for client actors registered in directory.
// This is different than an address with a hostname "client", because that will have a port e.g. "client:8000".
// This is a hack.
constexpr std::string_view CLIENT_ACTOR_ADDRESS = "client";

// Current limitations:
// - Uses only the first stub/directory
// - Uses only the first reply from inquiries
// - Registers and deregisters one service at a time
class Directory {
  using StubEntryType = std::shared_ptr<StubPool<cogmentAPI::DirectorySP>::Entry>;

public:
  enum ServiceType {
    UNKNOWN = cogmentAPI::ServiceType::UNKNOWN_SERVICE,
    LIFE_CYCLE = cogmentAPI::ServiceType::TRIAL_LIFE_CYCLE_SERVICE,
    CLIENT_ACTOR = cogmentAPI::ServiceType::CLIENT_ACTOR_CONNECTION_SERVICE,
    ACTOR = cogmentAPI::ServiceType::ACTOR_SERVICE,
    ENVIRONMENT = cogmentAPI::ServiceType::ENVIRONMENT_SERVICE,
    PRE_HOOK = cogmentAPI::ServiceType::PRE_HOOK_SERVICE,
    DATALOG = cogmentAPI::ServiceType::DATALOG_SERVICE,
    DATASTORE = cogmentAPI::ServiceType::DATASTORE_SERVICE,
    MODEL_REGISTRY = cogmentAPI::ServiceType::MODEL_REGISTRY_SERVICE,
    DIRECTORY = cogmentAPI::ServiceType::DIRECTORY_SERVICE,
    OTHER = cogmentAPI::ServiceType::OTHER_SERVICE,
  };

  struct RegisteredService {
    uint64_t service_id;
    std::string secret;

    RegisteredService() : service_id(0) {}
    RegisteredService(uint64_t id, std::string sec) : service_id(id), secret(sec) {}
  };

  struct InquiredAddress {
    std::string address;
    bool ssl;

    InquiredAddress() {}
    InquiredAddress(std::string addr, bool ssl) : address(addr), ssl(ssl) {}
  };

  void add_stub(const StubEntryType& stub) { m_stubs.emplace_back(stub); }
  void set_auth_token(std::string_view token) { m_auth_token = token; }
  bool is_set() const { return (!m_stubs.empty()); }

  InquiredAddress inquire_address(std::string_view service_name, const EndpointData& data) const;
  RegisteredService register_host(std::string_view host, uint16_t port, bool ssl, ServiceType type,
                                  const std::vector<PropertyView>& properties);
  void deregister_service(const RegisteredService&);

private:
  cogmentAPI::ServiceType get_service_type(const EndpointData& data, uint64_t* id) const;
  InquiredAddress get_address_from_directory(grpc::ClientContext&, cogmentAPI::InquireRequest&,
                                             const EndpointData&) const;

  std::vector<StubEntryType> m_stubs;
  std::string m_auth_token;
};

}  // namespace cogment
#endif