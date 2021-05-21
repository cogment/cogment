cmake_minimum_required(VERSION 3.10)

project(cogment_orchestrator)

set(Protobuf_USE_STATIC_LIBS ON)
find_package(protobuf)
find_package(gRPC)
find_package(easy_grpc)

find_package(prometheus-cpp REQUIRED)
find_package(spdlog REQUIRED)

add_subdirectory(third_party/slt_concur)
add_subdirectory(third_party/slt_setting)

find_program(CLANG_TIDY "clang-tidy")


set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

add_subdirectory(lib)

SET(ORCHESTRATOR_SRC
  main.cpp
)

add_executable(orchestrator ${ORCHESTRATOR_SRC})
IF(CLANG_TIDY)
  set_target_properties(orchestrator PROPERTIES CXX_CLANG_TIDY "clang-tidy;--format-style=file")
ENDIF()

target_include_directories(orchestrator PRIVATE .)

target_link_libraries(orchestrator
    debug -rdynamic
    orchestrator_lib
    prometheus-cpp::pull
    gRPC::grpc
)

set_target_properties(orchestrator PROPERTIES DEBUG_POSTFIX _dbg)

install(TARGETS orchestrator
        DESTINATION bin)

############################ code format ############################
if(NOT DEFINED CLANG_FORMAT_BIN)
  find_program(CLANG_FORMAT_BIN NAMES clang-format)
endif()

file(GLOB_RECURSE ALL_SOURCE_FILES *.cpp *.h *.cxx *.hxx *.hpp *.cc *.ipp)

set(CLANG_FORMAT_EXCLUDE_PATTERNS ${CLANG_FORMAT_EXCLUDE_PATTERNS} 
  "/CMakeFiles/" 
  "cmake"
  "third_party"
)

foreach (SOURCE_FILE ${ALL_SOURCE_FILES})
    foreach (EXCLUDE_PATTERN ${CLANG_FORMAT_EXCLUDE_PATTERNS})
        string(FIND ${SOURCE_FILE} ${EXCLUDE_PATTERN} EXCLUDE_FOUND)
        if (NOT ${EXCLUDE_FOUND} EQUAL -1)
            list(REMOVE_ITEM ALL_SOURCE_FILES ${SOURCE_FILE})
        endif ()
    endforeach ()
endforeach ()


add_custom_target(format
    COMMENT "Running clang-format to change files"
    COMMAND ${CLANG_FORMAT_BIN}
    -style=file
    -i
    ${ALL_SOURCE_FILES}
)
#############################################################

