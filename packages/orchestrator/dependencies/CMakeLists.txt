include(FetchContent)

set(FETCHCONTENT_QUIET OFF)

FetchContent_Declare(
  grpc
  GIT_REPOSITORY "https://github.com/grpc/grpc"
  GIT_TAG v1.41.1
)

FetchContent_Declare(
  yaml-cpp
  GIT_REPOSITORY "https://github.com/jbeder/yaml-cpp"
  GIT_TAG yaml-cpp-0.7.0
)

FetchContent_Declare(
  spdlog
  GIT_REPOSITORY "https://github.com/gabime/spdlog"
  GIT_TAG v1.x
)

FetchContent_Declare(
  prometheus-cpp
  GIT_REPOSITORY "https://github.com/jupp0r/prometheus-cpp"
  GIT_TAG v0.6.0
)

# build gRPC
FetchContent_GetProperties(grpc)
if(NOT grpc_POPULATED)
  FetchContent_Populate(grpc)

  set(gRPC_INSTALL ON CACHE BOOL "" FORCE)
  set(gRPC_BUILD_TESTS OFF CACHE BOOL "" FORCE)
  set(gRPC_BUILD_CODEGEN ON CACHE BOOL "" FORCE) # for grpc_cpp_plugin
  set(gRPC_BUILD_GRPC_CPP_PLUGIN ON CACHE BOOL "" FORCE) # we want to use only C++ plugin
  set(gRPC_BUILD_CSHARP_EXT OFF CACHE BOOL "" FORCE)
  set(gRPC_BUILD_GRPC_CSHARP_PLUGIN OFF CACHE BOOL "" FORCE)
  set(gRPC_BUILD_GRPC_NODE_PLUGIN OFF CACHE BOOL "" FORCE)
  set(gRPC_BUILD_GRPC_OBJECTIVE_C_PLUGIN OFF CACHE BOOL "" FORCE)
  set(gRPC_BUILD_GRPC_PHP_PLUGIN OFF CACHE BOOL "" FORCE)
  set(gRPC_BUILD_GRPC_PYTHON_PLUGIN OFF CACHE BOOL "" FORCE)
  set(gRPC_BUILD_GRPC_RUBY_PLUGIN OFF CACHE BOOL "" FORCE)

  add_subdirectory(${grpc_SOURCE_DIR} ${grpc_BINARY_DIR} EXCLUDE_FROM_ALL)

  set(PROTOBUF_INCLUDE_DIR ${grpc_SOURCE_DIR}/third_party/protobuf/src CACHE INTERNAL "")
endif()

# build yaml-cpp
FetchContent_GetProperties(yaml-cpp)
if(NOT yaml-cpp_POPULATED)
  FetchContent_Populate(yaml-cpp)

  set(YAML_CPP_BUILD_TESTS OFF CACHE BOOL "" FORCE)
  set(YAML_CPP_BUILD_TOOLS OFF CACHE BOOL "" FORCE)

  add_subdirectory(${yaml-cpp_SOURCE_DIR} ${yaml-cpp_BINARY_DIR} EXCLUDE_FROM_ALL)

  set(YAML_CPP_INCLUDE_DIR ${yaml-cpp_SOURCE_DIR}/include PARENT_SCOPE)
endif()

# build spdlog
FetchContent_GetProperties(spdlog)
if(NOT spdlog_POPULATED)
  FetchContent_Populate(spdlog)

  set(SPDLOG_BUILD_TESTS OFF CACHE BOOL "" FORCE)

  add_subdirectory(${spdlog_SOURCE_DIR} ${spdlog_BINARY_DIR} EXCLUDE_FROM_ALL)
endif()


# build prometheus-cpp
FetchContent_GetProperties(prometheus-cpp)
if(NOT prometheus-cpp_POPULATED)
  FetchContent_Populate(prometheus-cpp)

  set(ENABLE_TESTING OFF CACHE BOOL "" FORCE)
  set(ENABLE_COMPRESSION OFF CACHE BOOL "" FORCE)
  set(ENABLE_PUSH OFF CACHE BOOL "" FORCE)

  add_subdirectory(${prometheus-cpp_SOURCE_DIR} ${prometheus-cpp_BINARY_DIR} EXCLUDE_FROM_ALL)
endif()



