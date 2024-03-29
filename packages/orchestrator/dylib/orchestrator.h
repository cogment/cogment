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

#ifndef COGMENT_ORCHESTRATOR_H
#define COGMENT_ORCHESTRATOR_H

#if defined(_MSC_VER)
  //  Microsoft
  #define ORCH_EXPORT __declspec(dllexport)
  #define ORCH_IMPORT __declspec(dllimport)
#else
  #define ORCH_EXPORT
  #define ORCH_IMPORT
#endif

#include <time.h>

#ifdef __cplusplus
extern "C" {
#endif

ORCH_EXPORT void* cogment_orchestrator_options_create();
ORCH_EXPORT void cogment_orchestrator_options_destroy(void* options);

ORCH_EXPORT void cogment_orchestrator_options_set_lifecycle_port(void* options, unsigned int port);
ORCH_EXPORT void cogment_orchestrator_options_set_actor_port(void* options, unsigned int port);
ORCH_EXPORT void cogment_orchestrator_options_set_default_params_file(void* options, char* path);
ORCH_EXPORT void cogment_orchestrator_options_add_pretrial_hook(void* options, char* pretrial_hook_endpoint);
ORCH_EXPORT void cogment_orchestrator_options_add_directory_service(void* options, char* directory_service_endpoint);
ORCH_EXPORT void cogment_orchestrator_options_set_directory_auth_token(void* options, char* token);
ORCH_EXPORT void cogment_orchestrator_options_set_auto_registration(void* options, unsigned int autoRegister);
ORCH_EXPORT void cogment_orchestrator_options_set_directory_register_host(void* options, char* host);
ORCH_EXPORT void cogment_orchestrator_options_set_directory_properties(void* options, char* properties);
ORCH_EXPORT void cogment_orchestrator_options_set_prometheus_port(void* options, unsigned int port);
ORCH_EXPORT void cogment_orchestrator_options_set_private_key_file(void* options, char* path);
ORCH_EXPORT void cogment_orchestrator_options_set_root_certificate_file(void* options, char* path);
ORCH_EXPORT void cogment_orchestrator_options_trust_chain_file(void* options, char* path);
ORCH_EXPORT void cogment_orchestrator_options_garbage_collector_frequency(void* options, unsigned int frequency);
typedef void (*CogmentOrchestratorLogger)(void* ctx, const char* logger_name, int log_level, time_t timestamp,
                                          size_t thread_id, const char* file_name, int line, const char* function_name,
                                          const char* message);
ORCH_EXPORT void cogment_orchestrator_options_set_logging(void* options, const char* level, void* ctx,
                                                          CogmentOrchestratorLogger logger);

const int COGMENT_ORCHESTRATOR_INIT_STATUS = 1;
const int COGMENT_ORCHESTRATOR_READY_STATUS = 2;
const int COGMENT_ORCHESTRATOR_TERMINATED_STATUS = 3;

typedef void (*CogmentOrchestratorStatusListener)(void* ctx, int status);
ORCH_EXPORT void cogment_orchestrator_options_set_status_listener(void* options, void* ctx,
                                                                  CogmentOrchestratorStatusListener status_listener);

ORCH_EXPORT void* cogment_orchestrator_create_and_start(void* options);
ORCH_EXPORT void cogment_orchestrator_destroy(void* orchestrator);
ORCH_EXPORT int cogment_orchestrator_wait_for_termination(void* orchestrator);
ORCH_EXPORT void cogment_orchestrator_shutdown(void* orchestrator);

#ifdef __cplusplus
}
#endif

#endif
