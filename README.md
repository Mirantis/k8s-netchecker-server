Overview
========

Network checker is an application that is meant to provide simple test for
network connectivity. Connectivity is checked between the server part and multitude
of agents. The agents then make periodical reports to the server and presence of
reported data indicates aliveness of the link.

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
