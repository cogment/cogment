SET(ORCHESTRATOR_SRC
  orchestrator.cpp
)

SET(ORCHESTRATOR_HEADERS
  callback_spdlog_sink.h
)

SET(ORCHESTRATOR_PUBLIC_HEADERS
  orchestrator.h
)

add_library(orchestrator_dylib SHARED ${ORCHESTRATOR_SRC})
set_property(TARGET orchestrator_dylib PROPERTY OUTPUT_NAME orchestrator)

target_link_libraries(orchestrator_dylib
    debug -rdynamic
    orchestrator_lib
    prometheus-cpp::pull
    grpc
)

target_include_directories(orchestrator_dylib PUBLIC . ../lib)

install(TARGETS orchestrator_dylib)
install(FILES ${ORCHESTRATOR_PUBLIC_HEADERS} DESTINATION include)
