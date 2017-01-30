#!/bin/bash
# env: NS - a namespace name (also as $1)
# env: KUBE_DIR - manifests directory, e.g. /etc/kubernetes
# env: KUBE_USER - a user to own the manifests directory
# env: NODE_PORT - a node port for the server app to listen on
# env: PURGE - if true, will only erase applications
# env: AGENT_REPORT_INTERVAL - an interval for agents to report
set -e

SERVER_IMAGE_NAME=aateem/k8s-netchecker-server
SERVER_IMAGE_TAG=golang
AGENT_IMAGE_NAME=aateem/k8s-netchecker-agent
AGENT_IMAGE_TAG=golang

if [ -n "$1" -o -n "${NS}" ] ; then
  NS="--namespace=${1:-NS}"
fi
NODE_PORT=${NODE_PORT:-31081}
KUBE_DIR="${KUBE_DIR:-./}"
if [ "${KUBE_DIR}" != "./" -a -n "${KUBE_USER}" ]; then
  mkdir -p ${KUBE_DIR}
fi

# check there are nodes in the cluster
kubectl get nodes

echo "Installing netchecker server"
cat << EOF > ${KUBE_DIR}/netchecker-server-pod.yml
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
        - "v=5"
        - "-logtostderr"
        - "-kubeproxyinit"
        - "-endpoint=0.0.0.0:8081"
EOF

cat << EOF > ${KUBE_DIR}/netchecker-server-svc.yml
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

cat << EOF > ${KUBE_DIR}/netchecker-agent-ds.yml
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

cat << EOF > ${KUBE_DIR}/netchecker-agent-hostnet-ds.yml
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

if [ "${KUBE_DIR}" != "./" -a -n "${KUBE_USER}" ]; then
  chown -R ${KUBE_USER}: ${KUBE_DIR}
fi

kubectl delete --grace-period=1 -f ${KUBE_DIR}/netchecker-agent-ds.yml $NS || true
kubectl delete --grace-period=1 -f ${KUBE_DIR}/netchecker-agent-hostnet-ds.yml $NS || true
kubectl delete --grace-period=1 -f ${KUBE_DIR}/netchecker-server-svc.yml $NS || true
(kubectl delete --grace-period=1 -f ${KUBE_DIR}/netchecker-server-pod.yml $NS && sleep 10) || true

if [ "${PURGE:-false}" != "true" ]; then
  kubectl create -f ${KUBE_DIR}/netchecker-server-pod.yml $NS
  kubectl create -f ${KUBE_DIR}/netchecker-server-svc.yml $NS
  kubectl create -f ${KUBE_DIR}/netchecker-agent-ds.yml $NS
  kubectl create -f ${KUBE_DIR}/netchecker-agent-hostnet-ds.yml $NS
fi

echo "DONE"
echo "use the following commands to "
echo "- check agents responses:"
echo "curl -s -X GET 'http://localhost:${NODE_PORT}/api/v1/agents/' | python -mjson.tool"
echo "- check connectivity with agents:"
echo "curl -X GET 'http://localhost:${NODE_PORT}/api/v1/connectivity_check'"
