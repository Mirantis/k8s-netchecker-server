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

package extensions

import (
	"encoding/json"
	"time"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	GroupName string = "network-checker.ext"
	Version   string = "v1"
)

var (
	SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: Version}
	SchemeBuilder      = runtime.NewSchemeBuilder(addKnownTypes)
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(
		SchemeGroupVersion,
		&Agent{},
		&AgentList{},

		&meta_v1.ListOptions{},
		&meta_v1.DeleteOptions{},
	)
	return nil
}

type AgentSpec struct {
	ReportInterval int                 `json:"report_interval"`
	PodName        string              `json:"podname"`
	HostDate       time.Time           `json:"hostdate"`
	LastUpdated    time.Time           `json:"last_updated"`
	LookupHost     map[string][]string `json:"nslookup"`
	IPs            map[string][]string `json:"ips"`
}

type Agent struct {
	meta_v1.TypeMeta `json:",inline"`
	Metadata meta_v1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec AgentSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

type AgentList struct {
	meta_v1.TypeMeta `json:",inline"`
	Metadata meta_v1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Items []Agent `json:"items" protobuf:"bytes,2,rep,name=items"`
}


func (e *Agent) GetObjectKind() schema.ObjectKind {
	return &e.TypeMeta
}

func (e *Agent) GetObjectMeta() meta_v1.Object {
	return &e.Metadata
}

func (el *AgentList) GetObjectKind() schema.ObjectKind {
	return &el.TypeMeta
}

func (el *AgentList) GetListMeta() meta_v1.List {
	return &el.Metadata
}


type AgentCopy Agent
type AgentListCopy AgentList


func (e *Agent) UnmarshalJSON(data []byte) error {
	tmp := AgentCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
			return err
	}
	tmp2 := Agent(tmp)
	*e = tmp2
	return nil
}

func (el *AgentList) UnmarshalJSON(data []byte) error {
	tmp := AgentListCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
			return err
	}
	tmp2 := AgentList(tmp)
	*el = tmp2
	return nil
}
