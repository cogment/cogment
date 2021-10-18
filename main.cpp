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

#ifndef NDEBUG
  #define SPDLOG_ACTIVE_LEVEL SPDLOG_LEVEL_TRACE
#endif

#include "grpc++/grpc++.h"

#include "cogment/config_file.h"
#include "cogment/orchestrator.h"
#include "cogment/stub_pool.h"
#include "cogment/versions.h"

namespace cfg_file = cogment::cfg_file;

#include <prometheus/exposer.h>
#include <prometheus/registry.h>

#include "slt/settings.h"
#include "spdlog/sinks/daily_file_sink.h"
#include "spdlog/spdlog.h"

#include <csignal>
#include <filesystem>
#include <fstream>
#include <thread>

#include <execinfo.h>  // For backtrace
#include <unistd.h>    // STDERR_FINLENO

namespace settings {
slt::Setting lifecycle_port = slt::Setting_builder<std::uint16_t>()
                                  .with_default(9000)
                                  .with_description("The port to listen for trial lifecycle on")
                                  .with_env_variable("TRIAL_LIFECYCLE_PORT")
                                  .with_arg("lifecycle_port");

slt::Setting actor_port = slt::Setting_builder<std::uint16_t>()
                              .with_default(9000)
                              .with_description("The port to listen for trial actors on")
                              .with_env_variable("TRIAL_ACTOR_PORT")
                              .with_arg("actor_port");

slt::Setting config_file = slt::Setting_builder<std::string>()
                               .with_default("cogment.yaml")
                               .with_description("The trial configuration file")
                               .with_arg("config");

slt::Setting prometheus_port = slt::Setting_builder<std::uint16_t>()
                                   .with_default(0)
                                   .with_description("The port to broadcast prometheus metrics on")
                                   .with_env_variable("PROMETHEUS_PORT")
                                   .with_arg("prometheus_port");

slt::Setting display_version = slt::Setting_builder<bool>()
                                   .with_default(false)
                                   .with_description("Display the orchestrator's version and quit")
                                   .with_arg("version");

slt::Setting status_file = slt::Setting_builder<std::string>()
                               .with_default("")
                               .with_description("File to store orchestrator status to.")
                               .with_arg("status_file");

slt::Setting private_key = slt::Setting_builder<std::string>()
                               .with_default("")
                               .with_description("File containing PEM encoded private key.")
                               .with_arg("private_key");

slt::Setting root_cert = slt::Setting_builder<std::string>()
                             .with_default("")
                             .with_description("File containing a PEM encoded trusted root certificate.")
                             .with_arg("root_cert");

slt::Setting trust_chain = slt::Setting_builder<std::string>()
                               .with_default("")
                               .with_description("File containing a PEM encoded trust chain.")
                               .with_arg("trust_chain");

slt::Setting log_level = slt::Setting_builder<std::string>()
                             .with_default("info")
                             .with_description("Set minimum logging level (off, error, warning, info, debug, trace)")
                             .with_arg("log_level");

slt::Setting log_file = slt::Setting_builder<std::string>()
                            .with_default("")
                            .with_description("Set base file for daily log output")
                            .with_arg("log_file");
}  // namespace settings

namespace {
const std::string init_status_string = "I";
const std::string ready_status_string = "R";
const std::string term_status_string = "T";

void sigsegv_handler(int signal) {
  static constexpr size_t BACKTRACE_SIZE = 100;

  void* buffer[BACKTRACE_SIZE];
  int nptrs = backtrace(buffer, BACKTRACE_SIZE);

  // add2line can be used to get more info from output
  backtrace_symbols_fd(buffer, nptrs, STDERR_FILENO);

  std::_Exit(signal);
}

}  // namespace

