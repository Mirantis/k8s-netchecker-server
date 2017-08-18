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
	"reflect"
	"strings"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"

	ext_v1 "github.com/Mirantis/k8s-netchecker-server/pkg/extensions/apis/v1"
)

// NewAgentMetrics setup prometheus metrics
func NewAgentMetrics(ai *ext_v1.AgentSpec) AgentMetrics {
	am := AgentMetrics{
		PodName: ai.PodName,
	}

	suffix := "private_network"
	if strings.Contains(ai.PodName, "hostnet") {
		suffix = "host_network"
	}
	name := ai.NodeName

	// Basic Counter metrics
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

	// GaugeVec metrics for HTTP probes
	am.ProbeConnectionResult = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "ncagent",
			Name:      "http_probe_connection_result",
			Help:      "Connection result: 0 - error, 1 - success",
		},
		[]string{"agent", "url"},
	)
	am.ProbeHTTPCode = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "ncagent",
			Name:      "http_probe_code",
			Help:      "HTTP status code.",
		},
		[]string{"agent", "url"},
	)
	am.ProbeTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "ncagent",
			Name:      "http_probe_total_time_ms",
			Help:      "The total duration of http request.",
		},
		[]string{"agent", "url"},
	)
	am.ProbeContentTransfer = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "ncagent",
			Name:      "http_probe_content_transfer_time_ms",
			Help: fmt.Sprint(
				"The duration of content transfer, from the first ",
				"response byte till the end (in ms).",
			),
		},
		[]string{"agent", "url"},
	)
	am.ProbeTCPConnection = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "ncagent",
			Name:      "http_probe_tcp_connection_time_ms",
			Help:      "TCP establishing time in ms.",
		},
		[]string{"agent", "url"},
	)
	am.ProbeDNSLookup = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "ncagent",
			Name:      "http_probe_dns_lookup_time_ms",
			Help:      "DNS lookup time in ms.",
		},
		[]string{"agent", "url"},
	)
	am.ProbeConnect = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "ncagent",
			Name:      "http_probe_connect_time_ms",
			Help:      "Connection time in ms",
		},
		[]string{"agent", "url"},
	)
	am.ProbeServerProcessing = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "ncagent",
			Name:      "http_probe_server_processing_time_ms",
			Help:      "Server processing time in ms.",
		},
		[]string{"agent", "url"},
	)

	// Let's register all the metrics now
	params := reflect.ValueOf(&am).Elem()
	for i := 0; i < params.NumField(); i++ {
		if e, ok := params.Field(i).Interface().(*prometheus.GaugeVec); ok {
			if exists, ok := tryRegisterGaugeVec(e); !ok {
				params.Field(i).Set(reflect.ValueOf(exists))
			}
		} else if e, ok := params.Field(i).Interface().(prometheus.Counter); ok {
			if exists, ok := tryRegisterCounter(e); !ok {
				params.Field(i).Set(reflect.ValueOf(exists))
			}
		} else {
			glog.V(10).Infof("Skipping %v since it's not prometheus metric.", params.Type().Field(i).Name)
		}
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
			glog.V(10).Infof("Counter %v has been registered already.", existing.Desc())
			return existing, false
		}
		// Something else went wrong!
		panic(err)
	}
	return m, true
}

// returns true if registering went fine, false if GaugeVec was registered already,
// panics on other register errors
func tryRegisterGaugeVec(m *prometheus.GaugeVec) (*prometheus.GaugeVec, bool) {
	if err := prometheus.Register(m); err != nil {
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			// A gauge for that metric has been registered before.
			existing := are.ExistingCollector.(*prometheus.GaugeVec)
			return existing, false
		}
		// Something else went wrong!
		panic(err)
	}
	return m, true
}

// UpdateAgentBaseMetrics function updates basic metrics with reports and
// error counters
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

// UpdateAgentProbeMetrics function updates HTTP probe metrics.
func UpdateAgentProbeMetrics(ai ext_v1.AgentSpec, am AgentMetrics) {

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
