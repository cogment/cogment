#!/usr/bin/env bash

####
# Release finalization script
#
# Should be executed by the release manager to finalize a release and re-init the develop branch
####

# shellcheck disable=SC1091
source "$(dirname "${BASH_SOURCE[0]}")/commons.sh"

function usage() {
  local usage_str=""
  usage_str+="Tag a release: finalize the changelog, merge the release branch, tag the release and rebase the develop branch\n\n"
  usage_str+="Usage:\n"
  usage_str+="  $(basename "${BASH_SOURCE[0]}") <version> [--dry-run]\n\n"
  usage_str+="  version: looks like MAJOR.MINOR.PATCH[.PRERELEASE] having:\n"
  usage_str+="    - MAJOR, MINOR and PATCH digit only,\n"
  usage_str+="    - PRERELEASE optional, any string >1 alphanumerical characters.\n\n"
  usage_str+="Options:\n"
  usage_str+="  --dry-run:                      Do not push anything to git.\n"
  usage_str+="  -h, --help:                     Show this screen.\n"
  printf "%b" "${usage_str}"
}

# Parse the commande line arguments.
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

printf "* Finalizing release v%s...\n" "${version}"

# Move to the remote release branch
release_branch="release/v${version}"
git -C "${ROOT_DIR}" fetch -q "${GIT_REMOTE}"
git -C "${ROOT_DIR}" checkout -q -B "${release_branch}" "${GIT_REMOTE}/${release_branch}"

printf "** Now on the latest commit for release branch \"%s/%s\"\n" "${GIT_REMOTE}" "${release_branch}"

# Check the package version
package_version=$(retrieve_package_version)
if [[ "${version}" != "${package_version}" ]]; then
  printf "Package version, %s, doesn't match.\n" "${version}"
  exit 1
fi

printf "** Package version checked\n"

# Update the changelog and commit
changelog_md_file="${ROOT_DIR}/CHANGELOG.md"
today=$(date +%Y-%m-%d)
sed -i.bak "s/.*##\ Unreleased.*/## Unreleased${SED_NL}${SED_NL}## v${version} - ${today}/g" "${changelog_md_file}"
git -C "${ROOT_DIR}" commit -q -a -m"Finalizing release v${version}"

printf "** \"%s\" updated and committed\n" "${changelog_md_file}"

# Move to the remote main branch
git -C "${ROOT_DIR}" checkout -q -B main "${GIT_REMOTE}"/main

printf "** Now on the latest commit for main branch \"%s/main\"\n" "${GIT_REMOTE}"

# Fast forward merge!
git -C "${ROOT_DIR}" merge -q --ff-only "${release_branch}"

printf "** \"%s\" fast forward merged in \"main\"\n" "${GIT_REMOTE}"

# Tag !
git -C "${ROOT_DIR}" tag -a "v${version}" -m "v${version}"

# Push main and the tag to the remote
if [[ "${dry_run}" == 1 ]]; then
  printf "** DRY RUN SUCCESSFUL - Skipping pushing \"main\" branch and tags to %s \n" "${GIT_REMOTE}"
else
  git -C "${ROOT_DIR}" push -q --tags "${GIT_REMOTE}" main
  printf "** \"main\" branch pushed to \"%s\" \n" "${GIT_REMOTE}"
fi

# Move to the remote develop branch
git -C "${ROOT_DIR}" checkout -q -B develop "${GIT_REMOTE}"/develop

printf "** Now on the latest commit for develop branch \"%s/develop\"\n" "${GIT_REMOTE}"

# Rebase develop on the latest merge
git -C "${ROOT_DIR}" rebase -q main

printf "** \"develop\" rebased on the just release \"main\"\n"

# (Force) push develop to the remote
if [[ "${dry_run}" == 1 ]]; then
  printf "** DRY RUN SUCCESSFUL - Skipping (force) pushing \"develop\" branch to %s \n" "${GIT_REMOTE}"
else
  git -C "${ROOT_DIR}" push -q -f "${GIT_REMOTE}" develop
  printf "** \"develop\" branch (force) pushed to \"%s\" \n" "${GIT_REMOTE}"
fi
