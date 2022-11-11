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

#include "callback_spdlog_sink.h"

#include "cogment/config_file.h"
#include "cogment/orchestrator.h"
#include "cogment/utils.h"
#include "cogment/versions.h"
#include "cogment/services/actor_service.h"
#include "cogment/services/trial_lifecycle_service.h"

#include "prometheus/exposer.h"
#include "prometheus/registry.h"

#include "grpc++/grpc++.h"
#include "spdlog/spdlog.h"
#include "spdlog/sinks/stdout_color_sinks.h"

#include <fstream>
#include <thread>

#ifndef _WIN32
  #include <signal.h>
  #include <execinfo.h>  // backtrace
  #include <unistd.h>    // STDERR_FILENO

namespace {

void (*previous_sigsegv_handler)(int) = SIG_DFL;

void sigsegv_handler(int sig) {
  static constexpr size_t BACKTRACE_SIZE = 100;

  void* buffer[BACKTRACE_SIZE];
  int nptrs = backtrace(buffer, BACKTRACE_SIZE);

  // add2line can be used to get more info from output
  backtrace_symbols_fd(buffer, nptrs, STDERR_FILENO);

  // Reraising the signal to the previous handler
  signal(SIGSEGV, previous_sigsegv_handler);
  raise(sig);
}

// Define a "constructor" function that will be executed when the dynamic libary is loaded.
static void on_dylib_load() __attribute__((constructor));

void on_dylib_load() {
  // Registering the custom SIGSEGV handler and keeping around the previous one
  previous_sigsegv_handler = signal(SIGSEGV, sigsegv_handler);
}

}  // namespace
#endif

namespace {
constexpr uint32_t DEFAULT_MAX_INACTIVITY = 30;

struct Options {
  uint16_t lifecycle_port;
  uint16_t actor_port;
  std::string default_params_file;
  std::vector<std::string> pre_trial_hooks;
  std::vector<std::string> directory_services;
  std::string directory_auth_token;
  uint32_t directory_auto_register;
  std::string directory_register_host;
  std::string directory_register_props;
  uint16_t prometheus_port;
  std::string private_key;
  std::string root_cert;
  std::string trust_chain;
  std::uint32_t gc_frequency;
  std::string log_level;
  void* logger_ctx;
  CogmentOrchestratorLogger logger;
  void* status_listener_ctx;
  CogmentOrchestratorStatusListener status_listener;

  std::string debug_string() const {
    std::string result;
    result += cogment::MakeString("lifecycle port [{}], actor port [{}]", lifecycle_port, actor_port);
    result += cogment::MakeString(", params file [{}]", default_params_file);
    result += ", hooks [";
    for (auto& hook : pre_trial_hooks) {
      result += hook;
    }
    result += "]";
    result += ", directories [";
    for (auto& dir : directory_services) {
      result += dir;
    }
    result += "]";
    result += cogment::MakeString(", directory auth token [{}]", directory_auth_token);
    result += cogment::MakeString(", directory auto register [{}]", directory_auto_register);
    result += cogment::MakeString(", directory register host [{}]", directory_register_host);
    result += cogment::MakeString(", directory register properties [{}]", directory_register_props);
    result += cogment::MakeString(", prometheus port [{}], private key [{}]", prometheus_port, private_key);
    result += cogment::MakeString(", root cert [{}], trust chain [{}]", root_cert, trust_chain);
    result += cogment::MakeString(", garbage collection frequency [{}]", gc_frequency);
    result += cogment::MakeString(", log level [{}]", log_level);
    return result;
  }
};

class ServedOrchestrator {
public:
  ServedOrchestrator(const Options* options);
  ~ServedOrchestrator();

