set(COGMENT_API_CONFIG .cogment-api.yaml)

set(COGMENT_API_DIR ${CMAKE_CURRENT_BINARY_DIR}/cogment-api CACHE INTERNAL "")

set(COGMENT_API_FILES
  ${COGMENT_API_DIR}/cogment/api/agent.proto
  ${COGMENT_API_DIR}/cogment/api/common.proto
  ${COGMENT_API_DIR}/cogment/api/datalog.proto
  ${COGMENT_API_DIR}/cogment/api/directory.proto
  ${COGMENT_API_DIR}/cogment/api/environment.proto
  ${COGMENT_API_DIR}/cogment/api/health.proto
  ${COGMENT_API_DIR}/cogment/api/hooks.proto
  ${COGMENT_API_DIR}/cogment/api/model_registry.proto
  ${COGMENT_API_DIR}/cogment/api/orchestrator.proto
  ${COGMENT_API_DIR}/cogment/api/trial_datastore.proto
)

file(MAKE_DIRECTORY ${COGMENT_API_DIR}/cogment/api/)

file(READ ${COGMENT_API_CONFIG} COGMENT_API_VERSION_YML)
string(REGEX MATCH "cogment_api_path: \"?(.+)\"?\n" COGMENT_API_PATH_MATCH ${COGMENT_API_VERSION_YML})
set(COGMENT_API_PATH ${CMAKE_MATCH_1})
if(COGMENT_API_PATH)
  # Local cogment api
  set(COGMENT_API_VERSION "local" CACHE INTERNAL "")
  message("using local cogment api at \"${COGMENT_API_PATH}\"")
  add_custom_target(api_retrieve
    COMMAND ${CMAKE_COMMAND} -E copy_directory ${COGMENT_API_PATH} ${COGMENT_API_DIR}/cogment/api
    SOURCES ${COGMENT_API_CONFIG}
    BYPRODUCTS ${COGMENT_API_FILES}
    COMMENT "Copying cogment api from \"${COGMENT_API_PATH}\""
    VERBATIM
  )
else()
  # Remote cogment api
  string(REGEX MATCH "cogment_api_version: \"?([a-zA-Z0-9\.-]+)\"?\n" COGMENT_API_VERSION_MATCH ${COGMENT_API_VERSION_YML})
  set(COGMENT_API_VERSION ${CMAKE_MATCH_1} CACHE INTERNAL "")
  set(COGMENT_API_ARCHIVE_PATH ${CMAKE_CURRENT_BINARY_DIR}/cogment-api-${COGMENT_API_VERSION}.tar.gz)
  set(COGMENT_API_URL "https://cogment.github.io/cogment-api/${COGMENT_API_VERSION}/cogment-api-${COGMENT_API_VERSION}.tar.gz")

  message("fetching cogment api version \"${COGMENT_API_VERSION}\" from \"${COGMENT_API_URL}\"")
  file(
    DOWNLOAD  ${COGMENT_API_URL}
              ${COGMENT_API_ARCHIVE_PATH}
    SHOW_PROGRESS
    TLS_VERIFY ON
  )
  add_custom_target(api_retrieve
    COMMAND ${CMAKE_COMMAND} -E tar xzf ${COGMENT_API_ARCHIVE_PATH}
    WORKING_DIRECTORY ${COGMENT_API_DIR}/cogment/api
    SOURCES ${COGMENT_API_CONFIG}
    BYPRODUCTS ${COGMENT_API_FILES}
    COMMENT "Unpacking the cogment-api archive, version \"${COGMENT_API_VERSION}\""
    VERBATIM
  )
endif()

install(
  FILES ${COGMENT_API_FILES}
  DESTINATION include/cogment/api/
)
