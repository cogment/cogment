find_package(Protobuf REQUIRED)
find_package(yaml-cpp REQUIRED)

set(THREADS_PREFER_PTHREAD_FLAG ON)
find_package(Threads REQUIRED)

file(READ version.txt COGMENT_ORCH_VERSION)

configure_file("${CMAKE_CURRENT_SOURCE_DIR}/cogment/orch_config.h.in"
               "${CMAKE_CURRENT_SOURCE_DIR}/cogment/orch_config.h" )


set(COGMENT_API_DIR ${CMAKE_CURRENT_BINARY_DIR}/cogment-api)
file(MAKE_DIRECTORY ${COGMENT_API_DIR}/cogment/api/)

set(GENERATED_PROTOBUF_PATH ${CMAKE_CURRENT_BINARY_DIR}/generated)
file(MAKE_DIRECTORY ${GENERATED_PROTOBUF_PATH})

set(COGMENT_API_SRC "")
set(PROTO_SRC "")

macro(retrieve_protos)
        file(READ ../.cogment-api.yml COGMENT_API_VERSION_YML)
        string(REGEX MATCH "cogment_api_path: \"?(.+)\"?\n" COGMENT_API_PATH_MATCH ${COGMENT_API_VERSION_YML})
        set(COGMENT_API_PATH ${CMAKE_MATCH_1})
        if(COGMENT_API_PATH)
          # Local cogment api
          message("using local cogment api at \"${COGMENT_API_PATH}\"")
          add_custom_command(
                  OUTPUT ${COGMENT_API_SRC}
                  COMMAND ${CMAKE_COMMAND} -E copy_directory ${COGMENT_API_PATH} ${COGMENT_API_DIR}/cogment/api
                  COMMENT "Copying cogment api from \"${COGMENT_API_PATH}\""
                  VERBATIM
          )
        else()
          # Remote cogment api
          string(REGEX MATCH "cogment_api_version: \"?([a-zA-Z0-9\.]+)\"?\n" COGMENT_API_VERSION_MATCH ${COGMENT_API_VERSION_YML})
          set(COGMENT_API_VERSION ${CMAKE_MATCH_1})
          set(COGMENT_API_ARCHIVE_PATH ${CMAKE_CURRENT_BINARY_DIR}/cogment-api-${COGMENT_API_VERSION}.tar.gz)
          set(COGMENT_API_URL "https://cogment.github.io/cogment-api/${COGMENT_API_VERSION}/cogment-api-${COGMENT_API_VERSION}.tar.gz")

          message("fetching cogment api version \"${COGMENT_API_VERSION}\" from \"${COGMENT_API_URL}\"")
          file(
                  DOWNLOAD  ${COGMENT_API_URL}
                            ${COGMENT_API_ARCHIVE_PATH}
                  SHOW_PROGRESS
                  TLS_VERIFY ON
          )
          add_custom_command(
                  OUTPUT  ${COGMENT_API_SRC}
                  COMMAND ${CMAKE_COMMAND} -E tar xzf ${COGMENT_API_ARCHIVE_PATH}
                  MAIN_DEPENDENCY ${COGMENT_API_ARCHIVE_PATH}
                  WORKING_DIRECTORY ${COGMENT_API_DIR}/cogment/api
                  COMMENT "Unpacking the cogment-api archive, version \"${COGMENT_API_VERSION}\""
                  VERBATIM
          )
        endif()
endmacro()

