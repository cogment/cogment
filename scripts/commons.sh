#!/usr/bin/env bash

####
# Helper functions & variables
####

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
export ROOT_DIR

GIT_REMOTE="origin"
export GIT_REMOTE

# Let's check if we have GNU sed or BSD sed
if sed --help >/dev/null 2>&1; then
  # There is a '--help' option, it is GNU BSD
  SED_NL="\\n"
else
  SED_NL="\\
"
fi
export SED_NL

# Generic functions

## join_by
## Examples:
##  $ join_by "-delimiter-" "a" "b" "c"
##  "a-delimiter-b-delimiter-c"
function join_by() {
  local delimiter=$1
  shift
  local strings=$1
  shift
  printf %s "${strings}" "${@/#/$delimiter}"
}

## array_contains
## Examples:
##  $ array_contains "foo" "bar" "foobaz"
##  1
##
##  $ array_contains "foo" "bar" "foo" "baz"
##  0
function array_contains() {
  local seeking=$1
  shift
  local array=("$@")
  shift
  for element in "${array[@]}"; do
    if [[ "${element}" == "${seeking}" ]]; then
      return 0
    fi
  done
  return 1
}

# Version related functions

VERSION_SED_REGEX="[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*\(-[a-zA-Z0-9][a-zA-Z0-9]*\)\{0,1\}"

function validate_version() {
  local input_version=$1
  shift
  local parsed_version
  parsed_version=$(sed -n "s/^v\{0,1\}\(${VERSION_SED_REGEX}\)$/\1/p" <<<"${input_version}")
  printf %s "${parsed_version}"
}

function retrieve_package_version() {
  cat "${ROOT_DIR}/lib/version.txt"
}

function update_package_version() {
  local version=$1
  sed -i.bak "/.*Version\ *=/s/${VERSION_SED_REGEX}/${version}/" "${ROOT_DIR}/version/version.go"
  retrieve_package_version
}
