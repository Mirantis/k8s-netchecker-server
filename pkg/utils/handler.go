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
	"github.com/urfave/negroni"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Handler struct {
	AgentCache  map[string]AgentInfo
	Metrics     map[string]AgentMetrics
	KubeClient  Proxy
	HTTPHandler http.Handler
}

func NewHandler(createKubeClient bool) (*Handler, error) {
	h := &Handler{
		AgentCache: map[string]AgentInfo{},
		Metrics: map[string]AgentMetrics{},
	}

	var err error
	if createKubeClient {
		kProxy := &KubeProxy{}
		err = kProxy.SetupClientSet()
		if err == nil {
			h.KubeClient = kProxy
		}
	}

	h.SetupRouter()
	h.AddMiddleware()

	return h, err
}

func (h *Handler) SetupRouter() {
	glog.V(10).Info("Setting up the url multiplexer")
	router := httprouter.New()
	router.POST("/api/v1/agents/:name", h.UpdateAgents)
	router.GET("/api/v1/agents/:name", h.CleanCache(h.GetSingleAgent))
	router.GET("/api/v1/agents/", h.CleanCache(h.GetAgents))
	router.GET("/api/v1/connectivity_check", h.CleanCache(h.ConnectivityCheck))
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
	agentData := AgentInfo{}
	if err := ProcessRequest(r, &agentData, rw); err != nil {
		return
	}

	agentData.LastUpdated = time.Now()
	glog.V(10).Infof("Updating the agents cache with value: %v", agentData)
	agent_name := rp.ByName("name")
	if _, exists := h.AgentCache[agent_name]; !exists {
		h.Metrics[agent_name] = NewAgentMetrics(&agentData)
	}
	UpdateAgentMetrics(h.Metrics[agent_name], true, false)
	h.AgentCache[agent_name] = agentData
}

func (h *Handler) GetAgents(rw http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ProcessResponse(rw, h.AgentCache)
}

func (h *Handler) GetSingleAgent(rw http.ResponseWriter, r *http.Request, rp httprouter.Params) {
	aName := rp.ByName("name")
	aData, exists := h.AgentCache[aName]
	if !exists {
		glog.V(5).Infof("Agent with name %v is not found in the cache", aName)
		http.Error(rw, "There is no such entry in the agent cache", http.StatusNotFound)
		return
	}
	ProcessResponse(rw, aData)
}

func (h *Handler) ConnectivityCheck(rw http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	res := &CheckConnectivityInfo{
		Message: fmt.Sprintf(
			"All %v pods successfully reported back to the server",
			len(h.AgentCache)),
	}
	status := http.StatusOK
	errMsg := "Connectivity check fails. Reason: %v"

	absent, outdated, err := h.CheckAgents()
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

func (h *Handler) CheckAgents() ([]string, []string, error) {
	if h.KubeClient == nil {
		return nil, nil, nil
	}

	absent := []string{}
	outdated := []string{}

	pods, err := h.KubeClient.Pods()
	if err != nil {
		return nil, nil, err
	}
	for _, pod := range pods.Items {
		agentName := pod.ObjectMeta.Name
		agentData, exists := h.AgentCache[agentName]
		if !exists {
			absent = append(absent, agentName)
			continue
		}

		delta := time.Now().Sub(agentData.LastUpdated).Seconds()
		if delta > float64(agentData.ReportInterval) {
			outdated = append(outdated, agentName)
		}
	}

	return absent, outdated, nil
}

func (h *Handler) CleanCache(handle httprouter.Handle) httprouter.Handle {
	return func(rw http.ResponseWriter, r *http.Request, rp httprouter.Params) {
		if h.KubeClient != nil {
			pods, err := h.KubeClient.Pods()
			if err != nil {
				msg := fmt.Sprintf("Failed to get pods from k8s cluster. Details: %v", err)
				glog.Error(msg)
				http.Error(rw, msg, http.StatusInternalServerError)
				return
			}

			type empty struct{}

			podMap := make(map[string]empty)
			toRemove := []string{}

			for _, pod := range pods.Items {
				podMap[pod.ObjectMeta.Name] = empty{}
			}

			for agentName, _ := range h.AgentCache {
				if _, exists := podMap[agentName]; !exists {
					toRemove = append(toRemove, agentName)
				}
			}

			for _, agentName := range toRemove {
				delete(h.AgentCache, agentName)
				delete(h.Metrics, agentName)
			}
		}

		handle(rw, r, rp)
	}
}

func (h *Handler) CollectAgentsMetrics() {
	for {
		time.Sleep(5 * time.Second)
		for name, _ := range h.AgentCache {
			if _, exists := h.Metrics[name]; exists {
				deltaInIntervals := time.Now().Sub(h.AgentCache[name].LastUpdated).Seconds() /
						float64(h.AgentCache[name].ReportInterval)
				if int(deltaInIntervals) > h.Metrics[name].ErrorsFromLastReport {
					UpdateAgentMetrics(h.Metrics[name], false, true)
				}
			}
		}
	}
}
