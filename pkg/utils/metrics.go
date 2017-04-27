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

	"github.com/golang/glog"
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
	name := ai.NodeName
	am.ErrorCount = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace:   "ncagent",
		Name:        "error_count_total",
		ConstLabels: prometheus.Labels{"agent": fmt.Sprintf("%s-%s", name, suffix)},
		Help:        "Total number of errors (keepalive miss count) for the agent.",
	})
	am.ReportCount = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace:   "ncagent",
		Name:        "report_count_total",
		ConstLabels: prometheus.Labels{"agent": fmt.Sprintf("%s-%s", name, suffix)},
		Help:        "Total number of reports (keepalive messages) from the agent.",
	})

	if counter, ok := tryRegister(am.ErrorCount); !ok {
		// use existing counter
		am.ErrorCount = counter
	}
	if counter, ok := tryRegister(am.ReportCount); !ok {
		// use existing counter
		am.ReportCount = counter
	}

	return am
}

// returns true if registering went fine, false if counter was registered already,
// panics on other register errors
func tryRegister(m prometheus.Counter) (prometheus.Counter, bool) {
	if err := prometheus.Register(m); err != nil {
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			// A counter for that metric has been registered before.
			existing := are.ExistingCollector.(prometheus.Counter)
			glog.V(5).Infof("Counter %v has been registered already.", existing.Desc())
			return existing, false
		}
		// Something else went wrong!
		panic(err)
	}
	return m, true
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
