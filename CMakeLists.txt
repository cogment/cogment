project(orchestrator)

find_package(prometheus-cpp REQUIRED)
find_package(spdlog REQUIRED)

add_subdirectory(third_party/slt_concur)
add_subdirectory(third_party/slt_setting)

add_subdirectory(lib)

SET(ORCHESTRATOR_SRC
  main.cpp
)

add_executable(orchestrator ${ORCHESTRATOR_SRC})

target_include_directories(orchestrator PRIVATE .)

target_link_libraries(orchestrator
    orchestrator_lib
    prometheus-cpp::pull
)

install(TARGETS orchestrator
        DESTINATION bin)
