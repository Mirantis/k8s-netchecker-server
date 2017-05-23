// Copyright 2017 Mirantis
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1

import (
	"time"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AgentResourcePlural is a constant for plural form naming
const AgentResourcePlural = "agents"

// AgentSpec is a payload to keep Agent info
type AgentSpec struct {
	ReportInterval int                 `json:"report_interval"`
	NodeName       string              `json:"nodename"`
	PodName        string              `json:"podname"`
	HostDate       time.Time           `json:"hostdate"`
	LastUpdated    time.Time           `json:"last_updated"`
	LookupHost     map[string][]string `json:"nslookup"`
	NetworkProbes  []ProbeResult       `json:"network_probes"`
	IPs            map[string][]string `json:"ips"`
}

// ProbeResult structure for network probing results
type ProbeResult struct {
	URL              string
	ConnectionResult int
	HTTPCode         int
	Total            int
	ContentTransfer  int
	TCPConnection    int
	DNSLookup        int
	Connect          int
	ServerProcessing int
}

// AgentStatus is a payload to keep agent status and message
type AgentStatus struct {
	State   AgentState `json:"state,omitempty"`
	Message string     `json:"message,omitempty"`
}

// AgentState type
type AgentState string

const (
	// AgentStateCreated constant
	AgentStateCreated AgentState = "Created"
	// AgentStateProcessed constant
	AgentStateProcessed AgentState = "Processed"
)

// Agent struct to store AgentSpec info as json
type Agent struct {
	meta_v1.TypeMeta   `json:",inline"`
	meta_v1.ObjectMeta `json:"metadata"`
	Spec               AgentSpec   `json:"spec"`
	Status             AgentStatus `json:"status,omitempty"`
}

// AgentList struct to store many of agents
type AgentList struct {
	meta_v1.TypeMeta `json:",inline"`
	meta_v1.ListMeta `json:"metadata"`
	Items            []Agent `json:"items"`
}
