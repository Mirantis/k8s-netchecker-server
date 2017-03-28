# Network checker server

[![Build Status](https://goo.gl/XzSwDu)](https://goo.gl/bx20uy)
[![Stories in Progress](https://goo.gl/Y3SfPH)](https://goo.gl/eY1d9l)
[![Go Report Card](https://goo.gl/EN7y2i)](https://goo.gl/ultF3D)
[![Code Climate](https://goo.gl/F5iNWP)](https://goo.gl/mGsQj1)
[![License Apache 2.0](https://goo.gl/joRzTI)](https://goo.gl/pbOuG0)
[![Docker Pulls](https://goo.gl/ZYz1nt)](https://goo.gl/nAfD9C)

Network checker is an application that is meant to provide simple test for
network connectivity. Connectivity is checked between the server part and
multitude of agents. The agents then make periodical reports to the server and
presence of reported data indicates aliveness of the link.

## Usage

To start the server inside k8s pod and listen on port 8081 use following
arguments:

```bash
server -v 5 -logtostderr -kubeproxyinit -endpoint 0.0.0.0:8081
```

## API interface

The server exposes following API interface.

- GET/POST - /api/v1/agents/{agent_name} - get, create/update agent's entry in
  the agent cache

- GET - /api/v1/agents/ - get the whole agent cache dump

- GET - /api/v1/connectivity_check - get result of connectivity check between
  the server and the agents