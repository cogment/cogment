#!/usr/bin/env bash

# This script is meant to be packaged in the orchestrator's docker image.
# It launches the orchestrator alongside the grpcweb proxy if requested.

# Usage:
#   launch_orchestrator [options...]
#
#   options are simply forwarded to the orchestrator invocation
# Environment variables:
#   COGMENT_WEB_PROXY_PORT=<port>
#     Setting this will launch a grpc web proxy listening on <port>
#     and forwarding to the orchestrator's actor port
#   COGMENT_ORCHESTRATOR_VARIANT=<variant>
#     Use an alternate orchestrator executable. (e.g. debug)

set -e

cleanup() {
  rm -f "${STATUS_PIPE}"

  if [[ -n "${PROXY_PID}" ]]; then
    kill "${PROXY_PID}"
    PROXY_PID=""
  fi
}

# Notes:
# - COGMENT_ACTOR_PORT is the name of the environment variable
#   expected by the orchestrator
# - 9000 is the default value used by the orchestrator
if [[ -z "${COGMENT_ACTOR_PORT}" ]]; then
  COGMENT_ACTOR_PORT=9000
fi

ORCHESTRATOR_ARGS=("$@")
while [[ $# -gt 0 ]]; do
  case "$1" in
    # --actor_port=N takes precedence over COGMENT_ACTOR_PORT, and we pass
    # that value to the proxy.
    --actor_port=*)
      COGMENT_ACTOR_PORT="${1#*=}"
      ;;
    # There can only be a single status file and this script uses it.
    --status_file=*)
      printf "Cannot use a status file with the orchestrator launch script.\n"
      exit 1
      ;;
    *) ;;
  esac
  shift
done

PROXY_PID=""
STATUS_PIPE=$(mktemp -t cogment_orchestrator_status.XXXXXX)
rm "${STATUS_PIPE}"

trap cleanup EXIT

# Because the various commands can cause a fair bit of tty spam, we build the
# commands and report them all before launching the first one.

# Prepare the orchestrator launch command.
mkfifo "${STATUS_PIPE}"

ORCHESTRATOR_PROGRAM="orchestrator"
if [[ -n "${COGMENT_ORCHESTRATOR_VARIANT}" ]]; then
  ORCHESTRATOR_PROGRAM="${ORCHESTRATOR_PROGRAM}_${COGMENT_ORCHESTRATOR_VARIANT}"
fi

# Prepare the proxy launch command, if appropriate.
if [[ -n "${COGMENT_WEB_PROXY_PORT}" ]]; then
  # Apply the default value if needed

  PROXY_LAUNCH_CMD="grpcwebproxy --backend_addr=localhost:${COGMENT_ACTOR_PORT} --run_tls_server=false --allow_all_origins --use_websockets --server_http_debug_port=${COGMENT_WEB_PROXY_PORT}"
else
  PROXY_LAUNCH_CMD=""
fi

printf "Orchestrator launch script will execute the following commands:\n"
printf "  %s --status_file=%s %s\n" "${ORCHESTRATOR_PROGRAM}" "${STATUS_PIPE}" "${ORCHESTRATOR_ARGS[*]}"
if [[ -n "${PROXY_LAUNCH_CMD}" ]]; then
  printf "  %s\n" "${PROXY_LAUNCH_CMD}"
fi
printf "\n"
${ORCHESTRATOR_PROGRAM} --status_file="${STATUS_PIPE}" "${ORCHESTRATOR_ARGS[@]}" &

# Wait for the orchestrator to be done Initializing and be Running
read -r -n2 STATUS <"${STATUS_PIPE}"
if [[ "${STATUS}" != 'IR' ]]; then
  printf "Orchestrator failed to start.\n"
  exit 1
fi

if [[ -n "${PROXY_LAUNCH_CMD}" ]]; then
  ${PROXY_LAUNCH_CMD} &
  PROXY_PID=$!
fi

# Wait for the orchestrator to terminate
read -r -n1 STATUS <"${STATUS_PIPE}"

# Just to be absolutely safe
if [[ "${STATUS}" != 'T' ]]; then
  printf "Warning: Orchestrator did not terminate cleanly.\n"
  exit 1
fi
