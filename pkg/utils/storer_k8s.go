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

	ext_v1 "github.com/Mirantis/k8s-netchecker-server/pkg/extensions/apis/v1"
	ext_client "github.com/Mirantis/k8s-netchecker-server/pkg/extensions/client"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type k8sAgentStorage struct {
	NcAgentCache        NcAgentCache
	KubeClient          Proxy
	ExtensionsClientset ext_client.Clientset
}

func connect2k8s(createTPR bool) (Proxy, ext_client.Clientset, error) {
	var err error
	var clientset *kubernetes.Clientset

	proxy := &KubeProxy{}

	config, err := proxy.buildConfig()
	if err != nil {
		glog.Error(err)
		return nil, nil, err
	}

	clientset, err = proxy.SetupClientSet(config)
	if err != nil {
		glog.Error(err)
		return nil, nil, err
	}

	if !createTPR {
		return proxy, nil, err
	}

	err = ext_client.CreateAgentThirdPartyResource(clientset)
	if err != nil && !api_errors.IsAlreadyExists(err) {
		glog.Error(err)
		return nil, nil, err
	}

	ext, err := ext_client.WrapClientsetWithExtensions(clientset, config)
	if err != nil {
		glog.Error(err)
		return nil, nil, err
	}

	return proxy, ext, err
}

func NewK8sStorer() (*k8sAgentStorage, error) {
	var err error

	rv := &k8sAgentStorage{
		NcAgentCache: map[string]ext_v1.AgentSpec{},
	}

	rv.KubeClient, rv.ExtensionsClientset, err = connect2k8s(true)

	return rv, err
}

func (h *k8sAgentStorage) UpdateAgents(rw http.ResponseWriter, r *http.Request, rp httprouter.Params) (ext_v1.AgentSpec, error) {
	var err error
	agentData := ext_v1.AgentSpec{}

	if err = ProcessRequest(r, &agentData, rw); err != nil {
		return ext_v1.AgentSpec{}, err
	}

	agentData.LastUpdated = time.Now()
	glog.V(10).Infof("Updating the agents resource with value: %v", agentData)

	agentName := rp.ByName("name")

	_, err = h.ExtensionsClientset.Agents().Get(agentName)

	if err != nil {
		glog.Error(err)
	}

	agent := &ext_v1.Agent{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: agentName,
		},
		Spec: agentData,
	}

	if api_errors.IsNotFound(err) {
		h.CleanCacheOnDemand(nil)
		agent, err = h.ExtensionsClientset.Agents().Create(agent)
		glog.Infoln("Created agent", agentName, err)
	} else {
		agent, err = h.ExtensionsClientset.Agents().Update(agent)
		glog.Infoln("Updated agent", agentName, err)
	}

	if err != nil {
		glog.Error(err)
	}

	h.NcAgentCache[agentName] = agent.Spec

	return agentData, nil
}

func (h *k8sAgentStorage) GetAgents(rw http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	agentsData := map[string]ext_v1.AgentSpec{}
	agents, err := h.ExtensionsClientset.Agents().List()

	if err != nil {
		glog.Error(err)
	}

	for _, agent := range agents.Items {
		agentsData[agent.ObjectMeta.Name] = agent.Spec
	}

	ProcessResponse(rw, agentsData)
}

func (h *k8sAgentStorage) GetSingleAgent(rw http.ResponseWriter, r *http.Request, rp httprouter.Params) {
	agentName := rp.ByName("name")
	agent, err := h.ExtensionsClientset.Agents().Get(agentName)

	if err != nil {
		glog.Error(err)
		return
	}

	if api_errors.IsNotFound(err) {
		glog.V(5).Infof("Agent with name %v is not found in the cache", agentName)
		http.Error(rw, "There is no such entry in the agent cache", http.StatusNotFound)
		return
	}

	ProcessResponse(rw, agent)
}

func (h *k8sAgentStorage) CheckAgents() ([]string, []string, error) {
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

		if api_errors.IsNotFound(err) {
			absent = append(absent, agentName)
			continue
		}

		if err != nil {
			return nil, nil, err
		}

		delta := time.Now().Sub(agent.Spec.LastUpdated).Seconds()
		if delta > float64(agent.Spec.ReportInterval*2) {
			outdated = append(outdated, agentName)
		}
	}

	return absent, outdated, nil
}

func (h *k8sAgentStorage) AgentCache() NcAgentCache {
	return h.NcAgentCache
}

func (h *k8sAgentStorage) AgentCacheUpdate(key string, ag *ext_v1.AgentSpec) {
	// Required for tests
	h.NcAgentCache[key] = *ag
}

func (h *k8sAgentStorage) SetKubeClient(cl Proxy) {
	// Required for tests
	h.KubeClient = cl
}

func (h *k8sAgentStorage) CleanCacheOnDemand(rw http.ResponseWriter) {
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

		for agentName := range h.NcAgentCache {
			if _, exists := podMap[agentName]; !exists {
				toRemove = append(toRemove, agentName)
			}
		}

		glog.V(5).Infof("Data cache for agents %v is to be cleaned up.", toRemove)
		for _, agentName := range toRemove {
			delete(h.NcAgentCache, agentName)
			// delete(h.Metrics, agentName)
		}
	}
}
