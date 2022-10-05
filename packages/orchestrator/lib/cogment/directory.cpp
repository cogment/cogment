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

const char PORT_SEPARATOR = ':';

// gRPC directory metadata names
const std::string AUTHENTICATION_TOKEN_METADATA_NAME("authentication-token");

uint64_t convert_to_uint64(std::string_view value) {
  if (value.empty()) {
    throw MakeException("Empty string for uint");
  }

  for (unsigned char digit : value) {
    if (!std::isdigit(digit)) {
      throw MakeException("Invalid uint character [{}]", digit);
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
      if (prop.name == SERVICE_ID_PROPERTY_NAME) {
        uint64_t id = convert_to_uint64(prop.value);
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
  case EndpointData::PathType::DATASTORE:
  case EndpointData::PathType::MODELREGISTRY:
  case EndpointData::PathType::EMPTY_PATH:
  case EndpointData::PathType::UNKNOWN_PATH:
    throw MakeException("Internal error: path type [{}] should not have been requested from directory",
                        request_data.path);
    break;
  }
}

}  // namespace

// Get gRPC service type from endpoint data
// Provide service ID if the request is for a particular service ID (service type is unknown)
cogmentAPI::ServiceType Directory::get_service_type(const EndpointData& data, uint64_t* id) const {
  cogmentAPI::ServiceType service = cogmentAPI::UNKNOWN_SERVICE;

  switch (data.path) {
  case EndpointData::PathType::EMPTY_PATH: {
    // The caller should have added the service and properties from the context
    throw MakeException("Missing endpoint path ('service', 'actor', 'environment', etc): [{}]", data.original_endpoint);
    break;
  }

  case EndpointData::PathType::SERVICE: {
    bool id_found = false;
    for (auto& prop : data.query) {
      if (prop.name == SERVICE_ID_PROPERTY_NAME) {
        id_found = true;
        try {
          *id = convert_to_uint64(prop.value);
        }
        catch (const CogmentError& exc) {
          throw MakeException("Invalid service ID [{}] in endpoint [{}]", exc.what(), data.original_endpoint);
        }
      }
      else if (prop.name == AUTHENTICATION_TOKEN_PROPERTY_NAME) {
        continue;
      }
      else {
        throw MakeException("Invalid endpoint service path property [{}] (only [{}] and [{}] allowed): [{}]", prop.name,
                            SERVICE_ID_PROPERTY_NAME, AUTHENTICATION_TOKEN_PROPERTY_NAME, data.original_endpoint);
      }
    }

    if (!id_found) {
      throw MakeException("Endpoint Service path must have an `id` property: [{}]", data.original_endpoint);
    }

    break;
  }

  case EndpointData::PathType::ACTOR: {
    service = cogmentAPI::ACTOR_SERVICE;
    break;
  }

  case EndpointData::PathType::ENVIRONMENT: {
    service = cogmentAPI::ENVIRONMENT_SERVICE;
    break;
  }

  case EndpointData::PathType::DATALOG: {
    service = cogmentAPI::DATALOG_SERVICE;
    break;
  }

  case EndpointData::PathType::PREHOOK: {
    service = cogmentAPI::PRE_HOOK_SERVICE;
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

  case EndpointData::PathType::DATASTORE: {
    throw MakeException("The orchestrator does not connect to 'datastore' services: [{}]", data.original_endpoint);
    break;
  }

  case EndpointData::PathType::MODELREGISTRY: {
    throw MakeException("The orchestrator does not connect to 'modelregistry' services: [{}]", data.original_endpoint);
    break;
  }

  case EndpointData::PathType::UNKNOWN_PATH: {
    throw MakeException("Unknown endpoint path ('service', 'actor', 'environment', etc): [{}]", data.original_endpoint);
    break;
  }
  }

  return service;
}

// Little hack to differentiate a hostname and a special endpoint of the same name:
//   "client:XXX" -> a host named "client" (i.e. with a port number XXX). Equivalent to "grpc://client:XXX" endpoint.
//   "client" -> a special endpoint to indicate a client actor. Equivalent to "cogment://client" endpoint.
Directory::InquiredAddress Directory::inquire_address(std::string_view service_name, const EndpointData& data) const {
  SPDLOG_DEBUG("Get address for [{}] from directory with [{}]", service_name, data.debug_string());

  switch (data.scheme) {
  case EndpointData::SchemeType::GRPC: {
    return {std::string(data.address), false};
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
      return {std::string(CLIENT_ACTOR_ADDRESS), false};
    }
    else {
      throw MakeException("Invalid client actor special endpoint (should not have a path): [{}]",
                          data.original_endpoint);
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

  // endpoint == "cogment://discover" + ...
  if (m_stubs.empty()) {
    throw MakeException("No directory service defined to inquire discovery endpoint: [{}]", data.original_endpoint);
  }

  uint64_t id = 0;
  const auto service_requested = get_service_type(data, &id);

  bool endpoint_auth_token = false;
  cogmentAPI::InquireRequest request;
  grpc::ClientContext context;
  if (service_requested != cogmentAPI::UNKNOWN_SERVICE) {
    request.mutable_details()->set_type(service_requested);

    auto request_properties = request.mutable_details()->mutable_properties();
    for (auto& prop : data.query) {
      if (prop.name != AUTHENTICATION_TOKEN_PROPERTY_NAME) {
        request_properties->insert({std::string(prop.name), std::string(prop.value)});
      }
      else {
        context.AddMetadata(AUTHENTICATION_TOKEN_METADATA_NAME, std::string(prop.value));
        endpoint_auth_token = true;
      }
    }
  }
  else {
    request.set_service_id(id);

    for (auto& prop : data.query) {
      if (prop.name == AUTHENTICATION_TOKEN_PROPERTY_NAME) {
        context.AddMetadata(AUTHENTICATION_TOKEN_METADATA_NAME, std::string(prop.value));
        endpoint_auth_token = true;
      }
    }
  }

  if (!endpoint_auth_token && !m_auth_token.empty()) {
    context.AddMetadata(AUTHENTICATION_TOKEN_METADATA_NAME, m_auth_token);
  }

  auto address = get_address_from_directory(context, request, data);
  spdlog::debug("Directory result for [{}] [{}] from [{}]: [{}][{}]", service_name, data.original_endpoint,
                context.peer(), address.address, address.ssl);

  return address;
}

// TODO: Return all addresses found in all directories. But right now, this would only be useful for ssl use matching.
Directory::InquiredAddress Directory::get_address_from_directory(grpc::ClientContext& context,
                                                                 cogmentAPI::InquireRequest& request,
                                                                 const EndpointData& endpoint_data) const {
  cogmentAPI::InquireReply reply;
  try {
    // Limited to one directory for now
    auto stream = m_stubs[0]->get_stub().Inquire(&context, request);

    // We use the first response of the stream
    if (!stream->Read(&reply)) {
      auto status = stream->Finish();
      if (status.ok()) {
        throw MakeException("No service found in directory for [{}]", endpoint_data.original_endpoint);
      }
      else {
        throw MakeException("Failed directory inquiry for [{}]: [{}]", endpoint_data.original_endpoint,
                            status.error_message());
      }
    }

    context.TryCancel();
  }
  catch (const std::exception& exc) {
    throw MakeException("Failed to inquire from directory [{}]: [{}]", context.peer(), exc.what());
  }

  if (!reply.data().has_endpoint()) {
    throw MakeException("Invalid (empty address) inquiry response from directory [{}] for [{}]", context.peer(),
                        endpoint_data.original_endpoint);
  }

  if (spdlog::default_logger()->level() <= spdlog::level::debug) {
    test_reply(reply.data(), endpoint_data);
  }

  InquiredAddress result;
  auto& endpoint = reply.data().endpoint();
  result.address = endpoint.host();

  switch (endpoint.protocol()) {
  case cogmentAPI::ServiceEndpoint::GRPC: {
    if (endpoint.port() == 0) {
      throw MakeException("Invalid address port (0) from directory [{}] for [{}]", context.peer(),
                          endpoint_data.original_endpoint);
    }

    result.address += PORT_SEPARATOR + std::to_string(endpoint.port());
    result.ssl = false;
    break;
  }

  case cogmentAPI::ServiceEndpoint::GRPC_SSL: {
    if (endpoint.port() == 0) {
      throw MakeException("Invalid address port (0) from directory [{}] for [{}]", context.peer(),
                          endpoint_data.original_endpoint);
    }

    result.address += PORT_SEPARATOR + std::to_string(endpoint.port());
    result.ssl = true;
    break;
  }

  case cogmentAPI::ServiceEndpoint::COGMENT: {
    if (result.address != CLIENT_ACTOR_ADDRESS) {
      throw MakeException("Unhandled endpoint from directory [{}]: [cogment://{}]", context.peer(), result.address);
    }
    result.ssl = false;
    break;
  }

  case cogmentAPI::ServiceEndpoint::UNKNOWN:
  default: {
    throw MakeException("Unknown endpoint scheme from directory [{}] for [{}]: [{}]", context.peer(),
                        endpoint_data.original_endpoint, endpoint.protocol());
    break;
  }
  }

  return result;
}

Directory::RegisteredService Directory::register_host(std::string_view host, uint16_t port, bool ssl,
                                                      Directory::ServiceType type,
                                                      const std::vector<PropertyView>& properties) {
  if (m_stubs.empty()) {
    throw MakeException("No directory service defined to register service");
  }

  grpc::ClientContext context;
  if (!m_auth_token.empty()) {
    context.AddMetadata(AUTHENTICATION_TOKEN_METADATA_NAME, m_auth_token);
  }

  cogmentAPI::RegisterRequest request;
  if (ssl) {
    request.mutable_endpoint()->set_protocol(cogmentAPI::ServiceEndpoint_Protocol_GRPC_SSL);
  }
  else {
    request.mutable_endpoint()->set_protocol(cogmentAPI::ServiceEndpoint_Protocol_GRPC);
  }
  request.mutable_endpoint()->set_host(std::string(host));
  request.mutable_endpoint()->set_port(port);

  request.mutable_details()->set_type(static_cast<cogmentAPI::ServiceType>(type));
  auto request_properties = request.mutable_details()->mutable_properties();
  for (auto& prop : properties) {
    request_properties->insert({std::string(prop.name), std::string(prop.value)});
  }

  cogmentAPI::RegisterReply reply;
  try {
    // Limited to one directory for now
    auto stream = m_stubs[0]->get_stub().Register(&context);

    if (!stream->Write(request)) {
      throw MakeException("Failed to write to directory to register");
    }

    // There should be only one response of the stream (we made only one request)
    if (!stream->Read(&reply)) {
      throw MakeException("No response to directory register request");
    }

    if (stream->WritesDone()) {
      auto status = stream->Finish();
      if (!status.ok()) {
        spdlog::warn("Failed to finish directory registration stream [{}]", status.error_message());
      }
    }
    else {
      spdlog::warn("Failed to close writing to directory registration stream");
    }
  }
  catch (const std::exception& exc) {
    throw MakeException("Directory register failure: [{}]", exc.what());
  }

  if (reply.status() != cogmentAPI::RegisterReply_Status_OK) {
    throw MakeException("Directory registration failed for host [{}]: [{}] [{}]", host, reply.status(),
                        reply.error_msg());
  }

  return {reply.service_id(), reply.secret()};
}

// We don't want exceptions when deregistering, since it is not critical and will be used in destructors
void Directory::deregister_service(const RegisteredService& record) {
  if (m_stubs.empty()) {
    spdlog::error("No directory service defined to deregister service");
    return;
  }

  grpc::ClientContext context;
  if (!m_auth_token.empty()) {
    context.AddMetadata(AUTHENTICATION_TOKEN_METADATA_NAME, m_auth_token);
  }

  cogmentAPI::DeregisterRequest request;
  request.set_service_id(record.service_id);
  request.set_secret(record.secret);

  cogmentAPI::DeregisterReply reply;
  try {
    // Limited to one directory for now
    auto stream = m_stubs[0]->get_stub().Deregister(&context);

    if (!stream->Write(request)) {
      spdlog::error("Failed to write to directory to deregister");
      return;
    }

    // There should be only one response of the stream (we made only one request)
    if (!stream->Read(&reply)) {
      spdlog::error("No response to directory deregister request");
      return;
    }

    if (stream->WritesDone()) {
      auto status = stream->Finish();
      if (!status.ok()) {
        spdlog::warn("Failed to finish directory deregistration stream [{}]", status.error_message());
      }
    }
    else {
      spdlog::warn("Failed to close writing to directory deregistration stream");
    }
  }
  catch (const std::exception& exc) {
    spdlog::error("Direcotry deregister Failure: [{}]", exc.what());
    return;
  }

  if (reply.status() != cogmentAPI::DeregisterReply_Status_OK) {
    spdlog::error("Directory deregistration failed for service id [{}]: [{}] [{}]", record.service_id, reply.status(),
                  reply.error_msg());
  }
}

}  // namespace cogment