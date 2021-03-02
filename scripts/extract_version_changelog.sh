#!/usr/bin/env bash

CHANGELOG_PATH="$(dirname "${BASH_SOURCE[0]}")/../CHANGELOG.md"
VERSION=v1.0.0-alpha2

# Let's check if we have GNU sed or BSD sed
if sed --help >/dev/null 2>&1; then
  # There is a '--help' option, it is GNU BSD
  SED_NL="\\n"
else
  SED_NL="\\
"
fi

sed -n "s/##\ ${VERSION}${SED_NL}\(\(.*${SED_NL}\)*\)/\1/gp" "${CHANGELOG_PATH}"
