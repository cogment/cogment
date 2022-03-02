cmake_minimum_required(VERSION 3.9)

project(slt_settings VERSION 0.1.0)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

add_library(slt_concur
  src/work_pool.cpp)

target_include_directories(slt_concur PUBLIC
  $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/./include>
  $<INSTALL_INTERFACE:$<INSTALL_PREFIX>/include>)

if (UNIX)
  target_compile_options(slt_concur PUBLIC -Wall)
  if (NOT ${CMAKE_SYSTEM_NAME} MATCHES "Darwin")
    # Still a bunch of warning on macos
    target_compile_options(slt_concur PUBLIC -Werror)
  endif()
endif()

install(TARGETS slt_concur
  EXPORT   slt_concurTargets
  ARCHIVE  DESTINATION lib
  LIBRARY  DESTINATION lib
  RUNTIME  DESTINATION bin
)

install(DIRECTORY include/slt DESTINATION include)

#install cmake export
install(EXPORT slt_concurTargets DESTINATION lib/cmake/slt_concur)

configure_file(cmake/slt_concur-config.cmake.in
  "${PROJECT_BINARY_DIR}/slt_concur-config.cmake" @ONLY)

install(FILES
  "${PROJECT_BINARY_DIR}/slt_concur-config.cmake"
  DESTINATION lib/cmake/slt_concur)
