cmake_minimum_required(VERSION 3.9)

project(slt_settings VERSION 0.1.0)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

add_library(slt_settings
  src/setting.cpp)

target_include_directories(slt_settings PUBLIC
  $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/./include>
  $<INSTALL_INTERFACE:$<INSTALL_PREFIX>/include>)

if (UNIX)
  target_compile_options(slt_settings PUBLIC -Wall)
  if (NOT ${CMAKE_SYSTEM_NAME} MATCHES "Darwin")
    # Still a bunch of warning on macos
    target_compile_options(slt_settings PUBLIC -Werror)
  endif()
endif()

install(TARGETS slt_settings
  EXPORT   slt_settingsTargets
  ARCHIVE  DESTINATION lib
  LIBRARY  DESTINATION lib
  RUNTIME  DESTINATION bin
)

install(DIRECTORY include/slt DESTINATION include)

#install cmake export
install(EXPORT slt_settingsTargets DESTINATION lib/cmake/slt_settings)

configure_file(cmake/slt_settings-config.cmake.in
  "${PROJECT_BINARY_DIR}/slt_settings-config.cmake" @ONLY)

install(FILES
  "${PROJECT_BINARY_DIR}/slt_settings-config.cmake"
  DESTINATION lib/cmake/slt_settings)

# Samples
add_executable(sample_00 examples/00_hello_setting.cpp)
target_link_libraries(sample_00 slt_settings)

#enable_testing()

# Tests
#add_subdirectory(tests)
