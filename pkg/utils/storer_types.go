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
	ext_v1 "github.com/Mirantis/k8s-netchecker-server/pkg/extensions/apis/v1"
	"github.com/julienschmidt/httprouter"
	"net/http"
)

type NcAgentCache map[string]ext_v1.AgentSpec
type NcAgentMetrics map[string]AgentMetrics

type AgentStorer interface {
	UpdateAgents(http.ResponseWriter, *http.Request, httprouter.Params) (ext_v1.AgentSpec, error)
	GetSingleAgent(http.ResponseWriter, *http.Request, httprouter.Params)
	GetAgents(http.ResponseWriter, *http.Request, httprouter.Params)
	CleanCacheOnDemand(http.ResponseWriter)
	CheckAgents() ([]string, []string, error)
	//
	AgentCache() NcAgentCache               // Returns Agent Cache map (RO)
	AgentCacheUpdate(string, *ext_v1.Agent) // (agentName, agent.Spec) may be interface{} should be used, because format is storage-specific
}

type Handler struct {
	Agents      AgentStorer
	Metrics     NcAgentMetrics
	HTTPHandler http.Handler
}
