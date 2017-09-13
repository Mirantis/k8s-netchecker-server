# Status

[![Build Status](https://goo.gl/XzSwDu)](https://goo.gl/bx20uy)
[![Stories in Progress](https://goo.gl/Y3SfPH)](https://goo.gl/eY1d9l)
[![Go Report Card](https://goo.gl/EN7y2i)](https://goo.gl/ultF3D)
[![Code Climate](https://goo.gl/F5iNWP)](https://goo.gl/mGsQj1)
[![License Apache 2.0](https://goo.gl/joRzTI)](https://goo.gl/pbOuG0)
[![Docker Pulls](https://goo.gl/ZYz1nt)](https://goo.gl/nAfD9C)

## What it is and how it works

![Diagram](diagram.png)

Network checker is a Kubernetes application. Its main purpose is checking
of connectivity between the cluster's nodes. Network checker consists of two
parts: server (this repository) and agent
([developed here](https://github.com/Mirantis/k8s-netchecker-agent)). Agents
are deployed on every Kubernetes node using
[Daemonset](https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/).
Agents come in two flavors - and default setup includes two corresponding
daemonsets. The difference between them is that "Agent-hostnet" is tapped into
host network namespace via supplying `hostNetwork: True` key-value for the
corresponding Pod's specification. As shown on the diagram, both daemonsets
are enabled for each node meaning exactly one pod of each kind will be deployed
on each node.

The agents periodically gather network related information
(e.g. interfaces' info, results of nslookup, results of latencies measurement,
etc.) and send it to the server as periodic agent reports.
Report includes agent pod name and its node name so that the report is uniquely
identified using them.

The server is deployed in a dedicated pod using
[Deployment](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/)
and exposed inside of the cluster via Kubernetes service resource. Thus, every
agent can access the server by the service's DNS name.

Server processes the incoming agent data (agents' reports) and store it in a
persistent data storage. Server is capable to use either Kubernetes third party
resources (TPR) or etcd as a persistent data storage:

- TPR. New data type called `agent` was added into TPR, Kubernetes API was extended
  with this new type, and all agent data is stored using it. When using TPR,
  the server is vulnerable to [date change issue](https://github.com/Mirantis/k8s-netchecker-server/issues/80).
  The issue was solved by using etcd and its TTL feature. Please also note
  that TPR is deprecated starting from Kubernetes v.1.7 and can be removed in
  future Kubrenetes versions. It will not be supported in Netchecker then. No
  migration to Kubernetes CRD (substitution for TPR) is planned either.
- etcd. The recommended storage provider. When using etcd, the server is resistant
  to issues described in TPR section. Agent data is stored in etcd in this case,
  under `/netchecker` path.

Server also calculates metrics based on agent data. Metrics data is stored in
server's memory for now - this implicates loss of metrics data when server
application is shutdown or restarted; it is going to be reworked by moving to
a persistent storage (to etcd only) in future.

Server provides HTTP RESTful interface which currently includes the following
requests (verb - URI designator - meaning of the operation):

- GET/POST - /api/v1/agents/{agent_name} - get, create/update agent's data record
  in a persistant storage.
- GET - /api/v1/agents/ - get the whole agent data dump.
- GET - /api/v1/connectivity_check - get result of connectivity check between
  the server and the agents.
- GET - /metrics - get the network checker metrics.

The main logic of network checking is implemented behind `connectivity_check`
endpoint. It is the only user-facing URI.
In order to determine whether connectivity is present between the server and
agents, former retrieves the list of pods using Kubernetes API
(filtering by labels `netchecker-agent` and `netchecker-agent-hostnet`), then
analyses stored agent data.
Success of the checking is determined based on two criteria.
First - there is an entry in the stored data for the each retrieved agent's pod;
it means an agent request has got through the network to the server. Consequently,
link is established and active within the agent-server pair.
Second - difference between the time of the check and the time when the data
was received from particular agent must not exceed two periods of agent's
reporting (there is a field in the payload holding the report interval). In
opposite case, it will indicate that connection is lost and requests are not
coming through. In case of using etcd, period of agent's data obsolescence is
set explicitly in parameters to the server (`-report-ttl` parameter, in seconds).
Let us remember that each agent corresponds to one particular pod, unique for
particular node, so connection between agents and server means connection
between the corresponding nodes.

Results of the connectivity check which are represented in response from the
endpoint particularly indicate possible connectivity issue (e.g. there is an
`Absent` field listing agents which haven't reported at all and `Outdated` one
listing those which reports are out of data obsolescence period).

One aspect of functioning of network checker is worth mentioning. Payloads sent
by the agents are of relatively small byte size which in some cases can be less
than MTU value set for the cluster's network links. When this happens, the
network checker will not catch problems with network packet's fragmentation.
For that reason, special option can be used with the agent application -
`-zeroextenderlength`. By default, it has value of 1500. The parameter tells
the agent to extend each payload by given length to exceed packet fragmentation
trigger threshold. This dummy data has no effect on the server's processing
of the agent's requests (reports).

## Usage

To start the server inside Kubernetes pod using Kubernetes TPR as a persistent storage
and listen on port 8081, use the following command:

```bash
server -v 5 -logtostderr -kubeproxyinit -endpoint 0.0.0.0:8081
```

To start the server using etcd as a persistent storage, use the following setting:

```
-kubeproxyinit=false
```

Also, a few parameters are required to establish the connection with etcd:

```
-etcd-endpoints=https://192.0.10.11:4001,https://192.0.10.12:4001
-etcd-key=/var/lib/etcd/client.key (optional, ommited when using http)
-etcd-cert=/var/lib/etcd/client.pem (optional, ommited when using http)
-etcd-ca=/var/lib/etcd/ca.pem (optional, can be ommited even when using https)
```

For other possibilities regarding testing, code and Docker images building etc.
please refer to the Makefile.

## Deployment in Kubernetes cluster

In order to deploy the application, two options can be used.

First - using `./examples/deploy.sh` script. Users must provide all the needed
environment variables (e.g. name and tag for Docker images) before running the
script.

Second - deploy as a helm chart. If users have
[Helm](https://github.com/kubernetes/helm) installed on their Kubernetes cluster
they can build the chart from its description (`./helm-chart/`) and then deploy
it (please, use Helm's documentation for details).

## Additional documentation

- [Metrics](doc/metrics.md) - metrics and Prometheus configuration how to.

