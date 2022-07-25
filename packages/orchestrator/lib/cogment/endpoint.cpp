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

#include "cogment/endpoint.h"

#include "spdlog/spdlog.h"

namespace cogment {

namespace {

// Terminology
// URL: scheme://host/path?prop_name=prop_value&prop_name=prop_value
// GRPC scheme: grpc://address  ("address" is given whole to gRPC)

// Schemes
constexpr std::string_view GRPC_SCHEME("grpc");
constexpr std::string_view COGMENT_SCHEME("cogment");

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
constexpr std::string_view DATASTORE("datastore");
constexpr std::string_view MODELREGISTRY("modelregistry");

// Separator characters
constexpr char SCHEME_SEPARATOR1 = ':';
constexpr char SCHEME_SEPARATOR2 = '/';
constexpr char SCHEME_SEPARATOR3 = '/';
constexpr char PATH_SEPARATOR = '/';
constexpr char QUERY_SEPARATOR = '?';
constexpr char PROPERTY_SEPARATOR = '&';
constexpr char VALUE_SEPARATOR = '=';

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
  else if (update_if_starts(&url, GRPC_SCHEME, SCHEME_SEPARATOR1)) {
    if (url.size() < 2 || url[0] != SCHEME_SEPARATOR2 || url[1] != SCHEME_SEPARATOR3) {
      throw MakeException("Malformed URL (scheme must be followed by '://') [{}]", in_url);
    }

    data->scheme = EndpointData::SchemeType::GRPC;
    data->address = data->original_endpoint;
    data->address.remove_prefix(GRPC_SCHEME.size() + 3);
    return;
  }
  else if (update_if_starts(&url, COGMENT_SCHEME, SCHEME_SEPARATOR1)) {
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
  else if (update_if_starts(&url, DATASTORE, QUERY_SEPARATOR)) {
    data->path = EndpointData::PathType::DATASTORE;
  }
  else if (update_if_starts(&url, MODELREGISTRY, QUERY_SEPARATOR)) {
    data->path = EndpointData::PathType::MODELREGISTRY;
  }
  else {
    data->path = EndpointData::PathType::UNKNOWN_PATH;
    return;
  }

  // Properties
  const size_t query_start = url.data() - low_url.data();
  url = data->original_endpoint;
  url.remove_prefix(query_start);
  data->query = parse_properties(url, PROPERTY_SEPARATOR, VALUE_SEPARATOR);
}

}  // namespace

bool EndpointData::is_context_endpoint() const {
  return (scheme == SchemeType::COGMENT && host == HostType::DISCOVER && path == PathType::EMPTY_PATH);
}

void EndpointData::set_context_endpoint() {
  scheme = SchemeType::COGMENT;
  host = HostType::DISCOVER;
  path = PathType::EMPTY_PATH;
}

std::string EndpointData::debug_string() const {
  std::string result;

  result += MakeString("original endpoint [{}] address [{}]", original_endpoint, address);
  result += MakeString("  scheme [{}]", scheme);
  result += MakeString("  host [{}]", host);
  result += MakeString("  path [{}]", path);

  for (auto& prop : query) {
    result += MakeString("  prop [{}]=[{}]", prop.name, prop.value);
  }

  return result;
}

void EndpointData::parse(std::string_view endpoint) {
  auto url = trim(endpoint);
  fake_url_parser(url, this);
  SPDLOG_TRACE("Endpoint Parser: In - [{}]   Out - {}", endpoint, debug_string());
}

}  // namespace cogment
