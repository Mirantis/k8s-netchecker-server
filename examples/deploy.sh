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
#
# env: NS - a namespace name (also as $1)
# env: KUBE_DIR - manifests directory, e.g. /etc/kubernetes
# env: KUBE_USER - a user to own the manifests directory
# env: NODE_PORT - a node port for the server app to listen on
# env: PURGE - if true, will only erase applications
# env: AGENT_REPORT_INTERVAL - an interval for agents to report

set -o xtrace
set -o pipefail
set -o errexit
set -o nounset


NS=${NS:-default}
REAL_NS="--namespace=${1:-$NS}"
KUBE_DIR=${KUBE_DIR:-.}
KUBE_USER=${KUBE_USER:-}
NODE_PORT=${NODE_PORT:-31081}
PURGE=${PURGE:-false}
SERVER_IMAGE_NAME=${SERVER_IMAGE_NAME:-mirantis/k8s-netchecker-server}
AGENT_IMAGE_NAME=${AGENT_IMAGE_NAME:-mirantis/k8s-netchecker-agent}
IMAGE_TAG=${IMAGE_TAG:-latest}
SERVER_IMAGE_TAG=${SERVER_IMAGE_TAG:-$IMAGE_TAG}
AGENT_IMAGE_TAG=${AGENT_IMAGE_TAG:-$IMAGE_TAG}


if [ "${KUBE_DIR}" != "." ] && [ -n "${KUBE_USER}" ]; then
  mkdir -p "${KUBE_DIR}"
fi

# check there are nodes in the cluster
kubectl get nodes

echo "Installing netchecker server"
cat << EOF > "${KUBE_DIR}"/netchecker-server-pod.yml
apiVersion: v1
kind: Pod
metadata:
  name: netchecker-server
  labels:
    app: netchecker-server
spec:
  containers:
    - name: netchecker-server
      image: ${SERVER_IMAGE_NAME}:${SERVER_IMAGE_TAG}
      imagePullPolicy: Always
      ports:
        - containerPort: 8081
          hostPort: 8081
      args:
        - "-v=5"
        - "-logtostderr"
        - "-kubeproxyinit"
        - "-endpoint=0.0.0.0:8081"
EOF

cat << EOF > "${KUBE_DIR}"/netchecker-server-svc.yml
apiVersion: v1
kind: "Service"
metadata:
  name: netchecker-service
spec:
  selector:
    app: netchecker-server
  ports:
    -
      protocol: TCP
      port: 8081
      targetPort: 8081
      nodePort: ${NODE_PORT}
  type: NodePort
EOF

cat << EOF > "${KUBE_DIR}"/netchecker-agent-ds.yml
apiVersion: extensions/v1beta1
kind: DaemonSet
metadata:
  labels:
    app: netchecker-agent-hostnet
  name: netchecker-agent
spec:
  template:
    metadata:
      name: netchecker-agent
      labels:
        app: netchecker-agent
    spec:
      containers:
        - name: netchecker-agent
          image: ${AGENT_IMAGE_NAME}:${AGENT_IMAGE_TAG}
          env:
            - name: MY_NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            - name: MY_POD_NAME
              valueFrom:
                fieldRef:
                  fieldPath: metadata.name
          args:
            - "-v=5"
            - "-logtostderr"
            - "-serverendpoint=netchecker-service:8081"
            - "-reportinterval=60"
          imagePullPolicy: Always
EOF

cat << EOF > "${KUBE_DIR}"/netchecker-agent-hostnet-ds.yml
apiVersion: extensions/v1beta1
kind: DaemonSet
metadata:
  labels:
    app: netchecker-agent-hostnet
  name: netchecker-agent-hostnet
spec:
  template:
    metadata:
      name: netchecker-agent-hostnet
      labels:
        app: netchecker-agent-hostnet
    spec:
      hostNetwork: True
      containers:
        - name: netchecker-agent
          image: ${AGENT_IMAGE_NAME}:${AGENT_IMAGE_TAG}
          env:
            - name: MY_NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            - name: MY_POD_NAME
              valueFrom:
                fieldRef:
                  fieldPath: metadata.name
          args:
            - "-v=5"
            - "-logtostderr"
            - "-serverendpoint=netchecker-service:8081"
            - "-reportinterval=60"
          imagePullPolicy: Always
EOF

if [ "${KUBE_DIR}" != "." ] && [ -n "${KUBE_USER}" ]; then
  chown -R "${KUBE_USER}":"${KUBE_DIR}"
fi

kubectl delete --grace-period=1 -f "${KUBE_DIR}"/netchecker-agent-ds.yml "${REAL_NS}" || true
kubectl delete --grace-period=1 -f "${KUBE_DIR}"/netchecker-agent-hostnet-ds.yml "${REAL_NS}" || true
kubectl delete --grace-period=1 -f "${KUBE_DIR}"/netchecker-server-svc.yml "${REAL_NS}" || true
(kubectl delete --grace-period=1 -f "${KUBE_DIR}"/netchecker-server-pod.yml "${REAL_NS}" && sleep 10) || true

if [ "${PURGE}" != "true" ]; then
  kubectl create -f "${KUBE_DIR}"/netchecker-server-pod.yml "${REAL_NS}"
  kubectl create -f "${KUBE_DIR}"/netchecker-server-svc.yml "${REAL_NS}"
  kubectl create -f "${KUBE_DIR}"/netchecker-agent-ds.yml "${REAL_NS}"
  kubectl create -f "${KUBE_DIR}"/netchecker-agent-hostnet-ds.yml "${REAL_NS}"
fi

set +o xtrace
echo "DONE"
echo "Use the following commands to "
echo "- check agents responses:"
echo "  curl -s -X GET 'http://localhost:${NODE_PORT}/api/v1/agents/' | python -mjson.tool"
echo "- check connectivity with agents:"
echo "  curl -X GET 'http://localhost:${NODE_PORT}/api/v1/connectivity_check'"
