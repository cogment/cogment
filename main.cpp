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

#include "cogment/orchestrator.h"
#include "cogment/stub_pool.h"
#include "cogment/versions.h"

#include <prometheus/exposer.h>
#include <prometheus/registry.h>

#include "slt/settings.h"
#include "spdlog/spdlog.h"

#include "yaml-cpp/yaml.h"

#include <csignal>
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
}  // namespace settings

namespace {
const std::string init_status_string = "I";
const std::string ready_status_string = "R";
const std::string term_status_string = "T";
}  // namespace

int main(int argc, const char* argv[]) {
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

  ctx.validate_all();

  std::optional<std::ofstream> status_file;

  if (settings::status_file.get() != "") {
    status_file = std::ofstream(settings::status_file.get());
    *status_file << init_status_string << std::flush;
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
    spdlog::info("starting prometheus at: {}", prometheus_endpoint);
    prometheus::Exposer ExposePublicParser(prometheus_endpoint);

    // ******************* Orchestrator *******************
    cogment::Trial_spec trial_spec(cogment_yaml);
    auto params = cogment::load_params(cogment_yaml, trial_spec);

    cogment::Orchestrator orchestrator(std::move(trial_spec), std::move(params));

    // ******************* Networking *******************
    cogment::Stub_pool<cogment::TrialHooks> hook_stubs(orchestrator.channel_pool(), orchestrator.client_queue());
    std::vector<std::shared_ptr<cogment::Stub_pool<cogment::TrialHooks>::Entry>> hooks;

    if (cogment_yaml["trial"]) {
      for (auto hook : cogment_yaml["trial"]["pre_hooks"]) {
        hooks.push_back(hook_stubs.get_stub(hook.as<std::string>()));
        orchestrator.add_prehook(&hooks.back()->stub);
      }
    }

    if (cogment_yaml["datalog"] && cogment_yaml["datalog"]["type"]) {
      auto datalog_type = cogment_yaml["datalog"]["type"].as<std::string>();
      auto datalog = cogment::Datalog_storage_interface::create(datalog_type, cogment_yaml);
      orchestrator.set_log_exporter(std::move(datalog));
    }

    std::vector<rpc::server::Server> servers;

    rpc::server::Config cfg;
    cfg.add_default_listening_queues({server_queues.begin(), server_queues.end()})
        .add_feature(rpc::Reflection_feature())
        .add_service(orchestrator.trial_lifecycle_service())
        .add_listening_port(lifecycle_endpoint);

    // If the lifecycle endpoint is the same as the ActorEndpoint, then run them
    // off the same server, otherwise, start a second server.
    if (lifecycle_endpoint == actor_endpoint) {
      cfg.add_service(orchestrator.actor_service());
    }
    else {
      rpc::server::Config actor_cfg;
      actor_cfg.add_default_listening_queues({server_queues.begin(), server_queues.end()})
          .add_feature(rpc::Reflection_feature())
          .add_service(orchestrator.actor_service())
          .add_listening_port(actor_endpoint);

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
