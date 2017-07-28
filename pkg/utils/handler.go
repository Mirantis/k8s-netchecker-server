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
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/negroni"
)

func NewHandler() (*Handler, error) {
	appConfig := GetOrCreateConfig()

	h := &Handler{
		Metrics: NcAgentMetrics{},
	}

	var err error

	if appConfig.UseKubeClient {
		h.Agents, err = NewK8sStorer()
	} else {
		// use etcd for store states instead k8s 3d-part
		// h.Agents, err = NewEtcdStorer()
	}

	if err == nil {
		h.SetupRouter()
		h.AddMiddleware()
	}

	return h, err
}

func (h *Handler) SetupRouter() {
	glog.V(10).Info("Setting up the url multiplexer")

	router := httprouter.New()
	router.POST("/api/v1/agents/:name", h.UpdateAgents)
	router.GET("/api/v1/agents/:name", h.CleanCache(h.Agents.GetSingleAgent))
	router.GET("/api/v1/agents/", h.CleanCache(h.Agents.GetAgents))
	router.GET("/api/v1/connectivity_check", h.CleanCache(h.ConnectivityCheck))
	router.GET("/api/v1/ping", func(_ http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	})
	router.Handler("GET", "/metrics", promhttp.Handler())
	h.HTTPHandler = router
}

func (h *Handler) AddMiddleware() {
	n := negroni.New()
	n.Use(negroni.NewLogger())
	n.Use(negroni.NewRecovery())
	n.UseHandler(h.HTTPHandler)
	h.HTTPHandler = n
}

func (h *Handler) UpdateAgents(rw http.ResponseWriter, r *http.Request, rp httprouter.Params) {
	agentName := rp.ByName("name")

	agentData, err := h.Agents.UpdateAgents(rw, r, rp)
	if err != nil {
		glog.Error(err)
	}

	h.Metrics[agentName] = NewAgentMetrics(&agentData)
	UpdateAgentBaseMetrics(h.Metrics[agentName], true, false)
	UpdateAgentProbeMetrics(agentData, h.Metrics[agentName])
}

func (h *Handler) ConnectivityCheck(rw http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	res := &CheckConnectivityInfo{
		Message: fmt.Sprintf(
			"All %v pods successfully reported back to the server",
			len(h.Agents.AgentCache())),
	}
	status := http.StatusOK
	errMsg := "Connectivity check fails. Reason: %v"

	absent, outdated, err := h.Agents.CheckAgents()
	if err != nil {
		message := fmt.Sprintf(
			"Error occurred while checking the agents. Details: %v", err)
		glog.Error(message)
		http.Error(rw, message, http.StatusInternalServerError)
		return
	}

	if len(absent) != 0 || len(outdated) != 0 {
		glog.V(5).Infof(
			"Absent|outdated agents detected. Absent -> %v; outdated -> %v",
			absent, outdated,
		)
		res.Message = fmt.Sprintf(errMsg,
			"there are absent or outdated pods; look up the payload")
		res.Absent = absent
		res.Outdated = outdated

		status = http.StatusBadRequest
	}

	glog.V(10).Infof("Connectivity check result: %v", res)
	glog.V(10).Infof("Connectivity check HTTP response status code: %v", status)

	rw.WriteHeader(status)

	ProcessResponse(rw, res)
}

func (h *Handler) CleanCache(handle httprouter.Handle) httprouter.Handle {
	return func(rw http.ResponseWriter, r *http.Request, rp httprouter.Params) {
		h.Agents.CleanCacheOnDemand(rw)

		handle(rw, r, rp)
	}
}

func (h *Handler) CollectAgentsMetrics() {
	for {
		time.Sleep(5 * time.Second)
		for name := range h.Agents.AgentCache() {
			if _, exists := h.Metrics[name]; exists {
				deltaInIntervals := time.Now().Sub(h.Agents.AgentCache()[name].LastUpdated).Seconds() /
					float64(h.Agents.AgentCache()[name].ReportInterval)
				if int(deltaInIntervals) > (h.Metrics[name].ErrorsFromLastReport + 1) {
					UpdateAgentBaseMetrics(h.Metrics[name], false, true)
				}
			}
		}
	}
}
