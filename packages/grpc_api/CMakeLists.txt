set(COGMENT_API_FILES
  cogment/api/agent.proto
  cogment/api/common.proto
  cogment/api/datalog.proto
  cogment/api/directory.proto
  cogment/api/environment.proto
  cogment/api/health.proto
  cogment/api/hooks.proto
  cogment/api/model_registry.proto
  cogment/api/orchestrator.proto
  cogment/api/trial_datastore.proto
)

install(
  FILES ${COGMENT_API_FILES}
  DESTINATION include/cogment/api/
)
