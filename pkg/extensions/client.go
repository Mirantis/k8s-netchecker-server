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

package extensions

import (
	"bytes"
	"encoding/json"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/watch"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/rest"
)

// WrapClientsetWithExtensions function
func WrapClientsetWithExtensions(clientset *kubernetes.Clientset, config *rest.Config) (*WrappedClientset, error) {
	restConfig := &rest.Config{}
	*restConfig = *config
	rest, err := extensionClient(restConfig)
	if err != nil {
		return nil, err
	}
	return &WrappedClientset{
		Client: rest,
	}, nil
}

func extensionClient(config *rest.Config) (*rest.RESTClient, error) {
	config.APIPath = "/apis"
	config.ContentConfig = rest.ContentConfig{
		GroupVersion: &schema.GroupVersion{
			Group:   GroupName,
			Version: Version,
		},
		NegotiatedSerializer: serializer.DirectCodecFactory{CodecFactory: api.Codecs},
		ContentType:          runtime.ContentTypeJSON,
	}
	return rest.RESTClientFor(config)
}

// Clientset interface
type Clientset interface {
	Agents() AgentsInterface
}

// WrappedClientset structure
type WrappedClientset struct {
	Client *rest.RESTClient
}

// AgentsInterface interface
type AgentsInterface interface {
	Create(*Agent) (*Agent, error)
	Get(name string) (*Agent, error)
	List(api.ListOptions) (*AgentList, error)
	Watch(api.ListOptions) (watch.Interface, error)
	Update(*Agent) (*Agent, error)
	Delete(string, *api.DeleteOptions) error
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
func (c *AgentsClient) Create(agent *Agent) (result *Agent, err error) {
	result = &Agent{}
	resp, err := c.client.Post().
		Namespace(api.NamespaceDefault).
		Resource("agents").
		Body(agent).
		DoRaw()
	if err != nil {
		return result, err
	}
	return result, decodeResponseInto(resp, result)
}

// List agents function
func (c *AgentsClient) List(opts api.ListOptions) (result *AgentList, err error) {
	result = &AgentList{}
	resp, err := c.client.Get().
		Namespace(api.NamespaceDefault).
		Resource("agents").
		LabelsSelectorParam(opts.LabelSelector).
		DoRaw()
	if err != nil {
		return result, err
	}
	return result, decodeResponseInto(resp, result)
}

// Watch agents function
func (c *AgentsClient) Watch(opts api.ListOptions) (watch.Interface, error) {
	return c.client.Get().
		Namespace(api.NamespaceDefault).
		Prefix("watch").
		Resource("agents").
		VersionedParams(&opts, api.ParameterCodec).
		Watch()
}

// Update agents function
func (c *AgentsClient) Update(agent *Agent) (result *Agent, err error) {
	result = &Agent{}
	resp, err := c.client.Put().
		Namespace(api.NamespaceDefault).
		Resource("agents").
		Name(agent.Metadata.Name).
		Body(agent).
		DoRaw()
	if err != nil {
		return result, err
	}
	return result, decodeResponseInto(resp, result)
}

// Delete agent function
func (c *AgentsClient) Delete(name string, options *api.DeleteOptions) error {
	return c.client.Delete().
		Namespace(api.NamespaceDefault).
		Resource("agents").
		Name(name).
		Body(options).
		Do().
		Error()
}

// Get agent function
func (c *AgentsClient) Get(name string) (result *Agent, err error) {
	result = &Agent{}
	resp, err := c.client.Get().
		Namespace(api.NamespaceDefault).
		Resource("agents").
		Name(name).
		DoRaw()
	if err != nil {
		return result, err
	}
	return result, decodeResponseInto(resp, result)
}
