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


HELM_SCRIPT_URL=${HELM_SCRIPT_URL:-https://raw.githubusercontent.com/kubernetes/helm/master/scripts/get}
HELM_SCRIPT_NAME=${HELM_SCRIPT_NAME:-get_helm.sh}
HELM_SERVER_PATH=${HELM_SERVER_PATH:-helm-chart/netchecker-server}
HELM_AGENT_PATH=${HELM_AGENT_PATH:-helm-chart/netchecker-agent}
HELM_DEBUG=${HELM_DEBUG:-"--debug"}
NETCHECKER_REPO=${NETCHECKER_REPO:-}
KUBECTL_DIR="${KUBECTL_DIR:-${HOME}/.kubeadm-dind-cluster}"
PATH="${KUBECTL_DIR}:${PATH}"


function wait-for-tiller-pod-ready() {
  local name="${1}"
  local timeout_secs=60
  local increment_secs=1
  local waited_time=0

  while [ "${waited_time}" -lt "${timeout_secs}" ]; do
    tiller_replicas="$(kubectl get deployment "${name}" \
                       -o 'go-template={{.status.availableReplicas}}' \
                       --namespace kube-system)"

    if [ "${tiller_replicas}" == "1" ]; then
      return 0
    fi

    sleep "${increment_secs}"
    (( waited_time += increment_secs ))

    if [ "${waited_time}" -ge "${timeout_secs}" ]; then
      echo "${name} was never ready."
      exit 1
    fi
    echo -n . 1>&2
  done
}


function install-helm {
  pushd "./scripts" &> /dev/null
  wget -O "${HELM_SCRIPT_NAME}" "${HELM_SCRIPT_URL}"
  chmod +x ./"${HELM_SCRIPT_NAME}"
  set +o errexit
  bash -x ./"${HELM_SCRIPT_NAME}"
  echo "Uninstall tiller-deploy if exists"
  kubectl delete deployment "tiller-deploy" --namespace "kube-system" &> /dev/null || true
  set -o errexit
  helm "${HELM_DEBUG}" init
  wait-for-tiller-pod-ready "tiller-deploy"
  helm "${HELM_DEBUG}" version
  popd &> /dev/null
}


function lint-helm {
  if [ -z "${NETCHECKER_REPO}" ]; then
    echo "NETCHECKER_REPO is not set!"
    exit 1
  fi
  if [ "${NETCHECKER_REPO}" == "k8s-netchecker-server" ]; then
    helm "${HELM_DEBUG}" lint ./"${HELM_AGENT_PATH}"/
  else
    helm "${HELM_DEBUG}" lint ./"${HELM_SERVER_PATH}"/
  fi
}


function deploy-helm {
  if [ "${NETCHECKER_REPO}" == "k8s-netchecker-server" ]; then
    pushd "../${NETCHECKER_REPO}" &> /dev/null
    helm "${HELM_DEBUG}" install ./"${HELM_SERVER_PATH}"/
    popd &> /dev/null
    helm "${HELM_DEBUG}" install ./"${HELM_AGENT_PATH}"/
  else
    helm "${HELM_DEBUG}" install ./"${HELM_SERVER_PATH}"/
    pushd "../${NETCHECKER_REPO}" &> /dev/null
    helm "${HELM_DEBUG}" install ./"${HELM_AGENT_PATH}"/
    popd &> /dev/null
  fi
  helm "${HELM_DEBUG}" list
}


install-helm
lint-helm
deploy-helm
