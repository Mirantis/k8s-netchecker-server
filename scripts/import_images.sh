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


IMAGE_REPO_SERVER=${IMAGE_REPO_SERVER:-mirantis/k8s-netchecker-server}
IMAGE_REPO_AGENT=${IMAGE_REPO_AGENT:-mirantis/k8s-netchecker-agent}
IMAGE_TAG=${IMAGE_TAG:-latest}
NUM_NODES=${NUM_NODES:-3}
TMP_IMAGE_PATH=${TMP_IMAGE_PATH:-/tmp/netchecker-all.tar}
# export MASTER_NAME=kube-master if you need
# to import images in kube-master node
MASTER_NAME=${MASTER_NAME:-}
SLAVE_NAME=${SLAVE_NAME:-"kube-node-"}


function import-images {
	docker save -o "${TMP_IMAGE_PATH}" \
	"${IMAGE_REPO_SERVER}":"${IMAGE_TAG}" "${IMAGE_REPO_AGENT}":"${IMAGE_TAG}"

  if [ ! -z "${MASTER_NAME}" ]; then
    docker cp "${TMP_IMAGE_PATH}" "${MASTER_NAME}":/netchecker-all.tar
    docker exec -ti "${MASTER_NAME}" docker load -i /netchecker-all.tar
    docker exec -ti "${MASTER_NAME}" docker images
  fi

  for node in $(seq 1 "${NUM_NODES}"); do
    docker cp "${TMP_IMAGE_PATH}" "${SLAVE_NAME}""${node}":/netchecker-all.tar
    docker exec -ti "${SLAVE_NAME}""${node}" docker load -i /netchecker-all.tar
    docker exec -ti "${SLAVE_NAME}""${node}" docker images
  done
  echo "Finished copying docker images to dind nodes"
}

import-images