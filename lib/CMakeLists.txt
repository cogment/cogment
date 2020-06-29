find_package(Protobuf REQUIRED)
find_package(yaml-cpp REQUIRED)

set(THREADS_PREFER_PTHREAD_FLAG ON)
find_package(Threads REQUIRED)

file(READ version.txt COGMENT_ORCH_VERSION)

configure_file("${CMAKE_CURRENT_SOURCE_DIR}/aom/orch_config.h.in"
               "${CMAKE_CURRENT_SOURCE_DIR}/aom/orch_config.h" )

set(GENERATED_PROTOBUF_PATH ${CMAKE_CURRENT_BINARY_DIR}/generated)
file(MAKE_DIRECTORY ${GENERATED_PROTOBUF_PATH})

SET(PROTO_SRC "")
macro(compile_proto PROTO)
        add_custom_command(
                OUTPUT  "${GENERATED_PROTOBUF_PATH}/cogment/api/${PROTO}.pb.h"
                        "${GENERATED_PROTOBUF_PATH}/cogment/api/${PROTO}.pb.cc"
                COMMAND ${Protobuf_PROTOC_EXECUTABLE}
                ARGS 
                "--proto_path=${CMAKE_CURRENT_SOURCE_DIR}/../../api/"
                "--cpp_out=${GENERATED_PROTOBUF_PATH}"
                "${CMAKE_CURRENT_SOURCE_DIR}/../../api/cogment/api/${PROTO}.proto"
                MAIN_DEPENDENCY ${CMAKE_CURRENT_SOURCE_DIR}/../../api/cogment/api/${PROTO}.proto
          )

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
                "--proto_path=${CMAKE_CURRENT_SOURCE_DIR}/../../api/"
                "--egrpc_out=${GENERATED_PROTOBUF_PATH}"
                "--cpp_out=${GENERATED_PROTOBUF_PATH}"
                "--plugin=protoc-gen-egrpc=$<TARGET_FILE:easy_grpc_protoc_plugin>"
                "${CMAKE_CURRENT_SOURCE_DIR}/../../api/cogment/api/${PROTO}.proto"
                MAIN_DEPENDENCY ${CMAKE_CURRENT_SOURCE_DIR}/../../api/cogment/api/${PROTO}.proto
                )
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


add_library(orchestrator_lib 
  #aom/actor.cpp
  #aom/agent.cpp
  aom/base64.cpp
  #aom/config_file.cpp
  #aom/human.cpp
  aom/orchestrator.cpp
  aom/trial_params.cpp
  aom/trial_spec.cpp
  aom/trial.cpp

  aom/datalog/storage_interface.cpp
  aom/datalog/grpc_exporter.cpp

  aom/services/actor_service.cpp
  aom/services/trial_lifecycle_service.cpp

  ${PROTO_SRC}
)

target_link_libraries(orchestrator_lib
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
    ${GENERATED_PROTOBUF_PATH}
    ../third_party)
