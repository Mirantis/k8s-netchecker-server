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

package client

import (
	"time"

	api_errors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	api_v1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/rest"

	ext_v1 "github.com/Mirantis/k8s-netchecker-server/pkg/extensions/apis/v1"
)

// CreateAgentThirdPartyResource is a function to initialize schema for 3rd-party resource
func CreateAgentThirdPartyResource(clientset kubernetes.Interface) error {
	agent := &v1beta1.ThirdPartyResource{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "agents." + ext_v1.GroupName,
		},
		Versions: []v1beta1.APIVersion{
			{Name: ext_v1.SchemeGroupVersion.Version},
		},
		Description: "An Agent ThirdPartyResource",
	}

	_, err := clientset.ExtensionsV1beta1().
		ThirdPartyResources().
		Create(agent)

	return err
}

// WaitForAgentResource is a handler to check 3rd-party resource exist
func WaitForAgentResource(client *rest.RESTClient) error {
	return wait.Poll(100*time.Millisecond, 60*time.Second, func() (bool, error) {
		_, err := client.Get().
			Namespace(api_v1.NamespaceDefault).
			Resource(ext_v1.AgentResourcePlural).
			DoRaw()

		if err == nil {
			return true, nil
		}

		if api_errors.IsNotFound(err) {
			return false, nil
		}

		return false, err
	})
}

// WaitForAgentInstanceProcessed is a handler to check instance of 3rd-party resource created
func WaitForAgentInstanceProcessed(ext Clientset, name string) error {
	return wait.Poll(100*time.Millisecond, 10*time.Second, func() (bool, error) {
		agent, err := ext.Agents().Get(name)

		if err == nil && agent.Status.State == ext_v1.AgentStateProcessed {
			return true, nil
		}

		return false, err
	})
}
