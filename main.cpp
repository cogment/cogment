// Copyright 2020 Artificial Intelligence Redefined <dev+cogment@ai-r.com>
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

#include "easy_grpc/easy_grpc.h"
#include "easy_grpc_reflection/reflection.h"

namespace rpc = easy_grpc;

#include "cogment/config_file.h"
#include "cogment/orchestrator.h"
#include "cogment/stub_pool.h"
#include "cogment/versions.h"

namespace cfg_file = cogment::cfg_file;

#include <prometheus/exposer.h>
#include <prometheus/registry.h>

#include "slt/settings.h"
#include "spdlog/spdlog.h"

#include <csignal>
#include <filesystem>
#include <fstream>
#include <thread>

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
                                   .with_default(8080)
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
}  // namespace settings

namespace {
const std::string init_status_string = "I";
const std::string ready_status_string = "R";
const std::string term_status_string = "T";
}  // namespace

int main(int argc, const char* argv[]) {
  spdlog::set_level(spdlog::level::debug);  // Set global log level to debug

  slt::Settings_context ctx("orchestrator", argc, argv);
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

  if (ctx.help_requested()) {
    return 0;
  }

  if (settings::display_version.get()) {
    // This is used by the build process to tag docker images reliably.
    // It should remaing this terse.
    std::cout << COGMENT_ORCHESTRATOR_VERSION << "\n";
    return 0;
  }

  ctx.validate_all();

  std::optional<std::ofstream> status_file;

  if (settings::status_file.get() != "") {
    status_file = std::ofstream(settings::status_file.get());
    *status_file << init_status_string << std::flush;
  }

  std::shared_ptr<rpc::server::Credentials> server_creds;
  std::shared_ptr<rpc::client::Credentials> client_creds;
  const bool using_ssl = !(settings::private_key.get().empty() && settings::root_cert.get().empty() &&
                           settings::trust_chain.get().empty());
  if (using_ssl) {
    auto private_key_file = std::ifstream(settings::private_key.get());
    if (!private_key_file.is_open() || !private_key_file.good()) {
      auto cwd = std::filesystem::current_path().string();
      spdlog::error("Could not open private key file: {}/{}", cwd, settings::private_key.get());
      return 1;
    }
    std::stringstream private_key;
    private_key << private_key_file.rdbuf();

    auto root_cert_file = std::ifstream(settings::root_cert.get());
    if (!root_cert_file.is_open() || !root_cert_file.good()) {
      auto cwd = std::filesystem::current_path().string();
      spdlog::error("Could not open root certificate file: {}/{}", cwd, settings::root_cert.get());
      return 1;
    }
    std::stringstream root_cert;
    root_cert << root_cert_file.rdbuf();

    auto trust_chain_file = std::ifstream(settings::trust_chain.get());
    if (!trust_chain_file.is_open() || !trust_chain_file.good()) {
      auto cwd = std::filesystem::current_path().string();
      spdlog::error("Could not open certificate trust chain file: {}/{}", cwd, settings::trust_chain.get());
      return 1;
    }
    std::stringstream trust_chain;
    trust_chain << trust_chain_file.rdbuf();

    server_creds = std::make_shared<rpc::server::Credentials>(root_cert.str().c_str(), private_key.str().c_str(),
                                                              trust_chain.str().c_str());
    client_creds = std::make_shared<rpc::client::Credentials>(root_cert.str().c_str(), private_key.str().c_str(),
                                                              trust_chain.str().c_str());
  }

  {  // Create a scope so that all local variables created from this
    // point on get properly destroyed BEFORE the status is updated.
    YAML::Node cogment_yaml;
    try {
      cogment_yaml = YAML::LoadFile(settings::config_file.get());
    } catch (std::exception& e) {
      spdlog::error("failed to load {}: {}", settings::config_file.get(), e.what());
      return 1;
    }

    rpc::Environment grpc_env;

    std::array<rpc::Completion_queue, 4> server_queues;

    spdlog::info("Cogment Orchestrator v. {}", COGMENT_ORCHESTRATOR_VERSION);
    spdlog::info("Cogment API v. {}", COGMENT_API_VERSION);

    // ******************* Endpoints *******************
    auto lifecycle_endpoint = fmt::format("0.0.0.0:{}", settings::lifecycle_port.get());
    auto actor_endpoint = fmt::format("0.0.0.0:{}", settings::actor_port.get());
    auto prometheus_endpoint = fmt::format("0.0.0.0:{}", settings::prometheus_port.get());

    // ******************* Monitoring *******************
    spdlog::info("Starting prometheus at: {}", prometheus_endpoint);
    prometheus::Exposer ExposePublicParser(prometheus_endpoint);

    // ******************* Orchestrator *******************
    cogment::Trial_spec trial_spec(cogment_yaml);
    auto params = cogment::load_params(cogment_yaml, trial_spec);

    cogment::Orchestrator orchestrator(std::move(trial_spec), std::move(params), client_creds);

    // ******************* Networking *******************
    cogment::Stub_pool<cogment::TrialHooks> hook_stubs(orchestrator.channel_pool(), orchestrator.client_queue());
    std::vector<std::shared_ptr<cogment::Stub_pool<cogment::TrialHooks>::Entry>> hooks;

    if (cogment_yaml[cfg_file::trial_key]) {
      for (auto hook : cogment_yaml[cfg_file::trial_key][cfg_file::t_pre_hooks_key]) {
        hooks.push_back(hook_stubs.get_stub(hook.as<std::string>()));
        orchestrator.add_prehook(&hooks.back()->stub);
      }
    }

    if (cogment_yaml[cfg_file::datalog_key] && cogment_yaml[cfg_file::datalog_key][cfg_file::d_type_key]) {
      auto datalog_type = cogment_yaml[cfg_file::datalog_key][cfg_file::d_type_key].as<std::string>();
      auto datalog = cogment::Datalog_storage_interface::create(datalog_type, cogment_yaml);
      orchestrator.set_log_exporter(std::move(datalog));
    }

    std::vector<rpc::server::Server> servers;

    rpc::server::Config cfg;
    cfg.add_default_listening_queues({server_queues.begin(), server_queues.end()})
        .add_feature(rpc::Reflection_feature())
        .add_service(orchestrator.trial_lifecycle_service())
        .add_listening_port(lifecycle_endpoint, server_creds);

    // If the lifecycle endpoint is the same as the ClientActor, then run them
    // off the same server, otherwise, start a second server.
    if (lifecycle_endpoint == actor_endpoint) {
      cfg.add_service(orchestrator.actor_service());
    }
    else {
      rpc::server::Config actor_cfg;
      actor_cfg.add_default_listening_queues({server_queues.begin(), server_queues.end()})
          .add_feature(rpc::Reflection_feature())
          .add_service(orchestrator.actor_service())
          .add_listening_port(actor_endpoint, server_creds);

      servers.emplace_back(std::move(actor_cfg));
    }

    servers.emplace_back(std::move(cfg));

    if (status_file) {
      *status_file << ready_status_string << std::flush;
    }

    spdlog::info("Server listening for lifecycle on {}", lifecycle_endpoint);
    spdlog::info("Server listening for Actors on {}", actor_endpoint);

    // Wait for a termination signal.
    sigset_t sig_set;
    sigemptyset(&sig_set);
    sigaddset(&sig_set, SIGINT);
    sigaddset(&sig_set, SIGTERM);

    // Prevent normal handling of these signals.
    pthread_sigmask(SIG_BLOCK, &sig_set, nullptr);

    int sig = 0;
    sigwait(&sig_set, &sig);
    spdlog::info("Shutting down...");
  }

  if (status_file) {
    *status_file << term_status_string << std::flush;
  }

  return 0;
}
