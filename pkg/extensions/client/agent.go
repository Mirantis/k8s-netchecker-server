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
	ext_v1 "github.com/Mirantis/k8s-netchecker-server/pkg/extensions/apis/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

// CreateAgentThirdPartyResource is a function to initialize schema for 3rd-party resource
func CreateAgentThirdPartyResource(clientset kubernetes.Interface) error {
	agent := &v1beta1.ThirdPartyResource{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "agent." + ext_v1.GroupName,
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
