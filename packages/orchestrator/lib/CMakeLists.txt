set(THREADS_PREFER_PTHREAD_FLAG ON)
find_package(Threads REQUIRED)

set(GENERATED_PROTOBUF_PATH ${CMAKE_CURRENT_BINARY_DIR}/generated)
file(MAKE_DIRECTORY ${GENERATED_PROTOBUF_PATH})

set(PROTO_SRC "")

macro(compile_proto PROTO)
        add_custom_command(
                OUTPUT  "${GENERATED_PROTOBUF_PATH}/cogment/api/${PROTO}.pb.h"
                        "${GENERATED_PROTOBUF_PATH}/cogment/api/${PROTO}.pb.cc"
                COMMAND protobuf::protoc
                ARGS
                "--proto_path=${PROTOBUF_INCLUDE_DIR}"
                "--proto_path=${COGMENT_API_DIR}/"
                "--cpp_out=${GENERATED_PROTOBUF_PATH}"
                "${COGMENT_API_DIR}/cogment/api/${PROTO}.proto"
          )

        list(APPEND PROTO_SRC ${GENERATED_PROTOBUF_PATH}/cogment/api/${PROTO}.pb.cc)
endmacro()

macro(compile_grpc_proto PROTO)
        add_custom_command(
                OUTPUT  "${GENERATED_PROTOBUF_PATH}/cogment/api/${PROTO}.grpc.pb.h"
                        "${GENERATED_PROTOBUF_PATH}/cogment/api/${PROTO}.grpc.pb.cc"
                        "${GENERATED_PROTOBUF_PATH}/cogment/api/${PROTO}.pb.h"
                        "${GENERATED_PROTOBUF_PATH}/cogment/api/${PROTO}.pb.cc"
                COMMAND protobuf::protoc
                ARGS
                "--proto_path=${PROTOBUF_INCLUDE_DIR}"
                "--proto_path=${COGMENT_API_DIR}/"
                "--grpc_out=${GENERATED_PROTOBUF_PATH}"
                "--cpp_out=${GENERATED_PROTOBUF_PATH}"
                "--plugin=protoc-gen-grpc=$<TARGET_FILE:grpc_cpp_plugin>"
                "${COGMENT_API_DIR}/cogment/api/${PROTO}.proto"
                )
        list(APPEND PROTO_SRC ${GENERATED_PROTOBUF_PATH}/cogment/api/${PROTO}.grpc.pb.cc)
        list(APPEND PROTO_SRC ${GENERATED_PROTOBUF_PATH}/cogment/api/${PROTO}.pb.cc)
endmacro()

compile_proto(common)
compile_grpc_proto(datalog)
compile_grpc_proto(orchestrator)
compile_grpc_proto(agent)
compile_grpc_proto(environment)
compile_grpc_proto(hooks)
compile_grpc_proto(directory)

configure_file("${CMAKE_CURRENT_SOURCE_DIR}/cogment/versions.h.in"
               "${CMAKE_CURRENT_SOURCE_DIR}/cogment/versions.h" )

add_library(orchestrator_protos
  ${PROTO_SRC}
)

add_dependencies(orchestrator_protos api_retrieve)

target_link_libraries(orchestrator_protos
PUBLIC
    grpc++
)

target_include_directories(orchestrator_protos
PUBLIC
    ${GENERATED_PROTOBUF_PATH}
)

add_library(orchestrator_lib
  cogment/actor.cpp
  cogment/agent_actor.cpp
  cogment/client_actor.cpp
  cogment/datalog.cpp
  cogment/orchestrator.cpp
  cogment/trial_params.cpp
  cogment/trial.cpp
  cogment/utils.cpp
  cogment/directory.cpp
  cogment/environment.cpp

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
    yaml-cpp
    grpc++
    grpc
    gpr
    protobuf::libprotobuf
    prometheus-cpp::pull
    Threads::Threads
)

if(UNIX)
  target_compile_options(orchestrator_lib PUBLIC -Wno-unused-parameter)
endif()

target_include_directories(orchestrator_lib
PUBLIC
    .
    ../third_party ${YAML_CPP_INCLUDE_DIR}
    )
