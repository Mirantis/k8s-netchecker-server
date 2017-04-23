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

    "github.com/Mirantis/k8s-netchecker-server/pkg/extensions"

    meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/api/errors"
    "k8s.io/apimachinery/pkg/labels"
    "k8s.io/apimachinery/pkg/runtime"
    "k8s.io/apimachinery/pkg/runtime/schema"
    "k8s.io/apimachinery/pkg/runtime/serializer"
    "k8s.io/apimachinery/pkg/selection"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/pkg/api"
    "k8s.io/client-go/pkg/api/v1"
    "k8s.io/client-go/pkg/apis/extensions/v1beta1"
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

func (kp *KubeProxy) SetupClientSet() error {
	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}
	clientSet, err := kubernetes.NewForConfig(config)

	if err != nil {
		return err
	}

	kp.Client = clientSet
	return nil
}

func (kp *KubeProxy) initThirdParty() error {
	tpr, err := kp.Client.ExtensionsV1beta1().ThirdPartyResources().Get("agent.network-checker.ext", meta_v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			tpr := &v1beta1.ThirdPartyResource{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: "agent.network-checker.ext",
				},
				Versions: []v1beta1.APIVersion{
					{Name: "v1"},
				},
				Description: "Agent ThirdPartyResource",
			}
			result, err := kp.Client.ExtensionsV1beta1().ThirdPartyResources().Create(tpr)
			if err != nil {
				return err
			}
			glog.V(5).Infof("CREATED: %#v\nFROM: %#v\n", result, tpr)
		} else {
			return err
		}
	} else {
		glog.V(5).Infof("SKIPPING: already exists %#v\n", tpr)
	}

	return err
}

func configureClient(config *rest.Config) {
    groupversion := schema.GroupVersion{
        Group:   "network-checker.ext",
        Version: "v1",
    }

    config.GroupVersion = &groupversion
    config.APIPath = "/apis"
    config.ContentType = runtime.ContentTypeJSON
    config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: api.Codecs}

    schemeBuilder := runtime.NewSchemeBuilder(
        func(scheme *runtime.Scheme) error {
            scheme.AddKnownTypes(
                groupversion,
                &extensions.Agent{},
                &extensions.AgentList{},
            )
            return nil
        })
    meta_v1.AddToGroupVersion(api.Scheme, groupversion)
    schemeBuilder.AddToScheme(api.Scheme)
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
