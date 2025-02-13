#!/usr/bin/env bash

# SPDX-FileCopyrightText: 2018-2022 Open Networking Foundation
# SPDX-FileCopyrightText: 2022 Intel Corp.
# SPDX-License-Identifier: Apache-2.0

# version-check.sh
# validates if a version provided in VERSION file is valid

set -eu -o pipefail

# Load the shared library
my_dir="$(dirname "$0")"
# shellcheck source=oie_ci_testing/files/tagging-lib.sh
source "$my_dir/tagging-lib.sh"

# cd to the code checkout location
cd "$WORKSPACE/$BASEDIR"

TAG_PREFIX=${1:-""}

#print TAG_PREFIX
echo "TAG_PREFIX is: ${TAG_PREFIX}"

# check if a previous tag has been created (to avoid going from 2.7.0-dev to 2.7.1-dev)
# this check is skipped for calendar versions as they are not sequential
function is_valid_version {

  if [[ "$CALENDAR_VERSION" == "true" ]]; then
    return
  fi

  local MAJOR=0 MINOR=0 PATCH=0
  local C_MAJOR=0 C_MINOR=0 C_PATCH=0 # these are used in the inner loops to compare

  semverParse "$NEW_VERSION" MAJOR MINOR PATCH

  found_parent=false

  # if minor == 0, check that there was a release with MAJOR-1.X.X
  if [[ "$MINOR" == 0 ]]; then
    prev_major=$(( MAJOR - 1 ))
    parent_version="$prev_major.x.x"
    for existing_tag in $existing_tags
    do
      if [[ $TAG_PREFIX == "" || $existing_tag == "${TAG_PREFIX}"* ]]; then
        existing_tag="${TAG_PREFIX}${existing_tag}"
        semverParse "$existing_tag" C_MAJOR C_MINOR C_PATCH
        if [[ "$prev_major" == "$C_MAJOR" ]]; then
          echo "Found previous tag matching $parent_version: $existing_tag"
          found_parent=true
          break
        fi
      fi
    done
  fi

  # if patch == 0, check that there was a release with MAJOR.MINOR-1.X
  if [[ "$PATCH" == 0 ]]; then
    prev_minor=$(( MINOR - 1 ))
    parent_version="$MAJOR.$prev_minor.x"
    for existing_tag in $existing_tags
    do
      if [[ $TAG_PREFIX == "" || $existing_tag == "${TAG_PREFIX}"* ]]; then
        semverParse "$existing_tag" C_MAJOR C_MINOR C_PATCH
        if [[ "$MAJOR" == "$C_MAJOR" && "$prev_minor" == "$C_MINOR" ]]; then
          echo "Found previous tag matching $parent_version: $existing_tag"
          found_parent=true
          break
        fi
      fi
    done
  fi

  # if patch != 0 check that there was a release with MAJOR.MINOR.PATCH-1
  if [[ "$PATCH" != 0 ]]; then
    prev_patch=$(( PATCH - 1 ))
    parent_version="$MAJOR.$MINOR.$prev_patch"
    for existing_tag in $existing_tags
    do
      if [[ $TAG_PREFIX == "" || $existing_tag == "${TAG_PREFIX}"* ]]; then
        semverParse "$existing_tag" C_MAJOR C_MINOR C_PATCH
        if [[ "$MAJOR" == "$C_MAJOR" && "$MINOR" == "$C_MINOR" && "$prev_patch" == "$C_PATCH" ]]
        then
          echo "Found previous tag matching $parent_version: $existing_tag"
          found_parent=true
          break
        fi
      fi
    done
  fi

  # At the beginning there can be no parent, which is OK
  if [[ "$MAJOR" == 0 ]]; then
    echo "Major version is 0, prerelease, so OK"
    found_parent=true
  fi

  # May be tagging the first 1.0.0 release with no prior tags
  if [[ "$MAJOR" == 1 && "$MINOR" == 0 && "$PATCH" == 0 ]]; then
    echo "First 1.0.0 release, so OK"
    found_parent=true
  fi

  if [[ $found_parent == false ]]; then
    echo "Invalid $NEW_VERSION version. Expected parent version matching $parent_version does not exist."
    FAIL_VALIDATION=1
  fi
}

# Start of actual code
echo "Checking git repo with remotes:"
git remote -v

echo "Branches:"
branches=$(git branch -v)
echo "$branches"

echo "Existing git tags:"
existing_tags=$(git tag -l)
echo "$existing_tags"

read_version
is_valid_version
check_if_releaseversion

# perform checks if a released version
if [ "$RELEASE_VERSION" -eq "1" ]
then
  is_git_tag_duplicated "$existing_tags"
  dockerfile_parentcheck
fi

exit "$FAIL_VALIDATION"
