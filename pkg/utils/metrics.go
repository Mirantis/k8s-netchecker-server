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
	"fmt"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

func NewAgentMetrics(ai *AgentInfo) AgentMetrics {
	am := AgentMetrics{
		PodName: ai.PodName,
	}

	suffix := "private_network"
	if strings.Contains(ai.PodName, "hostnet") {
		suffix = "host_network"
	}
	name_splitted := strings.Split(ai.PodName, "-")
	name := name_splitted[len(name_splitted)-1]
  am.ErrorCount = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "ncagent",
		Name: "error_count_total",
    ConstLabels: prometheus.Labels{"agent": fmt.Sprintf("%s-%s", name, suffix)},
		Help: "Total number of errors (keepalive miss count) for the agent.",
	})
  am.ReportCount = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "ncagent",
		Name: "report_count_total",
    ConstLabels: prometheus.Labels{"agent": fmt.Sprintf("%s-%s", name, suffix)},
		Help: "Total number of reports (keepalive messages) from the agent.",
	})

	prometheus.MustRegister(am.ErrorCount)
	prometheus.MustRegister(am.ReportCount)
	return am
}

func UpdateAgentMetrics(am AgentMetrics, report, error bool) {
	if report {
		am.ReportCount.Inc()
		am.ErrorsFromLastReport = 0
	}
	if error {
		am.ErrorCount.Inc()
		am.ErrorsFromLastReport += 1
	}
}