  int WaitForTermination();
  void Shutdown();

private:
  int m_status;
  void* m_status_listnener_ctx;
  CogmentOrchestratorStatusListener m_status_listener;
  cogment::Orchestrator* m_orchestrator;
  cogment::ActorService* m_actor_service;
  cogment::TrialLifecycleService* m_trial_lifecycle_service;
  std::vector<std::unique_ptr<grpc::Server>> m_grpc_servers;
  std::unique_ptr<prometheus::Exposer> m_metrics_exposer;
  std::shared_ptr<prometheus::Registry> m_metrics_registry;
};

ServedOrchestrator::ServedOrchestrator(const Options* options) :
    m_status(COGMENT_ORCHESTRATOR_INIT_STATUS),
    m_status_listnener_ctx(options->status_listener_ctx),
    m_status_listener(options->status_listener),
    m_orchestrator(),
    m_actor_service(),
    m_trial_lifecycle_service(),
    m_grpc_servers(),
    m_metrics_exposer(),
    m_metrics_registry() {
  if (options->logger != nullptr) {
    auto sink = std::make_shared<cogment::CallbackSpdLogSink<std::mutex>>(options->logger_ctx, options->logger);
    auto logger = std::make_shared<spdlog::logger>("ext_callback", sink);
    spdlog::set_default_logger(logger);
  }
  else {
    spdlog::set_default_logger(spdlog::stdout_color_mt("console"));
  }

  const std::string log_level = cogment::to_lower_case(options->log_level);
  try {
    if (!log_level.empty()) {
      const auto level_setting = spdlog::level::from_str(log_level);

      // spdlog returns "off" if an unknown string is given!
      if (level_setting == spdlog::level::level_enum::off && log_level != "off") {
        throw cogment::MakeException("Unknown log level setting");
      }

      spdlog::set_level(level_setting);
    }
    else {
      spdlog::set_level(spdlog::level::info);
    }
  }
  catch (const std::exception& exc) {
    spdlog::set_level(spdlog::level::info);
    spdlog::warn("Failed to set log level to [{}], defaulting to 'info': [{}]", log_level, exc.what());
  }
  catch (...) {
    spdlog::set_level(spdlog::level::info);
    spdlog::warn("Failed to set log level to [{}], defaulting to 'info'", log_level);
  }

  if (m_status_listener != nullptr) {
    m_status_listener(m_status_listnener_ctx, m_status);
  }

  spdlog::debug("Orchestrator options: [{}]", options->debug_string());

  std::shared_ptr<grpc::ServerCredentials> server_creds;
  std::shared_ptr<grpc::ChannelCredentials> client_creds;
  const bool using_ssl = !(options->private_key.empty() && options->root_cert.empty() && options->trust_chain.empty());
  if (!using_ssl) {
    spdlog::info("Using unsecured communication");
    server_creds = grpc::InsecureServerCredentials();
    // We don't set client_creds because we use the presence of a pointer to check for SSL use.
    // ChannelCredentials offers no way to check if an encrypted communication is used.
  }
  else {
    spdlog::info("Using secured TLS communication");

    auto private_key_file = std::ifstream(options->private_key);
    if (!private_key_file.is_open() || !private_key_file.good()) {
      throw cogment::MakeException("Could not open private key file: [{}]", options->private_key);
    }
    std::stringstream private_key;
    private_key << private_key_file.rdbuf();

    auto root_cert_file = std::ifstream(options->root_cert);
    if (!root_cert_file.is_open() || !root_cert_file.good()) {
      throw cogment::MakeException("Could not open root certificate file: [{}]", options->root_cert);
    }
    std::stringstream root_cert;
    root_cert << root_cert_file.rdbuf();

    auto trust_chain_file = std::ifstream(options->trust_chain);
    if (!trust_chain_file.is_open() || !trust_chain_file.good()) {
      throw cogment::MakeException("Could not open certificate trust chain file: [{}]", options->trust_chain);
    }
    std::stringstream trust_chain;
    trust_chain << trust_chain_file.rdbuf();

    grpc::SslServerCredentialsOptions server_opt(GRPC_SSL_REQUEST_AND_REQUIRE_CLIENT_CERTIFICATE_AND_VERIFY);
    server_opt.pem_root_certs = root_cert.str();
    grpc::SslServerCredentialsOptions::PemKeyCertPair server_cert_pair;
    server_cert_pair.private_key = private_key.str();
    server_cert_pair.cert_chain = trust_chain.str();
    server_opt.pem_key_cert_pairs.emplace_back(server_cert_pair);
    server_creds = grpc::SslServerCredentials(server_opt);

    grpc::SslCredentialsOptions client_opt;
    client_opt.pem_root_certs = root_cert.str();
    client_opt.pem_private_key = private_key.str();
    client_opt.pem_cert_chain = trust_chain.str();
    client_creds = grpc::SslCredentials(client_opt);
  }

  // ******************* Endpoints *******************
  cogment::NetChecker checker;
  if (checker.is_tcp_port_used(options->lifecycle_port)) {
    throw cogment::MakeException("Lifecycle port [{}] is already in use", options->lifecycle_port);
  }
  if (checker.is_tcp_port_used(options->actor_port)) {
    throw cogment::MakeException("Actor port [{}] is already in use", options->actor_port);
  }
  auto lifecycle_endpoint = std::string("0.0.0.0:") + std::to_string(options->lifecycle_port);
  auto actor_endpoint = std::string("0.0.0.0:") + std::to_string(options->actor_port);

  // ******************* Monitoring *******************
  if (options->prometheus_port > 0) {
    if (checker.is_tcp_port_used(options->prometheus_port)) {
      throw cogment::MakeException("Prometheus port [{}] is already in use", options->prometheus_port);
    }
    auto prometheus_endpoint = std::string("0.0.0.0:") + std::to_string(options->prometheus_port);
    spdlog::info("Starting prometheus at [{}]", prometheus_endpoint);

    m_metrics_exposer = std::make_unique<prometheus::Exposer>(prometheus_endpoint);
    m_metrics_registry = std::make_shared<prometheus::Registry>();
    m_metrics_exposer->RegisterCollectable(m_metrics_registry);
  }
  else {
    spdlog::info("Prometheus disabled");
  }

  // ******************* Orchestrator *******************
  YAML::Node params_yaml;
  bool params_file_loaded = false;
  if (!options->default_params_file.empty()) {
    try {
      params_yaml = YAML::LoadFile(options->default_params_file);
      params_file_loaded = true;
    }
    catch (const std::exception& exc) {
      throw cogment::MakeException("Failed to load default parameters file [{}]: [{}]", options->default_params_file,
                                   exc.what());
    }
  }
  else {
    spdlog::debug("No default parameters.");
  }
  auto params = cogment::load_params(params_yaml);

  if (params_file_loaded) {
    if (!params_yaml[cogment::cfg_file::params_key] ||
        !params_yaml[cogment::cfg_file::params_key][cogment::cfg_file::p_max_inactivity_key]) {
      params.set_max_inactivity(DEFAULT_MAX_INACTIVITY);
    }
  }

  m_orchestrator =
      new cogment::Orchestrator(std::move(params), options->gc_frequency, client_creds, m_metrics_registry.get());

  // ******************* Networking *******************
  if (!options->directory_services.empty()) {
    for (auto& directory_service : options->directory_services) {
      m_orchestrator->add_directory(directory_service, options->directory_auth_token);
    }
  }
  else {
    spdlog::debug("No directory service defined");
  }

  int nb_prehooks = 0;
  for (auto& pre_trial_hook : options->pre_trial_hooks) {
    m_orchestrator->add_prehook(pre_trial_hook);
    nb_prehooks++;
  }
  spdlog::info("[{}] pre-trial hooks defined", nb_prehooks);

  m_actor_service = new cogment::ActorService(m_orchestrator);
  m_trial_lifecycle_service = new cogment::TrialLifecycleService(m_orchestrator);
  {
    grpc::ServerBuilder builder;
    builder.AddListeningPort(lifecycle_endpoint, server_creds);
    builder.RegisterService(m_trial_lifecycle_service);

    // If the lifecycle endpoint is the same as the ClientActorSP, then run them
    // off the same server, otherwise, start a second server.
    if (lifecycle_endpoint == actor_endpoint) {
      builder.RegisterService(m_actor_service);
    }
    else {
      grpc::ServerBuilder actor_builder;
      actor_builder.AddListeningPort(actor_endpoint, server_creds);
      actor_builder.RegisterService(m_actor_service);
      m_grpc_servers.emplace_back(actor_builder.BuildAndStart());
    }

    auto server = builder.BuildAndStart();
    spdlog::debug("Main server started");
    m_grpc_servers.emplace_back(std::move(server));
  }

  m_status = COGMENT_ORCHESTRATOR_READY_STATUS;
  if (m_status_listener) {
    m_status_listener(m_status_listnener_ctx, m_status);
  }

  const auto auto_option = options->directory_auto_register;
  if (auto_option == 1) {
    m_orchestrator->register_to_directory(options->directory_register_host, options->actor_port,
                                          options->lifecycle_port, options->directory_register_props);
  }
  else if (auto_option != 0) {
    spdlog::error("Unknown option for directory_auto_register [{}] (it must be 0 or 1)", auto_option);
  }

  spdlog::info("Server listening for lifecycle on [{}]", lifecycle_endpoint);
  spdlog::info("Server listening for Actors on [{}]", actor_endpoint);
}

ServedOrchestrator::~ServedOrchestrator() {
  if (m_status != COGMENT_ORCHESTRATOR_TERMINATED_STATUS) {
    this->Shutdown();
    this->WaitForTermination();
  }
  m_grpc_servers.clear();
  delete m_actor_service;
  delete m_trial_lifecycle_service;
  delete m_orchestrator;
}

int ServedOrchestrator::WaitForTermination() {
  int return_value = 0;
  try {
    std::vector<std::future<void>> futures;
    for (auto& server : m_grpc_servers) {
      grpc::Server* server_ptr = server.get();
      futures.emplace_back(m_orchestrator->thread_pool().push("awaiting gRPC server", [server_ptr]() {
        server_ptr->Wait();
      }));
    }
    for (auto& future : futures) {
      if (future.valid()) {
        future.wait();
      }
    }
  }
  catch (const std::exception& exc) {
    spdlog::error("Failure: [{}]", exc.what());
    return_value = -1;
  }
  catch (...) {
    spdlog::error("Failure");
    return_value = -1;
  }

  m_status = COGMENT_ORCHESTRATOR_TERMINATED_STATUS;
  if (m_status_listener) {
    m_status_listener(m_status_listnener_ctx, m_status);
  }

  return return_value;
}
void ServedOrchestrator::Shutdown() {
  for (auto& server : m_grpc_servers) {
    server->Shutdown();
  }
}
}  // namespace

