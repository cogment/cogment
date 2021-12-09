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

#include "cogment/config_file.h"
#include "cogment/orchestrator.h"
#include "cogment/utils.h"
#include "cogment/versions.h"
#include "cogment/services/actor_service.h"
#include "cogment/services/trial_lifecycle_service.h"

#include "prometheus/exposer.h"
#include "prometheus/registry.h"

#include "grpc++/grpc++.h"
#include "slt/settings.h"
#include "spdlog/spdlog.h"
#include "spdlog/sinks/daily_file_sink.h"
#include "spdlog/sinks/stdout_color_sinks.h"

#include <csignal>
#include <filesystem>
#include <fstream>
#include <thread>

#include <execinfo.h>  // backtrace
#include <unistd.h>    // STDERR_FILENO

namespace cfg_file = cogment::cfg_file;

namespace settings {
slt::Setting deprecated_lifecycle_port = slt::Setting_builder<std::string>()
                                             .with_default("")
                                             .with_description("DEPRECATED")
                                             .with_env_variable("TRIAL_LIFECYCLE_PORT");

slt::Setting lifecycle_port = slt::Setting_builder<std::uint16_t>()
                                  .with_default(9000)
                                  .with_description("The port to listen for trial lifecycle on")
                                  .with_env_variable("COGMENT_LIFECYCLE_PORT")
                                  .with_arg("lifecycle_port");

slt::Setting deprecated_actor_port = slt::Setting_builder<std::string>()
                                         .with_default("")
                                         .with_description("DEPRECATED")
                                         .with_env_variable("TRIAL_ACTOR_PORT");

slt::Setting actor_port = slt::Setting_builder<std::uint16_t>()
                              .with_default(9000)
                              .with_description("The port to listen for trial actors on")
                              .with_env_variable("COGMENT_ACTOR_PORT")
                              .with_arg("actor_port");

slt::Setting deprecated_config_file =
    slt::Setting_builder<std::string>().with_default("").with_description("DEPRECATED").with_arg("config");

slt::Setting default_params_file = slt::Setting_builder<std::string>()
                                       .with_default("")
                                       .with_description("Default trial parameters file name")
                                       .with_env_variable("COGMENT_DEFAULT_PARAMS_FILE")
                                       .with_arg("params");

slt::Setting pre_trial_hooks = slt::Setting_builder<std::string>()
                                   .with_default("")
                                   .with_description("Pre-trial hook gRPC endpoints, separated by a comma")
                                   .with_env_variable("COGMENT_PRE_TRIAL_HOOKS")
                                   .with_arg("pre_trial_hooks");

slt::Setting deprecated_prometheus_port = slt::Setting_builder<std::string>()
                                              .with_default("")
                                              .with_description("DEPRECATED")
                                              .with_env_variable("PROMETHEUS_PORT");

slt::Setting prometheus_port = slt::Setting_builder<std::uint16_t>()
                                   .with_default(0)
                                   .with_description("The port to broadcast prometheus metrics on")
                                   .with_env_variable("COGMENT_ORCHESTRATOR_PROMETHEUS_PORT")
                                   .with_arg("prometheus_port");

slt::Setting display_version = slt::Setting_builder<bool>()
                                   .with_default(false)
                                   .with_description("Display the Orchestrator's version and quit")
                                   .with_arg("version");

slt::Setting status_file = slt::Setting_builder<std::string>()
                               .with_default("")
                               .with_description("File to store the Orchestrator status to.")
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

slt::Setting gc_frequency = slt::Setting_builder<std::uint32_t>()
                                .with_default(10)
                                .with_description("Number of trials between garbage collection runs")
                                .with_arg("gc_frequency");
}  // namespace settings

namespace {
constexpr char INIT_STATUS_STRING = 'I';
constexpr char READY_STATUS_STRING = 'R';
constexpr char TERM_STATUS_STRING = 'T';
constexpr uint32_t DEFAULT_MAX_INACTIVITY = 30;

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

  auto cwd = std::filesystem::current_path().string();

  std::string log_filename = settings::log_file.get();
  try {
    // Replace default unnamed logger with a named one, so we can put back an unnamed logger
    spdlog::set_default_logger(spdlog::stdout_color_mt("console"));

    if (log_filename.empty()) {
      spdlog::set_default_logger(spdlog::stdout_color_mt(""));
    }
    else {
      std::cout << "Daily log base filename (date will be added): \"" << log_filename << "\"" << std::endl;
      auto logger = spdlog::daily_logger_mt("", log_filename, 0, 0);
      spdlog::set_default_logger(logger);
    }
  }
  catch (const std::exception& exc) {
    std::cerr << "Failed to set log filename [" << cwd << "] \"" << log_filename << "\": " << exc.what() << std::endl;
    return -1;
  }
  catch (...) {
    std::cerr << "Failed to set log filename [" << cwd << "] \"" << log_filename << "\"" << std::endl;
    return -1;
  }

