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

#ifndef COGMENT_ORCHESTRATOR_ENDPOINT_H
#define COGMENT_ORCHESTRATOR_ENDPOINT_H

#include "cogment/utils.h"

#include <string>
#include <vector>

namespace cogment {

// Query Property names
constexpr std::string_view ACTOR_CLASS_PROPERTY_NAME("__actor_class");
constexpr std::string_view IMPLEMENTATION_PROPERTY_NAME("__implementation");
constexpr std::string_view SERVICE_ID_PROPERTY_NAME("__id");
constexpr std::string_view AUTHENTICATION_TOKEN_PROPERTY_NAME("__authentication-token");

// Endpoint parsing result
struct EndpointData {
  enum SchemeType { UNKNOWN_SCHEME, GRPC, COGMENT };
  enum HostType { UNKNOWN_HOST, CLIENT, DISCOVER };
  enum PathType {
    UNKNOWN_PATH,
    EMPTY_PATH,
    SERVICE,
    ACTOR,
    ENVIRONMENT,
    DATALOG,
    PREHOOK,
    LIFECYCLE,
    ACTSERVICE,
    DATASTORE,
    MODELREGISTRY,
    DIRECTORY
  };

  std::string original_endpoint;  // The original endpoint and underlying base for 'std::string_view'

  SchemeType scheme;
  std::string_view address;  // Address if a GRPC scheme

  HostType host;
  PathType path;
  std::vector<PropertyView> query;

  EndpointData() : scheme(UNKNOWN_SCHEME), host(UNKNOWN_HOST), path(UNKNOWN_PATH) {}
  bool is_context_endpoint() const;
  void set_context_endpoint();
  void parse(std::string_view endpoint);
  std::string debug_string() const;
};

}  // namespace cogment
#endif
