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

	//"k8s.io/apimachinery/pkg/apis/meta/v1"
	//meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/unversioned"
	"k8s.io/client-go/pkg/api/meta"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/runtime"
)

const (
	GroupName string = "network-checker.ext"
	Version   string = "v1"
)

var (
	SchemeGroupVersion = unversioned.GroupVersion{Group: GroupName, Version: Version}
	SchemeBuilder      = runtime.NewSchemeBuilder(addKnownTypes)
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(
		SchemeGroupVersion,
		&Agent{},
		&AgentList{},

		&v1.ListOptions{},
		&v1.DeleteOptions{},
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
	unversioned.TypeMeta `json:",inline"`
	Metadata         api.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec AgentSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

type AgentList struct {
	unversioned.TypeMeta `json:",inline"`
	Metadata         unversioned.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Items []Agent `json:"items" protobuf:"bytes,2,rep,name=items"`
}

func (e *Agent) GetObjectKind() unversioned.ObjectKind {
	return &e.TypeMeta
}

func (e *Agent) GetObjectMeta() meta.Object {
	return &e.Metadata
}

//func (el *AgentList) GetObjectKind() unversioned.ObjectKind {
//	return &el.TypeMeta
//}

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
