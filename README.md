# Status

[![Build Status](https://goo.gl/XzSwDu)](https://goo.gl/bx20uy)
[![Stories in Progress](https://goo.gl/Y3SfPH)](https://goo.gl/eY1d9l)
[![Go Report Card](https://goo.gl/EN7y2i)](https://goo.gl/ultF3D)
[![Code Climate](https://goo.gl/F5iNWP)](https://goo.gl/mGsQj1)
[![License Apache 2.0](https://goo.gl/joRzTI)](https://goo.gl/pbOuG0)
[![Docker Pulls](https://goo.gl/ZYz1nt)](https://goo.gl/nAfD9C)

## What it is and how it works

![Diagram](diagram.png)

Network checker is a Kubernetes application main purpose of which is checking
of connectivity between the cluster's nodes. Network checker consists of two
parts: server (this repository) and agent
([developed here](https://github.com/Mirantis/k8s-netchecker-agent)). Agents
are deployed on every K8S node using
[Daemonset mechanism](https://kubernetes.io/docs/admin/daemons/)
(to ensure auto-management of the pods). Agents come in two flavors - and there
 exist two daemonsets for each kind. The difference between them is that
"Agent-hostnet" is tapped into host network namespace via supplying
`hostNetwork: True` key-value for corresponding Pod's spec. As shown on diagram
both daemonsets are enabled for each node meaning one and only one pod of each
kind will be present there; consequently those pods
uniquely represent nodes inside server's internals.

The agents then periodically gather network related information
(e.g. interfaces' info, results of nslookup, etc.) and send formed payload to
the server address in the K8S cluster's network space.

The server is deployed in dedicated Pod and exposed inside of the cluster via
K8S service resource. Thus every agent can access the server by the service's
DNS name.

Incoming agent data is stored in server's memory (this fact then implicates
loss of the data when server application is shutdown or restarted; it is going
to be reworked by adding a permanent storage in the future).
The cache is represented as Go map data structure with an agent's pod name as
unique key.

Server provides HTTP RESTful interface which for the time present consists of
the following (verb - URI designator - meaning of the operation):

- GET/POST - /api/v1/agents/{agent_name} - get, create/update agent's entry in
  the agent cache
- GET - /api/v1/agents/ - get the whole agent cache dump
- GET - /api/v1/connectivity_check - get result of connectivity check between
  the server and the agents

The main logic of network checking is implemented behind `connnectivity_check`
endpoint. In order to determine whether connectivity is present between
the server and agents former retrieves list of pods from K8S API having labels
`netchecker-agent` and `netchecker-agent-hostnet`, then analyses its cache.
Successful checking consists in meeting two criteria. First - there is entry in
the cache for one of the retrieved pods; it means an agent request made through
the network to the server consequently link is established and active between
them. Second - difference between time of the check and point when the data was
received from agent must not exceed the period of agent's reporting
(there is field in the payload storing the period); in opposite case
it will indicate that connection is lost and requests are not coming through.
Let us remember that each agent corresponds to one particular pod, unique for
particular node, so connection between agents and server means
connection between nodes on which all those components are deployed.

Results of connectivity check are represented in response from the endpoint
particularly indicating possible problematic situation in the payload (e.g.
there are dedicated field showing agents which haven't reported at all and
those which reports are exceeding its interval). Users then can consume the
data.

One aspect of functioning of network checker is worth mentioning. Payloads sent
by the agents are of relatively small byte size which in some cases can be less
than MTU value set for the cluster's network link. When this happens the
network checker will not catch problems with network packet's fragmentation.
For that reason special option can be used with the agent application -
`-zeroextenderlength`. By default it has value of 1500. The parameter tells
the agent to extend each payload by given length of zero bytes to exceed
packet fragmentation trigger threshold. This dummy data then has no effect
on the server's processing of the agent's requests.

## Usage

To start the server inside k8s pod and listen on port 8081 use following
arguments:

```bash
server -v 5 -logtostderr -kubeproxyinit -endpoint 0.0.0.0:8081
```

For testing purposes it may be needed to start the server without any attempts
to initialize Kubernetes client code (which is done by default). In that case
additional option must be provided

```
-kubeproxyinit=false
```

For other possibilities regarding testing, code and Docker images building etc.
please refer to the Makefile.

## Deployment on K8S cluster

For deployment two option can be used. One - `./examples/deploy.sh` script;
users must provide all needed variables (e.g. name and tag for Docker images)
by modifying of the script.

Deployment as helm chart: if users have
[Helm](https://github.com/kubernetes/helm) installed on their K8S cluster
they can build the chart from its description (`./helm-chart/`) and then deploy
it (please, use Helm's documentation for details).

## Additional documentation

* [Metrics](doc/metrics.md) - metrcis and Prometheus configuration how to.
