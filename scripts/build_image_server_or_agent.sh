#!/bin/bash
# Copyright 2017 Mirantis
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o xtrace
set -o pipefail
set -o errexit
set -o nounset


NETCHECKER_REPO=${NETCHECKER_REPO:-}
NETCHECKER_BRANCH=${NETCHECKER_BRANCH:-master}


function build-image-server-or-agent {
  if [ -z "${NETCHECKER_REPO}" ]; then
  	echo "NETCHECKER_REPO is not set!"
  	exit 1
  else
      pushd "../" &> /dev/null
      if [ ! -d "${NETCHECKER_REPO}" ]; then
        git clone --branch "${NETCHECKER_BRANCH}" \
            --depth 1 --single-branch "https://github.com/Mirantis/${NETCHECKER_REPO}.git"
      fi
  fi
  pushd "./${NETCHECKER_REPO}" &> /dev/null
  make build-image
  popd &> /dev/null
  popd &> /dev/null
}

build-image-server-or-agent