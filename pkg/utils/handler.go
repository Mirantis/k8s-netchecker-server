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

	"github.com/Mirantis/k8s-netchecker-server/pkg/extensions"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/negroni"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	agentv1 "github.com/Mirantis/k8s-netchecker-server/pkg/extensions/apis/agent/v1"
	agentclient "github.com/Mirantis/k8s-netchecker-server/pkg/extensions/client"
	agentcontroller "github.com/Mirantis/k8s-netchecker-server/pkg/extensions/controller"
)

type Handler struct {
	AgentCache  map[string]extensions.AgentSpec
	Metrics     map[string]AgentMetrics
	KubeClient  Proxy
	HTTPHandler http.Handler
	ExtensionsClientset extensions.Clientset
}

func NewHandler(createKubeClient bool) (*Handler, error) {
	h := &Handler{
		AgentCache: map[string]extensions.AgentSpec{},
		Metrics:    map[string]AgentMetrics{},
	}

	var err error
	var clientset *kubernetes.Clientset

	if createKubeClient {
		proxy := &KubeProxy{}

		config, err := proxy.buildConfig(*kubeconfig)
		if err != nil {
			return nil, err
		}

		clientset, err = proxy.SetupClientSet(config)
		if err == nil {
			h.KubeClient = proxy
		}

		err = proxy.initThirdParty()
		if err != nil {
			return nil, err
		}

		ext, err := extensions.WrapClientsetWithExtensions(clientset, config)
		if err != nil {
			return nil, err
		}
		h.ExtensionsClientset = ext
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
	agentData := extensions.AgentSpec{}
	if err := ProcessRequest(r, &agentData, rw); err != nil {
		return
	}

	agentData.LastUpdated = time.Now()
	glog.V(10).Infof("Updating the agents resource with value: %v", agentData)
	agentName := rp.ByName("name")
	existing_agent, err := h.ExtensionsClientset.Agents().Get(agentName)
	if err != nil {
		glog.V(5).Info(err)
	}
	if existing_agent == nil {
		h.Metrics[agentName] = NewAgentMetrics(&agentData)
		h.cleanCacheOnDemand(nil)
	}

	UpdateAgentBaseMetrics(h.Metrics[agentName], true, false)
	UpdateAgentProbeMetrics(agentData, h.Metrics[agentName])

	agent := &extensions.Agent{
		Metadata: meta_v1.ObjectMeta{Name: agentName},
		Spec: agentData,
	}
	glog.V(5).Info("================== HERE ==================")
	glog.V(5).Info(agent)
	h.ExtensionsClientset.Agents().Update(agent)
}

func (h *Handler) GetAgents(rw http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	agentsData := map[string]extensions.AgentSpec{}

	agents, err := h.ExtensionsClientset.Agents().List(api.ListOptions{})
	if err != nil {
		glog.V(5).Info(err)
	}

	for _, agent := range agents.Items {
		glog.V(5).Info("================== HERE ==================")
		glog.V(5).Info(agent)
		agentsData[agent.Metadata.Name] = agent.Spec
	}

	if err := ProcessResponse(rw, agentsData); err != nil {
		return
	}
}

func (h *Handler) GetSingleAgent(rw http.ResponseWriter, r *http.Request, rp httprouter.Params) {
	agentName := rp.ByName("name")
	agent, err := h.ExtensionsClientset.Agents().Get(agentName)
	if err != nil {
		glog.V(5).Info(err)
		return
	}

	if agent == nil {
		glog.V(5).Infof("Agent with name %v is not found in the cache", agentName)
		http.Error(rw, "There is no such entry in the agent cache", http.StatusNotFound)
		return
	}

	if err := ProcessResponse(rw, agent); err != nil {
		return
	}
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
		agent, err := h.ExtensionsClientset.Agents().Get(agentName)
		if err != nil {
			return nil, nil, err
		}

		if agent == nil {
			absent = append(absent, agentName)
			continue
		}

		delta := time.Now().Sub(agent.Spec.LastUpdated).Seconds()
		if delta > float64(agent.Spec.ReportInterval*2) {
			outdated = append(outdated, agentName)
		}
	}

	return absent, outdated, nil
}

func (h *Handler) cleanCacheOnDemand(rw http.ResponseWriter) {
	if h.KubeClient != nil {
		pods, err := h.KubeClient.Pods()
		if err != nil {
			msg := fmt.Sprintf("Failed to get pods from k8s cluster. Details: %v", err)
			glog.Error(msg)
			if rw != nil {
				http.Error(rw, msg, http.StatusInternalServerError)
			}
			return
		}

		type empty struct{}

		podMap := make(map[string]empty)
		toRemove := []string{}

		for _, pod := range pods.Items {
			podMap[pod.ObjectMeta.Name] = empty{}
		}

		for agentName := range h.AgentCache {
			if _, exists := podMap[agentName]; !exists {
				toRemove = append(toRemove, agentName)
			}
		}

		glog.V(5).Infof("Data cache for agents %v is to be cleaned up.", toRemove)
		for _, agentName := range toRemove {
			delete(h.AgentCache, agentName)
			delete(h.Metrics, agentName)
		}
	}
}

func (h *Handler) CleanCache(handle httprouter.Handle) httprouter.Handle {
	return func(rw http.ResponseWriter, r *http.Request, rp httprouter.Params) {
		h.cleanCacheOnDemand(rw)

		handle(rw, r, rp)
	}
}

func (h *Handler) CollectAgentsMetrics() {
	for {
		time.Sleep(5 * time.Second)
		for name := range h.AgentCache {
			if _, exists := h.Metrics[name]; exists {
				deltaInIntervals := time.Now().Sub(h.AgentCache[name].LastUpdated).Seconds() /
					float64(h.AgentCache[name].ReportInterval)
				if int(deltaInIntervals) > (h.Metrics[name].ErrorsFromLastReport + 1) {
					UpdateAgentBaseMetrics(h.Metrics[name], false, true)
				}
			}
		}
	}
}
