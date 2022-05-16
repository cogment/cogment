configure_file("${CMAKE_CURRENT_SOURCE_DIR}/version/version.go.in"
               "${CMAKE_CURRENT_SOURCE_DIR}/version/version.go" )

file(MAKE_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}/services/orchestrator/lib)
file(TOUCH ${CMAKE_CURRENT_SOURCE_DIR}/services/orchestrator/lib/liborchestrator.dylib)
file(TOUCH ${CMAKE_CURRENT_SOURCE_DIR}/services/orchestrator/lib/liborchestrator.so)
file(TOUCH ${CMAKE_CURRENT_SOURCE_DIR}/services/orchestrator/lib/orchestrator.dll)

if(COGMENT_EMBEDS_ORCHESTRATOR)
  add_custom_target(cogment_code_generation
  DEPENDS $<TARGET_FILE:orchestrator_dylib>
  COMMAND ${CMAKE_COMMAND} -E copy $<TARGET_FILE:orchestrator_dylib> ./services/orchestrator/lib
  COMMAND ${CMAKE_COMMAND} -E env COGMENT_API_DIR=${COGMENT_API_DIR} PROTOC_EXEC=$<TARGET_FILE:protobuf::protoc> PROTOBUF_INCLUDE_DIR=${PROTOBUF_INCLUDE_DIR} go generate -tags tools
  WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
  COMMENT "Running the code generation phase"
  VERBATIM)
else()
  add_custom_target(cogment_code_generation
  COMMAND ${CMAKE_COMMAND} -E env COGMENT_API_DIR=${COGMENT_API_DIR} go generate -tags tools
  WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
  COMMENT "Running the code generation phase"
  VERBATIM)
endif()

set(COGMENT_EXEC_RELPATH "build/cogment")
if(${COGMENT_OS} MATCHES "windows")
  set(COGMENT_EXEC_RELPATH "build/cogment.exe")
endif()

add_custom_target(cogment_exec ALL
  COMMAND ${CMAKE_COMMAND} -E env CGO_ENABLED=$<BOOL:${COGMENT_EMBEDS_ORCHESTRATOR}> GOOS=${COGMENT_OS} GOARCH=${COGMENT_ARCH} go build -ldflags "-s -w" -o ${COGMENT_EXEC_RELPATH}
  BYPRODUCTS ${CMAKE_CURRENT_SOURCE_DIR}/${COGMENT_EXEC_RELPATH}
  WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
  VERBATIM)
add_dependencies(cogment_exec cogment_code_generation)

install(PROGRAMS ${CMAKE_CURRENT_SOURCE_DIR}/${COGMENT_EXEC_RELPATH} TYPE BIN)

add_custom_target(cogment_lint_fix
  COMMAND golangci-lint run --fix
  COMMENT "Running golangci-lint to fix the format of 'cogment' files"
  WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
  VERBATIM
)
add_dependencies(cogment_lint_fix cogment_code_generation)

add_custom_target(cli_lint
  COMMAND golangci-lint run
  COMMENT "Running golangci-lint to check the format of 'cogment' files"
  WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
  VERBATIM
)
add_dependencies(cli_lint cogment_code_generation)

add_custom_target(cli_test
  COMMAND go test -v ./...
  COMMENT "Running 'cogment' tests"
  WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
  VERBATIM
)
add_dependencies(cli_test cogment_code_generation)

add_custom_target(cogment_test_with_junit_report
  COMMAND rm -f test_failed.txt
  COMMAND go test -v ./... 2>&1 > raw_report.txt || echo test_failed > test_failed.txt
  COMMAND go run github.com/jstemmer/go-junit-report < raw_report.txt > report.xml
  COMMAND test ! -f test_failed.txt
  COMMENT "Running 'cogment' tests"
  WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
  VERBATIM
)
add_dependencies(cogment_test_with_junit_report cogment_code_generation)

add_custom_target(cli_benchmark
  COMMAND go test ./... -run xxx -bench . -test.benchtime 60s
  COMMENT "Running 'cogment' benchmarks"
  WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
  VERBATIM
)
add_dependencies(cli_test cogment_code_generation)
