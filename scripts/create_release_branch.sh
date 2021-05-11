#!/usr/bin/env bash

####
# Release preparation script
#
# Should be executed by the release manager to initiate the finalization work on a particular release
####

# shellcheck disable=SC1091
source "$(dirname "${BASH_SOURCE[0]}")/commons.sh"

function usage() {
  local usage_str=""
  usage_str+="Update the version and create a release branch\n\n"
  usage_str+="Usage:\n"
  usage_str+="  $(basename "${BASH_SOURCE[0]}") <version> [--dry-run]\n\n"
  usage_str+="  version: looks like MAJOR.MINOR.PATCH[.PRERELEASE] having:\n"
  usage_str+="    - MAJOR, MINOR and PATCH digit only,\n"
  usage_str+="    - PRERELEASE optional, any string >1 alphanumerical characters.\n\n"
  usage_str+="Options:\n"
  usage_str+="  --dry-run:                              Do not push anything to the remote.\n"
  usage_str+="  -h, --help:                             Show this screen.\n"
  printf "%b" "${usage_str}"
}

set -o errexit

# Parse the command line arguments.
dry_run=0

while [[ "$1" != "" ]]; do
  case $1 in
    --dry-run)
      dry_run=1
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

printf "* Preparing release v%s...\n" "${version}"

# Move to the remote `develop` branch
git -C "${ROOT_DIR}" fetch -q "${GIT_REMOTE}"
git -C "${ROOT_DIR}" checkout -q "${GIT_REMOTE}/develop"

printf "** Now on the latest %s/develop\n" "${GIT_REMOTE}"

# Retrieving the current version of the package
current_version=$(retrieve_package_version)

printf "** Current version is v%s\n" "${current_version}"

# Creating the release branch, this will fail if the branch already exists
release_branch="release/v${version}"
git -C "${ROOT_DIR}" branch "${release_branch}"
git -C "${ROOT_DIR}" checkout -q "${release_branch}"

printf "** Release branch \"%s\" created\n" "${release_branch}"

# Updating the version of the package and commit
updated_version=$(update_package_version "${version}")
if [[ "${version}" != "${updated_version}" ]]; then
  printf "Error while updating the version to %s.\n" "${version}"
  exit 1
fi
git -C "${ROOT_DIR}" commit -q -a -m"chore(release): prepare release v${version}"

# TODO here we could ask / retrieve the latest versions of the different modules and update /api/default_cogment.yaml

printf "** Version updated to v%s\n" "${updated_version}"

if [[ "${dry_run}" == 1 ]]; then
  printf "** DRY RUN SUCCESSFUL - Nothing pushed to %s\n" "${GIT_REMOTE}"
else
  git -C "${ROOT_DIR}" push -q "${GIT_REMOTE}" "${release_branch}"
  printf "** Release branch \"%s\" pushed to \"%s\" \n" "${release_branch}" "${GIT_REMOTE}"
fi

printf "* To finalize the release:\n"
printf "** Update the dependencies, in particular make sure nothing is relying on a \"latest\" version of another package.\n"
printf "** Check and update the package's changelog at \"%s/CHANGELOG.md\"\n" "${ROOT_DIR}"
printf "** Make sure the CI builds everything properly\n"
printf "** Finally, run \"%s %s\"\n" "$(dirname "${BASH_SOURCE[0]}")/tag_release.sh" "${version}"