macro(compile_proto PROTO)
        add_custom_command(
                OUTPUT  "${GENERATED_PROTOBUF_PATH}/cogment/api/${PROTO}.pb.h"
                        "${GENERATED_PROTOBUF_PATH}/cogment/api/${PROTO}.pb.cc"
                COMMAND ${Protobuf_PROTOC_EXECUTABLE}
                ARGS
                "--proto_path=${COGMENT_API_DIR}/"
                "--cpp_out=${GENERATED_PROTOBUF_PATH}"
                "${COGMENT_API_DIR}/cogment/api/${PROTO}.proto"
                MAIN_DEPENDENCY ${COGMENT_API_DIR}/cogment/api/${PROTO}.proto
          )

        list(APPEND COGMENT_API_SRC ${COGMENT_API_DIR}/cogment/api/${PROTO}.proto)
        list(APPEND PROTO_SRC ${GENERATED_PROTOBUF_PATH}/cogment/api/${PROTO}.pb.cc)
endmacro()

macro(compile_grpc_proto PROTO)
        add_custom_command(
                OUTPUT  "${GENERATED_PROTOBUF_PATH}/cogment/api/${PROTO}.egrpc.pb.h"
                        "${GENERATED_PROTOBUF_PATH}/cogment/api/${PROTO}.egrpc.pb.cc"
                        "${GENERATED_PROTOBUF_PATH}/cogment/api/${PROTO}.pb.h"
                        "${GENERATED_PROTOBUF_PATH}/cogment/api/${PROTO}.pb.cc"
                COMMAND ${Protobuf_PROTOC_EXECUTABLE}
                ARGS
                "--proto_path=${COGMENT_API_DIR}/"
                "--egrpc_out=${GENERATED_PROTOBUF_PATH}"
                "--cpp_out=${GENERATED_PROTOBUF_PATH}"
                "--plugin=protoc-gen-egrpc=$<TARGET_FILE:easy_grpc_protoc_plugin>"
                "${COGMENT_API_DIR}/cogment/api/${PROTO}.proto"
                MAIN_DEPENDENCY ${COGMENT_API_DIR}/cogment/api/${PROTO}.proto
                )
        list(APPEND COGMENT_API_SRC ${COGMENT_API_DIR}/cogment/api/${PROTO}.proto)
        list(APPEND PROTO_SRC ${GENERATED_PROTOBUF_PATH}/cogment/api/${PROTO}.egrpc.pb.cc)
        list(APPEND PROTO_SRC ${GENERATED_PROTOBUF_PATH}/cogment/api/${PROTO}.pb.cc)
endmacro()

compile_proto(common)
compile_proto(message)
compile_grpc_proto(data)
compile_grpc_proto(orchestrator)
compile_grpc_proto(agent)
compile_grpc_proto(environment)
compile_grpc_proto(hooks)
# Last one to be 'called' because ${COGMENT_API_SRC} is constructed by the previous calls.
retrieve_protos()

add_library(orchestrator_protos
  ${PROTO_SRC}
)

target_include_directories(orchestrator_protos
PUBLIC
    ${GENERATED_PROTOBUF_PATH}
)

add_library(orchestrator_lib
  cogment/actor.cpp
  cogment/agent_actor.cpp
  cogment/base64.cpp
  #cogment/config_file.cpp
  #cogment/human.cpp
  cogment/orchestrator.cpp
  cogment/trial_params.cpp
  cogment/trial_spec.cpp
  cogment/trial.cpp

  cogment/datalog/storage_interface.cpp
  cogment/datalog/grpc_exporter.cpp

  cogment/services/actor_service.cpp
  cogment/services/trial_lifecycle_service.cpp
)

IF(CLANG_TIDY)
  set_target_properties(orchestrator_lib PROPERTIES CXX_CLANG_TIDY "clang-tidy;--format-style=file")
ENDIF()

target_link_libraries(orchestrator_lib
    orchestrator_protos
    spdlog::spdlog
    slt_settings
    slt_concur
    uuid.a
    easy_grpc
    easy_grpc_reflection
    yaml-cpp
    grpc.a
    gpr.a
    protobuf::libprotobuf
    Threads::Threads
)

target_compile_options(orchestrator_lib PUBLIC -Wno-unused-parameter)
target_include_directories(orchestrator_lib
PUBLIC
    .
    ../third_party)
