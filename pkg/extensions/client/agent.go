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

    apierrors "k8s.io/apimachinery/pkg/api/errors"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/util/wait"
    agentv1 "github.com/Mirantis/k8s-netchecker-server/pkg/extensions/apis/agent/v1"
    "k8s.io/client-go/kubernetes"
    apiv1 "k8s.io/client-go/pkg/api/v1"
    "k8s.io/client-go/pkg/apis/extensions/v1beta1"
    "k8s.io/client-go/rest"
    // Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
    // _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func CreateTPR(clientset kubernetes.Interface) error {
    agent := &v1beta1.ThirdPartyResource{
        ObjectMeta: metav1.ObjectMeta{
            Name: "agent." + agentv1.GroupName,
        },
        Versions: []v1beta1.APIVersion{
            {Name: agentv1.SchemeGroupVersion.Version},
        },
        Description: "An Agent ThirdPartyResource",
    }
    _, err := clientset.ExtensionsV1beta1().ThirdPartyResources().Create(agent)
    return err
}

func WaitForAgentResource(agentClient *rest.RESTClient) error {
    return wait.Poll(100*time.Millisecond, 60*time.Second, func() (bool, error) {
        _, err := agentClient.Get().
            Resource(agentv1.AgentResourcePlural).
            Namespace(apiv1.NamespaceDefault).
            DoRaw()

        if err == nil {
            return true, nil
        }
        if apierrors.IsNotFound(err) {
            return false, nil
        }
        return false, err
    })
}

func WaitForAgentInstanceProcessed(exampleClient *rest.RESTClient, name string) error {
    return wait.Poll(100*time.Millisecond, 10*time.Second, func() (bool, error) {
        var agent agentv1.Agent
        err := exampleClient.Get().
            Resource(agentv1.AgentResourcePlural).
            Namespace(apiv1.NamespaceDefault).
            Name(name).
            Do().Into(&agent)

        if err == nil && agent.Status.State == agentv1.AgentStateProcessed {
            return true, nil
        }

        return false, err
    })
}