int main(int argc, const char* argv[]) {
  signal(SIGSEGV, sigsegv_handler);

  slt::Settings_context ctx("orchestrator", argc, argv);
  if (ctx.help_requested()) {
    return 0;
  }
  if (settings::display_version.get()) {
    // This is used by the build process to tag docker images reliably.
    // It should remaing this terse.
    std::cout << COGMENT_ORCHESTRATOR_VERSION << "\n";
    return 0;
  }

  std::string log_filename = settings::log_file.get();
  if (!log_filename.empty()) {
    std::cout << "Daily log base filename: \"" << log_filename << "\"" << std::endl;
    auto logger = spdlog::daily_logger_mt("daily_logger", log_filename, 0, 0);
    spdlog::set_default_logger(logger);
  }

  const auto level_setting = spdlog::level::from_str(settings::log_level.get());
  spdlog::set_level(level_setting);

  spdlog::debug("Orchestrator starting with arguments:");
  spdlog::debug("\t--{}={}", settings::lifecycle_port.arg().value_or(""), settings::lifecycle_port.get());
  spdlog::debug("\t--{}={}", settings::actor_port.arg().value_or(""), settings::actor_port.get());
  spdlog::debug("\t--{}={}", settings::config_file.arg().value_or(""), settings::config_file.get());
  spdlog::debug("\t--{}={}", settings::prometheus_port.arg().value_or(""), settings::prometheus_port.get());
  spdlog::debug("\t--{}={}", settings::display_version.arg().value_or(""), settings::display_version.get());
  spdlog::debug("\t--{}={}", settings::status_file.arg().value_or(""), settings::status_file.get());
  spdlog::debug("\t--{}={}", settings::private_key.arg().value_or(""), settings::private_key.get());
  spdlog::debug("\t--{}={}", settings::root_cert.arg().value_or(""), settings::root_cert.get());
  spdlog::debug("\t--{}={}", settings::trust_chain.arg().value_or(""), settings::trust_chain.get());
  spdlog::debug("\t--{}={}", settings::log_level.arg().value_or(""), settings::log_level.get());
  spdlog::debug("\t   --> {}", level_setting);

  ctx.validate_all();

  std::optional<std::ofstream> status_file;

  if (settings::status_file.get() != "") {
    status_file = std::ofstream(settings::status_file.get());
    *status_file << init_status_string << std::flush;
  }

  std::shared_ptr<grpc::ServerCredentials> server_creds;
  std::shared_ptr<grpc::ChannelCredentials> client_creds;
  const bool using_ssl = !(settings::private_key.get().empty() && settings::root_cert.get().empty() &&
                           settings::trust_chain.get().empty());
  if (!using_ssl) {
    spdlog::info("Using unsecured communication");
    server_creds = grpc::InsecureServerCredentials();
    client_creds = grpc::InsecureChannelCredentials();
  }
  else {
    spdlog::info("Using secured TLS communication");

    auto private_key_file = std::ifstream(settings::private_key.get());
    if (!private_key_file.is_open() || !private_key_file.good()) {
      auto cwd = std::filesystem::current_path().string();
      spdlog::error("Could not open private key file: [{}/{}]", cwd, settings::private_key.get());
      return 1;
    }
    std::stringstream private_key;
    private_key << private_key_file.rdbuf();

    auto root_cert_file = std::ifstream(settings::root_cert.get());
    if (!root_cert_file.is_open() || !root_cert_file.good()) {
      auto cwd = std::filesystem::current_path().string();
      spdlog::error("Could not open root certificate file: [{}/{}]", cwd, settings::root_cert.get());
      return 1;
    }
    std::stringstream root_cert;
    root_cert << root_cert_file.rdbuf();

    auto trust_chain_file = std::ifstream(settings::trust_chain.get());
    if (!trust_chain_file.is_open() || !trust_chain_file.good()) {
      auto cwd = std::filesystem::current_path().string();
      spdlog::error("Could not open certificate trust chain file: [{}/{}]", cwd, settings::trust_chain.get());
      return 1;
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

  int return_value = 0;
  try {
    // Create a scope so that all local variables created from this
    // point on get properly destroyed BEFORE the status is updated.
    YAML::Node cogment_yaml;
    try {
      cogment_yaml = YAML::LoadFile(settings::config_file.get());
    }
    catch (const std::exception& exc) {
      spdlog::error("Failed to load config file [{}]: {}", settings::config_file.get(), exc.what());
      return 1;
    }

    spdlog::info("Cogment Orchestrator [{}]", COGMENT_ORCHESTRATOR_VERSION);
    spdlog::info("Cogment API [{}]", COGMENT_API_VERSION);

    // ******************* Endpoints *******************
    auto lifecycle_endpoint = std::string("0.0.0.0:") + std::to_string(settings::lifecycle_port.get());
    auto actor_endpoint = std::string("0.0.0.0:") + std::to_string(settings::actor_port.get());

    // ******************* Monitoring *******************
    std::unique_ptr<prometheus::Exposer> metrics_exposer;
    std::shared_ptr<prometheus::Registry> metrics_registry;
    if (settings::prometheus_port.get() > 0) {
      auto prometheus_endpoint = std::string("0.0.0.0:") + std::to_string(settings::prometheus_port.get());
      spdlog::info("Starting prometheus at [{}]", prometheus_endpoint);

      metrics_exposer = std::make_unique<prometheus::Exposer>(prometheus_endpoint);
      metrics_registry = std::make_shared<prometheus::Registry>();
      metrics_exposer->RegisterCollectable(metrics_registry);
    }
    else {
      spdlog::info("Prometheus disabled");
    }

    // ******************* Orchestrator *******************
    cogment::Trial_spec trial_spec(cogment_yaml);
    auto params = cogment::load_params(cogment_yaml, trial_spec);

    cogment::Orchestrator orchestrator(std::move(trial_spec), std::move(params), client_creds, metrics_registry.get());

    // ******************* Networking *******************
    cogment::StubPool<cogmentAPI::TrialHooksSP> hook_stubs(orchestrator.channel_pool());
    std::vector<std::shared_ptr<cogment::StubPool<cogmentAPI::TrialHooksSP>::Entry>> hooks;

    int nb_prehooks = 0;
    if (cogment_yaml[cfg_file::trial_key]) {
      for (auto hook : cogment_yaml[cfg_file::trial_key][cfg_file::t_pre_hooks_key]) {
        hooks.push_back(hook_stubs.get_stub_entry(hook.as<std::string>()));
        orchestrator.add_prehook(hooks.back());
        nb_prehooks++;
      }
    }
    spdlog::info("[{}] prehooks defined", nb_prehooks);

    std::string datalog_type = "none";
    if (cogment_yaml[cfg_file::datalog_key] && cogment_yaml[cfg_file::datalog_key][cfg_file::d_type_key]) {
      datalog_type = cogment_yaml[cfg_file::datalog_key][cfg_file::d_type_key].as<std::string>();
    }

    std::transform(datalog_type.begin(), datalog_type.end(), datalog_type.begin(), ::tolower);
    if (datalog_type == "none") {
      orchestrator.set_log_exporter("");
    }
    else if (datalog_type == "grpc") {
      if (cogment_yaml[cfg_file::datalog_key][cfg_file::d_endpoint_key] == nullptr) {
        spdlog::error("Datalog config 'endpoint' could not be found (was 'url' in v1).");
        return 1;
      }
      auto url = cogment_yaml[cfg_file::datalog_key][cfg_file::d_endpoint_key].as<std::string>();
      orchestrator.set_log_exporter(url);
    }
    else {
      spdlog::error("Invalid datalog specification [{}]", datalog_type);
      return 1;
    }

    std::vector<std::unique_ptr<grpc::Server>> servers;

    {
      grpc::ServerBuilder builder;
      builder.AddListeningPort(lifecycle_endpoint, server_creds);
      builder.RegisterService(orchestrator.trial_lifecycle_service());

      // If the lifecycle endpoint is the same as the ClientActorSP, then run them
      // off the same server, otherwise, start a second server.
      if (lifecycle_endpoint == actor_endpoint) {
        SPDLOG_TRACE("Actor endpoint has same server as Trial Lifecycle");
        builder.RegisterService(orchestrator.actor_service());
      }
      else {
        SPDLOG_TRACE("Actor endpoint has a different server than Trial Lifecycle");
        grpc::ServerBuilder actor_builder;
        actor_builder.AddListeningPort(actor_endpoint, server_creds);
        actor_builder.RegisterService(orchestrator.actor_service());
        servers.emplace_back(actor_builder.BuildAndStart());
      }

      SPDLOG_TRACE("Building main server");
      auto server = builder.BuildAndStart();
      SPDLOG_TRACE("Main server built");
      servers.emplace_back(std::move(server));
    }

    if (status_file) {
      *status_file << ready_status_string << std::flush;
    }
    spdlog::info("Server listening for lifecycle on [{}]", lifecycle_endpoint);
    spdlog::info("Server listening for Actors on [{}]", actor_endpoint);

    // Wait for a termination signal.
    sigset_t sig_set;
    sigemptyset(&sig_set);
    sigaddset(&sig_set, SIGINT);
    sigaddset(&sig_set, SIGTERM);

    // Prevent normal handling of these signals.
    pthread_sigmask(SIG_BLOCK, &sig_set, nullptr);

    SPDLOG_TRACE("Main done");
    int sig = 0;
    sigwait(&sig_set, &sig);  // Blocking
    spdlog::info("Shutting down...");
  }
  catch(const std::exception& exc) {
    spdlog::error("Exception in Main: [{}]", exc.what());
    return_value = 1;
  }
  catch(...) {
    spdlog::error("Exception in Main");
    return_value = 1;
  }

  if (status_file) {
    *status_file << term_status_string << std::flush;
  }

  return return_value;
}
