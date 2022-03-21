#!/usr/bin/env bash

####
# Version changes extraction
#
# Extract the changes for a given version from the changelog file
####

# shellcheck disable=SC1091
source "$(dirname "${BASH_SOURCE[0]}")/commons.sh"

changelog="${ROOT_DIR}/CHANGELOG.md"

function usage() {
  local usage_str=""
  usage_str+="Extract the changes for a given version\n\n"
  usage_str+="Usage:\n"
  usage_str+="  $(basename "${BASH_SOURCE[0]}") <version> [--changelog=PATH_TO_CHANGELOG_MD_FILE]\n\n"
  usage_str+="  version: looks like MAJOR.MINOR.PATCH[.PRERELEASE] having:\n"
  usage_str+="    - MAJOR, MINOR and PATCH digit only,\n"
  usage_str+="    - PRERELEASE optional, any string >1 alphanumerical characters.\n\n"
  usage_str+="Options:\n"
  usage_str+="  --changelog=PATH:                       Source changelog file (default is \"${changelog}\").\n"
  usage_str+="  -h, --help:                             Show this screen.\n"
  printf "%b" "${usage_str}"
}

# Parse the commande line arguments.
while [[ "$1" != "" ]]; do
  case $1 in
    --changelog)
      shift
      changelog="$(cd "$(dirname "$1")" && pwd)/$(basename "$1")"
      ;;
    --help | -h)
      usage
      exit 0
      ;;
    *)
      if [[ -z "${version}" ]]; then
        input_version=$1
        validated_version=$(validate_version "${input_version}")
        if [[ -z "${validated_version}" ]]; then
          printf "%s: provided version is invalid.\n\n" "${input_version}"
          usage
          exit 1
        fi
        version="${validated_version}"
      else
        printf "%s: unrecognized argument.\n\n" "$1"
        usage
        exit 1
      fi
      ;;
  esac
  shift
done

if [[ -z "${version}" ]]; then
  printf "Missing version.\n\n"
  usage
  exit 1
fi

# Inspired by https://stackoverflow.com/a/40450360/454743
awk -v "ver=v${version}" '
 /^## v/ { if (p) { exit }; if ($2 == ver) { p=1 } } p
' "${changelog}"