// Defining the C symbols
extern "C" {
void* cogment_orchestrator_options_create() {
  Options* options = new Options();
  return options;
}
void cogment_orchestrator_options_destroy(void* c_options) {
  assert(c_options);
  Options* options = static_cast<Options*>(c_options);
  delete options;
}

void cogment_orchestrator_options_set_lifecycle_port(void* c_options, unsigned int port) {
  assert(c_options);
  Options* options = static_cast<Options*>(c_options);
  options->lifecycle_port = port;
}
void cogment_orchestrator_options_set_actor_port(void* c_options, unsigned int port) {
  assert(c_options);
  Options* options = static_cast<Options*>(c_options);
  options->actor_port = port;
}
void cogment_orchestrator_options_set_default_params_file(void* c_options, char* path) {
  assert(c_options);
  Options* options = static_cast<Options*>(c_options);
  options->default_params_file = path;
}
void cogment_orchestrator_options_add_pretrial_hook(void* c_options, char* pretrial_hook_endpoint) {
  assert(c_options);
  Options* options = static_cast<Options*>(c_options);
  options->pre_trial_hooks.push_back(pretrial_hook_endpoint);
}
void cogment_orchestrator_options_add_directory_service(void* c_options, char* directory_service_endpoint) {
  assert(c_options);
  Options* options = static_cast<Options*>(c_options);
  options->directory_services.push_back(directory_service_endpoint);
}
void cogment_orchestrator_options_set_directory_auth_token(void* c_options, char* token) {
  assert(c_options);
  Options* options = static_cast<Options*>(c_options);
  options->directory_auth_token = token;
}
void cogment_orchestrator_options_set_auto_registration(void* c_options, unsigned int auto_register) {
  assert(c_options);
  Options* options = static_cast<Options*>(c_options);
  options->directory_auto_register = auto_register;
}
void cogment_orchestrator_options_set_directory_register_host(void* c_options, char* host) {
  assert(c_options);
  Options* options = static_cast<Options*>(c_options);
  options->directory_register_host = host;
}
void cogment_orchestrator_options_set_directory_properties(void* c_options, char* properties) {
  assert(c_options);
  Options* options = static_cast<Options*>(c_options);
  options->directory_register_props = properties;
}
void cogment_orchestrator_options_set_prometheus_port(void* c_options, unsigned int port) {
  assert(c_options);
  Options* options = static_cast<Options*>(c_options);
  options->prometheus_port = port;
}
void cogment_orchestrator_options_set_private_key_file(void* c_options, char* path) {
  assert(c_options);
  Options* options = static_cast<Options*>(c_options);
  options->private_key = path;
}
void cogment_orchestrator_options_set_root_certificate_file(void* c_options, char* path) {
  assert(c_options);
  Options* options = static_cast<Options*>(c_options);
  options->root_cert = path;
}
void cogment_orchestrator_options_trust_chain_file(void* c_options, char* path) {
  assert(c_options);
  Options* options = static_cast<Options*>(c_options);
  options->trust_chain = path;
}
void cogment_orchestrator_options_garbage_collector_frequency(void* c_options, unsigned int frequency) {
  assert(c_options);
  Options* options = static_cast<Options*>(c_options);
  options->gc_frequency = frequency;
}
void cogment_orchestrator_options_set_logging(void* c_options, const char* level, void* ctx,
                                              CogmentOrchestratorLogger logger) {
  assert(c_options);
  Options* options = static_cast<Options*>(c_options);
  options->log_level = level;
  options->logger_ctx = ctx;
  options->logger = logger;
}

void cogment_orchestrator_options_set_status_listener(void* c_options, void* ctx,
                                                      CogmentOrchestratorStatusListener status_listener) {
  assert(c_options);
  Options* options = static_cast<Options*>(c_options);
  options->status_listener_ctx = ctx;
  options->status_listener = status_listener;
}

void* cogment_orchestrator_create_and_start(void* c_options) {
  assert(c_options);
  Options* options = static_cast<Options*>(c_options);
  try {
    return new ServedOrchestrator(options);
  }
  catch (const std::exception& exc) {
    spdlog::error("Failure: [{}]", exc.what());
    return nullptr;
  }
  catch (...) {
    spdlog::error("Failure");
    return nullptr;
  }
}

void cogment_orchestrator_destroy(void* c_orchestrator) {
  assert(c_orchestrator);
  ServedOrchestrator* orchestrator = static_cast<ServedOrchestrator*>(c_orchestrator);
  delete orchestrator;
}

int cogment_orchestrator_wait_for_termination(void* c_orchestrator) {
  assert(c_orchestrator);
  ServedOrchestrator* orchestrator = static_cast<ServedOrchestrator*>(c_orchestrator);
  return orchestrator->WaitForTermination();
}
void cogment_orchestrator_shutdown(void* c_orchestrator) {
  assert(c_orchestrator);
  ServedOrchestrator* orchestrator = static_cast<ServedOrchestrator*>(c_orchestrator);
  return orchestrator->Shutdown();
}
}