  std::string log_level = settings::log_level.get();
  std::transform(log_level.begin(), log_level.end(), log_level.begin(), ::tolower);
  try {
    if (!log_level.empty()) {
      const auto level_setting = spdlog::level::from_str(log_level);

      // spdlog returns "off" if an unknown string is given!
      if (level_setting == spdlog::level::level_enum::off && log_level != "off") {
        throw MakeException("Unknown log level setting");
      }

      spdlog::set_level(level_setting);
    }
  }
  catch (const std::exception& exc) {
    std::cerr << "Failed to set log level \"" << log_level << "\": " << exc.what() << std::endl;
    return -1;
  }
  catch (...) {
    std::cerr << "Failed to set log level \"" << log_level << "\": " << std::endl;
    return -1;
  }

  std::ofstream status_file;
  if (!settings::status_file.get().empty()) {
    status_file = std::ofstream(settings::status_file.get());
    if (status_file.is_open() && status_file.good()) {
      status_file << INIT_STATUS_STRING << std::flush;
    }
    else {
      spdlog::error("Failed to open status file [{}] [{}]", cwd, settings::status_file.get());
      return -1;
    }
  }

  spdlog::debug("Orchestrator starting with arguments:");
  spdlog::debug("\t--{}={}", settings::lifecycle_port.arg().value_or(""), settings::lifecycle_port.get());
  spdlog::debug("\t--{}={}", settings::actor_port.arg().value_or(""), settings::actor_port.get());
  spdlog::debug("\t--{}={}", settings::default_params_file.arg().value_or(""), settings::default_params_file.get());
  spdlog::debug("\t--{}={}", settings::prometheus_port.arg().value_or(""), settings::prometheus_port.get());
  spdlog::debug("\t--{}={}", settings::display_version.arg().value_or(""), settings::display_version.get());
  spdlog::debug("\t--{}={}", settings::status_file.arg().value_or(""), settings::status_file.get());
  spdlog::debug("\t--{}={}", settings::private_key.arg().value_or(""), settings::private_key.get());
  spdlog::debug("\t--{}={}", settings::root_cert.arg().value_or(""), settings::root_cert.get());
  spdlog::debug("\t--{}={}", settings::trust_chain.arg().value_or(""), settings::trust_chain.get());
  spdlog::debug("\t--{}={}", settings::log_level.arg().value_or(""), settings::log_level.get());
  spdlog::debug("\t--{}={}", settings::gc_frequency.arg().value_or(""), settings::gc_frequency.get());

  spdlog::info("Cogment Orchestrator version [{}]", COGMENT_ORCHESTRATOR_VERSION);
  spdlog::info("Cogment API version [{}]", COGMENT_API_VERSION);

  int return_value = 0;
  try {
    ctx.validate_all();

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
        throw MakeException("Could not open private key file: [{}] [{}]", cwd, settings::private_key.get());
      }
      std::stringstream private_key;
      private_key << private_key_file.rdbuf();

      auto root_cert_file = std::ifstream(settings::root_cert.get());
      if (!root_cert_file.is_open() || !root_cert_file.good()) {
        throw MakeException("Could not open root certificate file: [{}] [{}]", cwd, settings::root_cert.get());
      }
      std::stringstream root_cert;
      root_cert << root_cert_file.rdbuf();

