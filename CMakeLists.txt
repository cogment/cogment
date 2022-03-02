cmake_minimum_required(VERSION 3.10)

# Enables passing <Library>_ROOT as the directory to find <Library>-config.cmake files
cmake_policy(SET CMP0074 NEW)

set(COGMENT_EMBEDS_ORCHESTRATOR OFF CACHE BOOL "Whether or not the orchestrator should be embedded")
set(COGMENT_OS "" CACHE STRING "Target OS (as a golang os identifier)")
set(COGMENT_ARCH "" CACHE STRING "Target architecture (as a golang architecture identifier)")

if(${CMAKE_SYSTEM_NAME} MATCHES "Darwin")
  # Set the deployment target to the earliest macos version possible
  set(CMAKE_OSX_DEPLOYMENT_TARGET "10.15" CACHE STRING "" FORCE)
  mark_as_advanced(CMAKE_OSX_DEPLOYMENT_TARGET)
endif()

set(CMAKE_POSITION_INDEPENDENT_CODE ON CACHE BOOL "" FORCE)
mark_as_advanced(CMAKE_POSITION_INDEPENDENT_CODE)

project(cogment)

file(READ version.txt COGMENT_VERSION)
string(STRIP "${COGMENT_VERSION}" COGMENT_VERSION)

execute_process(
  COMMAND git log -1 --format=%h
  WORKING_DIRECTORY ${CMAKE_CURRENT_LIST_DIR}
  OUTPUT_VARIABLE COGMENT_GIT_HASH
  OUTPUT_STRIP_TRAILING_WHITESPACE
)

message("Building cogment v${COGMENT_VERSION}@${COGMENT_GIT_HASH}")

add_subdirectory(packages/api)

if(COGMENT_EMBEDS_ORCHESTRATOR)
  add_subdirectory(packages/orchestrator)
else()
  # Required because some other stuffs depends on it
  add_custom_target(orchestrator_dylib COMMENT "Dummy target")
endif()

add_subdirectory(packages/cogment)
