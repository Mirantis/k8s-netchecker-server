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
	"reflect"
	ext_v1 "github.com/Mirantis/k8s-netchecker-server/pkg/extensions/apis/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateAgentCustomResourceDefinition is a function to initialize schema for custom reource
func CreateAgentCustomResourceDefinition(clientset apiextensionsclient.Interface) error {
	agent := &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: ext_v1.AgentResourcePlural + "." + ext_v1.GroupName,
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   ext_v1.GroupName,
			Version: ext_v1.SchemeGroupVersion.Version,
			Scope:   apiextensionsv1beta1.NamespaceScoped,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Plural: ext_v1.AgentResourcePlural,
				Kind:   reflect.TypeOf(ext_v1.Agent{}).Name(),
			},
		},
	}
	_, err := clientset.ApiextensionsV1beta1().
		CustomResourceDefinitions().
		Create(agent)
	return err
}
