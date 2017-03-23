![Diagram](diagram.png)

Overview
========

Network checker is an application that is meant to provide simple test for
network connectivity. Connectivity is checked between the server part and multitude
of agents. The agents then make periodical reports to the server and presence of
reported data indicates aliveness of the link.

How it works
============

Network checker is an Kubernetes application main purpose of which is checking of connectivity
between the cluster's nodes. Network checker consists of two parts: server (this repository)
and agent ([developed here](https://github.com/Mirantis/k8s-netchecker-agent)). Agents are deployed on
every K8S node using [Daemonset mechanism](https://kubernetes.io/docs/admin/daemons/)
(to ensure auto-management of the pods). Agents come in two flavors - and there exists two daemonsets
for each of them. The difference between them is that "Agent-hostnet" is tapped into host network namespace
via supplying `hostNetwork: True` key-value for corresponding Pod's spec.

The agents then periodically gather network related information
(e.g. interfaces' info, results of nslookup, etc.) and send formed payload to
the server address in the K8S cluster's network space.

The server is deployed behind a Kubernetes service.


Usage
=====

To start the server inside k8s pod and listen on port 8081 use following
arguments:

```bash
server -v 5 -logtostderr -kubeproxyinit -endpoint 0.0.0.0:8081
```

API interface
=============

The server exposes following API interface.

- GET/POST - /api/v1/agents/{agent_name} - get, create/update agent's entry in the agent cache
- GET - /api/v1/agents/ - get the whole agent cache dump
- GET - /api/v1/connectivity_check - get result of connectivity check between the server and the agents
