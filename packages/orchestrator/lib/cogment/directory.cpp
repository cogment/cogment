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

#ifndef NDEBUG
  #define SPDLOG_ACTIVE_LEVEL SPDLOG_LEVEL_TRACE
#endif

#include "cogment/directory.h"
#include "cogment/utils.h"

#include "spdlog/spdlog.h"
#include "grpc++/grpc++.h"

namespace cogment {

namespace {

// Terminology
// URL: scheme://host/path?prop_name=prop_value&prop_name=prop_value
// GRPC scheme: grpc://address  ("address" is given whole to gRPC)

// Schemes
constexpr std::string_view GRPC("grpc");
constexpr std::string_view COGMENT("cogment");

// Hosts
constexpr std::string_view CLIENT("client");
constexpr std::string_view DISCOVER("discover");

// Paths
constexpr std::string_view SERVICE("service");
constexpr std::string_view ACTOR("actor");
constexpr std::string_view ENVIRONMENT("environment");
constexpr std::string_view DATALOG("datalog");
constexpr std::string_view PREHOOK("prehook");
constexpr std::string_view LIFECYCLE("lifecycle");
constexpr std::string_view ACTSERVICE("actservice");

// Property names
constexpr std::string_view ID_PROPERTY_NAME("id");
constexpr std::string_view AUTHENTICATION_TOKEN_PROPERTY_NAME("__authentication-token");

// Separator characters
const char SCHEME_SEPARATOR1 = ':';
const char SCHEME_SEPARATOR2 = '/';
const char SCHEME_SEPARATOR3 = '/';
const char PATH_SEPARATOR = '/';
const char QUERY_SEPARATOR = '?';
const char PROPERTY_SEPARATOR = '&';
const char VALUE_SEPARATOR = '=';
const char PORT_SEPARATOR = ':';

// gRPC directory metadata names
const std::string AUTHENTICATION_TOKEN_METADATA_NAME("authentication-token");

// Returns true if the full string starts with the given substring
bool starts_with(std::string_view full, const std::string_view& sub) {
  if (full.size() >= sub.size()) {
    for (size_t index = 0; index < sub.size(); index++) {
      if (sub[index] != full[index]) {
        return false;
      }
    }
  }
  else {
    return false;
  }

  return true;
}

// If the url starts with the sub string followed by a separator (or end of url),
// update url by removing the sub+separator and return true.
bool update_if_starts(std::string_view* url, const std::string_view& sub, char separator) {
  if (starts_with(*url, sub)) {
    if (url->size() == sub.size()) {
      url->remove_prefix(sub.size());
      return true;
    }
    else if ((*url)[sub.size()] == separator) {
      url->remove_prefix(sub.size() + 1);
      return true;
    }
  }

  return false;
}

// Very specific "parser" for our use case. "Fake" because it is not a generic URL parser.
void fake_url_parser(const std::string_view in_url, EndpointData* data) {
  // Assuming cleared data

  data->original_endpoint = std::string(in_url);

  const std::string low_url = to_lower_case(data->original_endpoint);
  std::string_view url = low_url;

  auto res = std::find_if(url.begin(), url.end(), [](unsigned char val) -> char {
    return std::isspace(val);
  });
  if (res != url.end()) {
    throw MakeException("Malformed URL (cannot contain spaces) [{}]", in_url);
  }

  // Scheme
  if (url.empty()) {
    data->scheme = EndpointData::SchemeType::UNKNOWN_SCHEME;
    return;
  }
  else if (update_if_starts(&url, GRPC, SCHEME_SEPARATOR1)) {
    if (url.size() < 2 || url[0] != SCHEME_SEPARATOR2 || url[1] != SCHEME_SEPARATOR3) {
      throw MakeException("Malformed URL (scheme must be followed by '://') [{}]", in_url);
    }

    data->scheme = EndpointData::SchemeType::GRPC;
    data->address = data->original_endpoint;
    data->address.remove_prefix(GRPC.size() + 3);
    return;
  }
  else if (update_if_starts(&url, COGMENT, SCHEME_SEPARATOR1)) {
    if (url.size() < 2 || url[0] != SCHEME_SEPARATOR2 || url[1] != SCHEME_SEPARATOR3) {
      throw MakeException("Malformed URL (scheme must be followed by '://') [{}]", in_url);
    }
    url.remove_prefix(2);

    data->scheme = EndpointData::SchemeType::COGMENT;
  }
  else {
    data->scheme = EndpointData::SchemeType::UNKNOWN_SCHEME;
    return;
  }

  // Host
  bool empty_path_with_query = false;
  if (url.empty()) {
    data->host = EndpointData::HostType::UNKNOWN_HOST;
    return;
  }
  else if (update_if_starts(&url, CLIENT, PATH_SEPARATOR)) {
    data->host = EndpointData::HostType::CLIENT;
  }
  else if (update_if_starts(&url, DISCOVER, PATH_SEPARATOR)) {
    data->host = EndpointData::HostType::DISCOVER;
  }
  else if (update_if_starts(&url, DISCOVER, QUERY_SEPARATOR)) {
    data->host = EndpointData::HostType::DISCOVER;
    empty_path_with_query = true;
  }
  else {
    data->host = EndpointData::HostType::UNKNOWN_HOST;
    return;
  }

  // Path
  if (url.empty()) {
    data->path = EndpointData::PathType::EMPTY_PATH;
    return;
  }
  else if (empty_path_with_query) {
    data->path = EndpointData::PathType::EMPTY_PATH;
  }
  else if (url[0] == QUERY_SEPARATOR) {
    url.remove_prefix(1);
    data->path = EndpointData::PathType::EMPTY_PATH;
  }
  else if (update_if_starts(&url, SERVICE, QUERY_SEPARATOR)) {
    data->path = EndpointData::PathType::SERVICE;
  }
  else if (update_if_starts(&url, ACTOR, QUERY_SEPARATOR)) {
    data->path = EndpointData::PathType::ACTOR;
  }
  else if (update_if_starts(&url, ENVIRONMENT, QUERY_SEPARATOR)) {
    data->path = EndpointData::PathType::ENVIRONMENT;
  }
  else if (update_if_starts(&url, DATALOG, QUERY_SEPARATOR)) {
    data->path = EndpointData::PathType::DATALOG;
  }
  else if (update_if_starts(&url, PREHOOK, QUERY_SEPARATOR)) {
    data->path = EndpointData::PathType::PREHOOK;
  }
  else if (update_if_starts(&url, LIFECYCLE, QUERY_SEPARATOR)) {
    data->path = EndpointData::PathType::LIFECYCLE;
  }
  else if (update_if_starts(&url, ACTSERVICE, QUERY_SEPARATOR)) {
    data->path = EndpointData::PathType::ACTSERVICE;
  }
  else {
    data->path = EndpointData::PathType::UNKNOWN_PATH;
    return;
  }

  // Properties
  const size_t query_start = url.data() - low_url.data();
  url = data->original_endpoint;
  url.remove_prefix(query_start);

  while (!url.empty()) {
    size_t equ = url.find(VALUE_SEPARATOR);
    if (equ != url.npos) {
      auto prop = url.substr(0, equ);
      url.remove_prefix(equ + 1);

      std::string_view val;
      size_t sep = url.find(PROPERTY_SEPARATOR);
      if (sep != url.npos) {
        val = url.substr(0, sep);
        url.remove_prefix(sep + 1);
      }
      else {
        val = url;
        url.remove_prefix(url.size());
      }

      data->query.emplace_back(prop, val);
    }
    else {
      std::string_view prop;
      size_t sep = url.find(PROPERTY_SEPARATOR);
      if (sep != url.npos) {
        prop = url.substr(0, sep);
        url.remove_prefix(sep + 1);
      }
      else {
        prop = url;
        url.remove_prefix(url.size());
      }

      data->query.emplace_back(prop, std::string_view());
    }
  }
}

uint64_t convert_to_id(std::string_view value) {
  if (value.empty()) {
    throw MakeException("Empty");
  }

  for (unsigned char digit : value) {
    if (!std::isdigit(digit)) {
      throw MakeException("Invalid character [{}]", digit);
    }
  }

  try {
    return std::stoull(std::string(value));
  }
  catch (const std::out_of_range&) {
    throw MakeException("Invalid uint64 (too large)");
  }
}

void test_reply(const cogmentAPI::FullServiceData& reply, const EndpointData& request_data) {
  switch (request_data.path) {
  case EndpointData::SERVICE:
    for (auto& prop : request_data.query) {
      if (prop.first == ID_PROPERTY_NAME) {
        uint64_t id = convert_to_id(prop.second);
        if (reply.service_id() != id) {
          throw MakeException("Directory reply service ID [{}] does not match request [{}]", reply.service_id(), id);
        }
      }
    }
    break;

  case EndpointData::PathType::ACTOR:
    if (reply.details().type() != cogmentAPI::ACTOR_SERVICE) {
      throw MakeException("Directory reply service type [{}] does not match request (actor)", reply.details().type());
    }
    break;

  case EndpointData::PathType::ENVIRONMENT:
    if (reply.details().type() != cogmentAPI::ENVIRONMENT_SERVICE) {
      throw MakeException("Directory reply service type [{}] does not match request (environment)",
                          reply.details().type());
    }
    break;

  case EndpointData::PathType::DATALOG:
    if (reply.details().type() != cogmentAPI::DATALOG_SERVICE) {
      throw MakeException("Directory reply service type [{}] does not match request (datalog)", reply.details().type());
    }
    break;

  case EndpointData::PathType::PREHOOK:
    if (reply.details().type() != cogmentAPI::PRE_HOOK_SERVICE) {
      throw MakeException("Directory reply service type [{}] does not match request (pre-trial hook)",
                          reply.details().type());
    }
    break;

  case EndpointData::PathType::LIFECYCLE:
  case EndpointData::PathType::ACTSERVICE:
  case EndpointData::PathType::EMPTY_PATH:
  case EndpointData::PathType::UNKNOWN_PATH:
    throw MakeException("Internal error: path type [{}] should not have been requested from directory",
                        request_data.path);
    break;
  }
}

}  // namespace

std::string EndpointData::debug_string() const {
  std::string result;

  result += MakeString("original endpoint [{}] address [{}]", original_endpoint, address);
  result += MakeString("  scheme [{}]", scheme);
  result += MakeString("  host [{}]", host);
  result += MakeString("  path [{}]", path);

  for (auto& prop : query) {
    result += MakeString("  prop [{}]=[{}]", prop.first, prop.second);
  }

  return result;
}

void parse_endpoint(const std::string& endpoint, EndpointData* data) {
  auto url = trim(endpoint);
  fake_url_parser(url, data);
  SPDLOG_TRACE("Endpoint Parser: In - [{}]   Out - {}", endpoint, data->debug_string());
}

void Directory::add_stub(const StubEntryType& stub) { m_stubs.emplace_back(stub); }

bool Directory::is_context_endpoint(const EndpointData& data) const {
  return (data.scheme == EndpointData::SchemeType::COGMENT && data.host == EndpointData::HostType::DISCOVER &&
          data.path == EndpointData::PathType::EMPTY_PATH);
}

std::string Directory::get_address(std::string_view name, const EndpointData& data) const {
  SPDLOG_DEBUG("Get address from directory with: [{}]", data.debug_string());

  switch (data.scheme) {
  case EndpointData::SchemeType::GRPC: {
    return std::string(data.address);
    break;
  }

  case EndpointData::SchemeType::COGMENT: {
    break;  // Continue processing
  }

  case EndpointData::SchemeType::UNKNOWN_SCHEME: {
    if (data.original_endpoint.empty()) {
      throw MakeException("Invalid empty endpoint");
    }
    else {
      throw MakeException("Unknown endpoint scheme (must be 'grpc' or 'cogment'): [{}]", data.original_endpoint);
    }
    break;
  }
  }

  // "cogment://"
  switch (data.host) {
  case EndpointData::HostType::CLIENT: {
    if (data.path == EndpointData::PathType::EMPTY_PATH) {
      return std::string(CLIENT_ACTOR_ADDRESS);
    }
    else {
      throw MakeException("Invalid client endpoint (should not have a path): [{}]", data.original_endpoint);
    }
    break;
  }

  case EndpointData::HostType::DISCOVER: {
    break;  // Continue processing
  }

  case EndpointData::HostType::UNKNOWN_HOST: {
    throw MakeException("Unknown endpoint host (must be 'discover' or 'client'): [{}]", data.original_endpoint);
    break;
  }
  }

  // "cogment://discover"
  if (m_stubs.empty()) {
    throw MakeException("No directory service defined to inquire endpoint: [{}]", data.original_endpoint);
  }

  cogmentAPI::InquireRequest req;
  grpc::ClientContext context;

  uint64_t id = 0;
  cogmentAPI::ServiceType service_requested = cogmentAPI::UNKNOWN_SERVICE;
  switch (data.path) {
  case EndpointData::PathType::EMPTY_PATH: {
    // The caller should have added the service and properties from the context
    throw MakeException("Missing endpoint path ('service', 'actor', 'environment', etc): [{}]", data.original_endpoint);
    break;
  }

  case EndpointData::PathType::SERVICE: {
    bool id_found = false;
    for (auto& prop : data.query) {
      const auto& name = prop.first;
      const auto& value = prop.second;

      if (name == ID_PROPERTY_NAME) {
        id_found = true;
        try {
          id = convert_to_id(value);
        }
        catch (const CogmentError& exc) {
          throw MakeException("Invalid service ID [{}] in endpoint [{}]", exc.what(), data.original_endpoint);
        }
      }
      else if (name == AUTHENTICATION_TOKEN_PROPERTY_NAME) {
        continue;
      }
      else {
        throw MakeException(
            "Invalid endpoint service path property [{}] (only 'id' and '__authentication-token' allowed): [{}]", name,
            data.original_endpoint);
      }
    }

    if (!id_found) {
      throw MakeException("Endpoint Service path must have an `id` property: [{}]", data.original_endpoint);
    }

    break;
  }

  case EndpointData::PathType::ACTOR: {
    service_requested = cogmentAPI::ACTOR_SERVICE;
    break;
  }

  case EndpointData::PathType::ENVIRONMENT: {
    service_requested = cogmentAPI::ENVIRONMENT_SERVICE;
    break;
  }

  case EndpointData::PathType::DATALOG: {
    service_requested = cogmentAPI::DATALOG_SERVICE;
    break;
  }

  case EndpointData::PathType::PREHOOK: {
    service_requested = cogmentAPI::PRE_HOOK_SERVICE;
    break;
  }

  case EndpointData::PathType::LIFECYCLE: {
    throw MakeException("The orchestrator does not connect to 'lifecycle' services: [{}]", data.original_endpoint);
    break;
  }

  case EndpointData::PathType::ACTSERVICE: {
    throw MakeException("The orchestrator does not connect to 'actservice' services: [{}]", data.original_endpoint);
    break;
  }

  case EndpointData::PathType::UNKNOWN_PATH: {
    throw MakeException("Unknown endpoint path ('service', 'actor', 'environment', etc): [{}]", data.original_endpoint);
    break;
  }
  }

  if (service_requested != cogmentAPI::UNKNOWN_SERVICE) {
    req.mutable_details()->set_type(service_requested);

    auto& req_prop = *req.mutable_details()->mutable_properties();
    for (auto& prop : data.query) {
      if (prop.first != AUTHENTICATION_TOKEN_PROPERTY_NAME) {
        req_prop[std::string(prop.first)] = std::string(prop.second);
      }
      else {
        context.AddMetadata(AUTHENTICATION_TOKEN_METADATA_NAME, std::string(prop.second));
      }
    }
  }
  else {
    req.set_service_id(id);

    for (auto& prop : data.query) {
      if (prop.first == AUTHENTICATION_TOKEN_PROPERTY_NAME) {
        context.AddMetadata(AUTHENTICATION_TOKEN_METADATA_NAME, std::string(prop.second));
      }
    }
  }

  // ----------------------------------------------------------------------------------------

  cogmentAPI::InquireReply reply;
  try {
    // Limited to one directory for now
    auto stream = m_stubs[0]->get_stub().Inquire(&context, req);

    // We use the first response of the stream
    if (!stream->Read(&reply)) {
      throw MakeException("No response for [{}]", data.original_endpoint);
    }

    context.TryCancel();
  }
  catch (const std::exception& exc) {
    throw MakeException("Failed to inquire from directory [{}]: {}", context.peer(), exc.what());
  }

  if (!reply.data().has_endpoint()) {
    throw MakeException("Could not find service from directory [{}]: [{}]", context.peer(), data.original_endpoint);
  }

  test_reply(reply.data(), data);

  auto& endpoint = reply.data().endpoint();
  std::string address = endpoint.hostname();

  switch (endpoint.protocol()) {
  case cogmentAPI::ServiceEndpoint::GRPC: {
    if (endpoint.port() == 0) {
      throw MakeException("Invalid address port (0) from directory [{}]: [{}]", context.peer(), address);
    }

    address += PORT_SEPARATOR + std::to_string(endpoint.port());
    break;
  }

  case cogmentAPI::ServiceEndpoint::GRPC_SSL: {
    throw MakeException("Unhandled endpoint ssl scheme from directory [{}]: [{}]", context.peer(),
                        data.original_endpoint);
    break;
  }

  case cogmentAPI::ServiceEndpoint::COGMENT: {
    if (address != CLIENT_ACTOR_ADDRESS) {
      throw MakeException("Unhandled endpoint from directory [{}]: {}://{}", context.peer(), COGMENT, address);
    }
    break;
  }

  case cogmentAPI::ServiceEndpoint::UNKNOWN:
  default: {
    throw MakeException("Unknown endpoint scheme from directory [{}]: [{}] [{}]", context.peer(),
                        data.original_endpoint, endpoint.protocol());
    break;
  }
  }

  spdlog::debug("Directory result for [{}] [{}] from [{}]: [{}]", name, data.original_endpoint, context.peer(),
                address);

  // Little hack to differentiate a host named client and a client endpoint:
  //   "client:XXX" -> a host named "client" (i.e. with a port number XXX)
  //   "client" -> a client actor endpoint
  return address;
}

}  // namespace cogment