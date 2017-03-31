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
	am.gaugeErrorCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "Agent",
		Name: fmt.Sprintf("error_count_%s_%s", name, suffix),
		Help: "Total number of errors (keepalive miss count) for the agent.",
	})
	am.gaugeReportCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "Agent",
		Name: fmt.Sprintf("agent_report_count_%s_%s", name, suffix),
		Help: "Total number of reports (keepalive messages) from the agent.",
	})

	prometheus.MustRegister(am.gaugeErrorCount)
	prometheus.MustRegister(am.gaugeReportCount)
	return am
}

func UpdateAgentMetrics(am AgentMetrics, report, error bool) {
	if report {
		am.ReportCount += 1
		am.ErrorsFromLastReport = 0
		am.gaugeReportCount.Set(float64(am.ReportCount))
	}
	if error {
		am.ErrorCount += 1
		am.ErrorsFromLastReport += 1
		am.gaugeErrorCount.Set(float64(am.ErrorCount))
	}
}
