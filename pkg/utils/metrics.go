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

	if counter, ok := tryRegisterCounter(am.ErrorCount); !ok {
		// use existing counter
		am.ErrorCount = counter
	}
	if counter, ok := tryRegisterCounter(am.ReportCount); !ok {
		// use existing counter
		am.ReportCount = counter
	}

	am.ProbeConnectionResult = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   "ncagent",
			Name:        "http_probe_connection_result",
			Help:        "Connection result: 0 - error, 1 - success",
		},
		[]string{"agent", "url"},
	)
	am.ProbeHTTPCode = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   "ncagent",
			Name:        "http_probe_code",
			Help:        "HTTP status code.",
		},
		[]string{"agent", "url"},
	)
	am.ProbeTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   "ncagent",
			Name:        "http_probe_total_time_ms",
			Help:        "The duration of total http request.",
		},
		[]string{"agent", "url"},
	)
	am.ProbeContentTransfer = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   "ncagent",
			Name:        "http_probe_content_transfer_time_ms",
			Help:        fmt.Sprint(
			               "The duration of content transfer time, from the first ",
			               "reponse byte till the end (in ms).",
			             ),
		},
		[]string{"agent", "url"},
	)
	am.ProbeTCPConnection = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   "ncagent",
			Name:        "http_probe_tcp_connection_time_ms",
			Help:        "TCP establishing time in ms.",
		},
		[]string{"agent", "url"},
	)
	am.ProbeDNSLookup = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   "ncagent",
			Name:        "http_probe_dns_lookup_time_ms",
			Help:        "DNS lookup time in ms.",
		},
		[]string{"agent", "url"},
	)
	am.ProbeConnect = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   "ncagent",
			Name:        "http_probe_connect_time_ms",
			Help:        "Connection time in ms",
		},
		[]string{"agent", "url"},
	)
	am.ProbeServerProcessing = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   "ncagent",
			Name:        "http_probe_server_processing_time_ms",
			Help:        "Server processing time in ms.",
		},
		[]string{"agent", "url"},
	)

	if gauge, ok := tryRegisterGaugeVec(am.ProbeConnectionResult); !ok {
		// use existing gauge
		am.ProbeConnectionResult = gauge
	}
	if gauge, ok := tryRegisterGaugeVec(am.ProbeHTTPCode); !ok {
		// use existing gauge
		am.ProbeHTTPCode = gauge
	}
	if gauge, ok := tryRegisterGaugeVec(am.ProbeTotal); !ok {
		// use existing gauge
		am.ProbeTotal = gauge
	}
	if gauge, ok := tryRegisterGaugeVec(am.ProbeContentTransfer); !ok {
		// use existing gauge
		am.ProbeContentTransfer = gauge
	}
	if gauge, ok := tryRegisterGaugeVec(am.ProbeTCPConnection); !ok {
		// use existing gauge
		am.ProbeTCPConnection = gauge
	}
	if gauge, ok := tryRegisterGaugeVec(am.ProbeTCPConnection); !ok {
		// use existing gauge
		am.ProbeTCPConnection = gauge
	}
	if gauge, ok := tryRegisterGaugeVec(am.ProbeDNSLookup); !ok {
		// use existing gauge
		am.ProbeDNSLookup = gauge
	}
	if gauge, ok := tryRegisterGaugeVec(am.ProbeConnect); !ok {
		// use existing gauge
		am.ProbeConnect = gauge
	}
	if gauge, ok := tryRegisterGaugeVec(am.ProbeServerProcessing); !ok {
		// use existing gauge
		am.ProbeServerProcessing = gauge
	}

	return am
}

// returns true if registering went fine, false if counter was registered already,
// panics on other register errors
func tryRegisterCounter(m prometheus.Counter) (prometheus.Counter, bool) {
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

func tryRegisterGaugeVec(m *prometheus.GaugeVec) (*prometheus.GaugeVec, bool) {
	if err := prometheus.Register(m); err != nil {
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			// A gauge for that metric has been registered before.
			existing := are.ExistingCollector.(*prometheus.GaugeVec)
			//glog.V(5).Infof("GaugeVec %v has been registered already.", existing.Desc())
			return existing, false
		}
		// Something else went wrong!
		panic(err)
	}
	return m, true
}

func UpdateAgentBaseMetrics(am AgentMetrics, report, error bool) {
	if report {
		am.ReportCount.Inc()
		am.ErrorsFromLastReport = 0
	}
	if error {
		am.ErrorCount.Inc()
		am.ErrorsFromLastReport += 1
	}
}

func UpdateAgentProbeMetrics(ai AgentInfo, am AgentMetrics) {

	suffix := "private_network"
	if strings.Contains(ai.PodName, "hostnet") {
		suffix = "host_network"
	}
	name := fmt.Sprintf("%s-%s", ai.NodeName, suffix)

	for _, pr := range ai.NetworkProbes {
		am.ProbeConnectionResult.WithLabelValues(name, pr.URL).Set(float64(pr.ConnectionResult))
		am.ProbeHTTPCode.WithLabelValues(name, pr.URL).Set(float64(pr.HTTPCode))
		am.ProbeTotal.WithLabelValues(name, pr.URL).Set(float64(pr.Total))
		am.ProbeContentTransfer.WithLabelValues(name, pr.URL).Set(float64(pr.ContentTransfer))
		am.ProbeTCPConnection.WithLabelValues(name, pr.URL).Set(float64(pr.TCPConnection))
		am.ProbeDNSLookup.WithLabelValues(name, pr.URL).Set(float64(pr.DNSLookup))
		am.ProbeConnect.WithLabelValues(name, pr.URL).Set(float64(pr.Connect))
		am.ProbeServerProcessing.WithLabelValues(name, pr.URL).Set(float64(pr.ServerProcessing))
	}
}
