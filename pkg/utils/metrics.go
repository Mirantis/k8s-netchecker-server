// Copyright 2017 Mirantis
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	gaugeNumberOfAgents = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "netcheck_num_agents",
		Help: "Total number of agents in cluster.",
	})
	gaugeNumberOfHosts = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "netcheck_num_hosts",
		Help: "Total number of hosts having agents in cluster.",
	})
)

func init() {
	prometheus.MustRegister(gaugeNumberOfAgents)
	prometheus.MustRegister(gaugeNumberOfHosts)
}

type Metrics struct {
	numberOfAgents int
	numberOfHosts  int
}

func (s *Metrics) Update() {
	gaugeNumberOfAgents.Set(float64(s.numberOfAgents))
	gaugeNumberOfHosts.Set(float64(s.numberOfHosts))
}