      auto trust_chain_file = std::ifstream(settings::trust_chain.get());
      if (!trust_chain_file.is_open() || !trust_chain_file.good()) {
        throw MakeException("Could not open certificate trust chain file: [{}] [{}]", cwd, settings::trust_chain.get());
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
    if (!settings::deprecated_lifecycle_port.get().empty()) {
      spdlog::warn("Environment variable [{}] is deprecated.  Use [{}].",
                   settings::deprecated_lifecycle_port.env_var().value(), settings::lifecycle_port.env_var().value());
    }
    auto lifecycle_endpoint = std::string("0.0.0.0:") + std::to_string(settings::lifecycle_port.get());

    if (!settings::deprecated_actor_port.get().empty()) {
      spdlog::warn("Environment variable [{}] is deprecated.  Use [{}].",
                   settings::deprecated_actor_port.env_var().value(), settings::actor_port.env_var().value());
    }
    auto actor_endpoint = std::string("0.0.0.0:") + std::to_string(settings::actor_port.get());

    // ******************* Monitoring *******************
    std::unique_ptr<prometheus::Exposer> metrics_exposer;
    std::shared_ptr<prometheus::Registry> metrics_registry;
    if (!settings::deprecated_prometheus_port.get().empty()) {
      spdlog::warn("Environment variable [{}] is deprecated.  Use [{}].",
                   settings::deprecated_prometheus_port.env_var().value(), settings::prometheus_port.env_var().value());
    }
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
    YAML::Node params_yaml;
    bool deprecated_config_file_loaded = false;
    bool params_file_loaded = false;
    if (!settings::default_params_file.get().empty()) {
      try {
        params_yaml = YAML::LoadFile(settings::default_params_file.get());
        params_file_loaded = true;
      }
      catch (const std::exception& exc) {
        throw MakeException("Failed to load default parameters file [{}]: {}", settings::default_params_file.get(),
                            exc.what());
      }
    }
    else if (!settings::deprecated_config_file.get().empty()) {
      spdlog::warn("Config file use is deprecated.");
      try {
        params_yaml = YAML::LoadFile(settings::deprecated_config_file.get());
        deprecated_config_file_loaded = true;
      }
      catch (const std::exception& exc) {
        throw MakeException("Failed to load default deprecated config file [{}]: {}",
                            settings::deprecated_config_file.get(), exc.what());
      }
    }
    else {
      spdlog::info("No default parameters.");
    }
    auto params = cogment::load_params(params_yaml);

    if (params_file_loaded) {
      if (params_yaml[cfg_file::params_key] == nullptr ||
          params_yaml[cfg_file::params_key][cfg_file::p_max_inactivity_key] == nullptr) {
        params.set_max_inactivity(DEFAULT_MAX_INACTIVITY);
      }
    }
    else if (deprecated_config_file_loaded) {
      if (params_yaml[cfg_file::datalog_key] != nullptr) {
        std::string type;
        if (params_yaml[cfg_file::datalog_key][cfg_file::d_type_key] != nullptr) {
          type = params_yaml[cfg_file::datalog_key][cfg_file::d_type_key].as<std::string>();
        }
        std::transform(type.begin(), type.end(), type.begin(), ::tolower);

        if (type == "grpc") {
          auto datalog = params.mutable_datalog();

          if (params_yaml[cfg_file::datalog_key][cfg_file::d_url_key] != nullptr) {
            auto url = params_yaml[cfg_file::datalog_key][cfg_file::d_url_key].as<std::string>();
            url.insert(0, "grpc://");
            datalog->set_endpoint(url);
          }
        }
      }
    }

    cogment::Orchestrator orchestrator(std::move(params), settings::gc_frequency.get(), client_creds,
                                       metrics_registry.get());

    // ******************* Networking *******************
    int nb_prehooks = 0;
    const auto hooks_urls = split(settings::pre_trial_hooks.get(), ',');
    if (!hooks_urls.empty()) {
      if (deprecated_config_file_loaded) {
        spdlog::warn(
            "Deprecated config file hook definition will be ignored. Using command line or environment variable.");
      }
      for (auto& url : hooks_urls) {
        orchestrator.add_prehook(url);
        nb_prehooks++;
      }
    }
    else {
      if (deprecated_config_file_loaded) {
        for (auto hook : params_yaml[cfg_file::trial_key][cfg_file::t_pre_hooks_key]) {
          orchestrator.add_prehook(hook.as<std::string>());
          nb_prehooks++;
        }
      }
      else {
        if (!params_file_loaded) {
          throw MakeException("No default parameter file and no pre-trial hook definition.");
        }
      }
    }
    spdlog::info("[{}] pre-trial hooks defined", nb_prehooks);

    cogment::ActorService actor_service(&orchestrator);
    cogment::TrialLifecycleService trial_lifecycle_service(&orchestrator);
    std::vector<std::unique_ptr<grpc::Server>> servers;
    {
      grpc::ServerBuilder builder;
      builder.AddListeningPort(lifecycle_endpoint, server_creds);
      builder.RegisterService(&trial_lifecycle_service);

      // If the lifecycle endpoint is the same as the ClientActorSP, then run them
      // off the same server, otherwise, start a second server.
      if (lifecycle_endpoint == actor_endpoint) {
        builder.RegisterService(&actor_service);
      }
      else {
        grpc::ServerBuilder actor_builder;
        actor_builder.AddListeningPort(actor_endpoint, server_creds);
        actor_builder.RegisterService(&actor_service);
        servers.emplace_back(actor_builder.BuildAndStart());
      }

      auto server = builder.BuildAndStart();
      spdlog::debug("Main server started");
      servers.emplace_back(std::move(server));
    }

    if (status_file.is_open() && status_file.good()) {
      status_file << READY_STATUS_STRING << std::flush;
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

    spdlog::debug("Main done");
    int sig = 0;
    sigwait(&sig_set, &sig);  // Blocking
    spdlog::info("Shutting down...");
  }
  catch (const std::exception& exc) {
    spdlog::error("Failure: {}", exc.what());
    return_value = -1;
  }
  catch (...) {
    spdlog::error("Failure");
    return_value = -1;
  }

  if (status_file.is_open() && status_file.good()) {
    status_file << TERM_STATUS_STRING << std::flush;
  }

  return return_value;
}
