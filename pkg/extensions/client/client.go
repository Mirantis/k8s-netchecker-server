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
    "bytes"
    "encoding/json"

    "k8s.io/apimachinery/pkg/runtime"
    "k8s.io/apimachinery/pkg/runtime/serializer"
    api_v1 "k8s.io/client-go/pkg/api/v1"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/rest"

    ext_v1 "github.com/Mirantis/k8s-netchecker-server/pkg/extensions/apis/v1"
)

func WrapClientsetWithExtensions(clientset *kubernetes.Clientset, config *rest.Config) (*WrappedClientset, error) {
    restConfig := &rest.Config{}
    *restConfig = *config
    rest, scheme, err := ExtensionClient(restConfig)
    if err != nil {
        return nil, err
    }
    return &WrappedClientset{
        Client: rest,
        Scheme: scheme,
    }, nil
}

func ExtensionClient(cfg *rest.Config) (*rest.RESTClient, *runtime.Scheme, error) {
    scheme := runtime.NewScheme()
    if err := ext_v1.AddToScheme(scheme); err != nil {
        return nil, nil, err
    }

    config := *cfg
    config.GroupVersion = &ext_v1.SchemeGroupVersion
    config.APIPath = "/apis"
    config.ContentType = runtime.ContentTypeJSON
    config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: serializer.NewCodecFactory(scheme)}

    client, err := rest.RESTClientFor(&config)
    if err != nil {
        return nil, nil, err
    }

    return client, scheme, nil
}

// Clientset interface
type Clientset interface {
    Agents() AgentsInterface
}

// WrappedClientset structure
type WrappedClientset struct {
    Client *rest.RESTClient
    Scheme *runtime.Scheme
}

// AgentsInterface interface
type AgentsInterface interface {
    Create(*ext_v1.Agent) (*ext_v1.Agent, error)
    Get(name string) (*ext_v1.Agent, error)
    List() (*ext_v1.AgentList, error)
    Update(*ext_v1.Agent) (*ext_v1.Agent, error)
    Delete(string, *api_v1.DeleteOptions) error
}

// Agents function
func (w *WrappedClientset) Agents() AgentsInterface {
    return &AgentsClient{w.Client}
}

// AgentsClient structure
type AgentsClient struct {
    client *rest.RESTClient
}

func decodeResponseInto(resp []byte, obj interface{}) error {
    return json.NewDecoder(bytes.NewReader(resp)).Decode(obj)
}

// Create agent function
func (c *AgentsClient) Create(agent *ext_v1.Agent) (result *ext_v1.Agent, err error) {
    result = &ext_v1.Agent{}
    resp, err := c.client.Post().
        Namespace(api_v1.NamespaceDefault).
        Resource(ext_v1.AgentResourcePlural).
        Body(agent).
        DoRaw()
    if err != nil {
        return result, err
    }
    return result, decodeResponseInto(resp, result)
}

// List agents function
func (c *AgentsClient) List() (result *ext_v1.AgentList, err error) {
    result = &ext_v1.AgentList{}
    resp, err := c.client.Get().
        Namespace(api_v1.NamespaceDefault).
        Resource(ext_v1.AgentResourcePlural).
        DoRaw()
    if err != nil {
        return result, err
    }
    return result, decodeResponseInto(resp, result)
}

// Update agents function
func (c *AgentsClient) Update(agent *ext_v1.Agent) (result *ext_v1.Agent, err error) {
    result = &ext_v1.Agent{}
    resp, err := c.client.Put().
        Namespace(api_v1.NamespaceDefault).
        Resource(ext_v1.AgentResourcePlural).
        Name(agent.ObjectMeta.Name).
        Body(agent).
        DoRaw()
    if err != nil {
        return result, err
    }
    return result, decodeResponseInto(resp, result)
}

// Delete agent function
func (c *AgentsClient) Delete(name string, options *api_v1.DeleteOptions) error {
    return c.client.Delete().
        Namespace(api_v1.NamespaceDefault).
        Resource(ext_v1.AgentResourcePlural).
        Name(name).
        Body(options).
        Do().
        Error()
}

// Get agent function
func (c *AgentsClient) Get(name string) (result *ext_v1.Agent, err error) {
    result = &ext_v1.Agent{}
    resp, err := c.client.Get().
        Namespace(api_v1.NamespaceDefault).
        Resource(ext_v1.AgentResourcePlural).
        Name(name).
        DoRaw()
    if err != nil {
        return result, err
    }
    return result, decodeResponseInto(resp, result)
}
