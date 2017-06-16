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
	"github.com/golang/glog"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
)

const AgentLabelKey = "app"

var AgentLabelValues = []string{"netchecker-agent", "netchecker-agent-hostnet"}

type Proxy interface {
	Pods() (*v1.PodList, error)
}

type KubeProxy struct {
	Client kubernetes.Interface
}

// SetupClientSet is a function for initialize kubernetes clientset
func (kp *KubeProxy) SetupClientSet(config *rest.Config) (*kubernetes.Clientset, error) {
	clientSet, err := kubernetes.NewForConfig(config)

	if err != nil {
		return nil, err
	}

	kp.Client = clientSet

	return clientSet, nil
}

func (kp *KubeProxy) buildConfig() (*rest.Config, error) {
	return rest.InClusterConfig()
}

func (kp *KubeProxy) Pods() (*v1.PodList, error) {
	requirement, err := labels.NewRequirement(AgentLabelKey, selection.In, AgentLabelValues)
	if err != nil {
		return nil, err
	}
	glog.V(10).Infof("Selector for kubernetes pods: %v", requirement.String())

	pods, err := kp.Client.Core().Pods("").List(meta_v1.ListOptions{LabelSelector: requirement.String()})
	return pods, err
}
