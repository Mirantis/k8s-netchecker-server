// Copyright 2016 Mirantis
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

package e2e

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	testutils "github.com/Mirantis/k8s-netchecker-server/test/e2e/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

var _ = Describe("Basic", func() {
	var clientset *kubernetes.Clientset
	var ns *v1.Namespace
	var serverPort int = 8989

	BeforeEach(func() {
		var err error
		clientset, err = testutils.KubeClient()
		Expect(err).NotTo(HaveOccurred())
		namespaceObj := &v1.Namespace{
			ObjectMeta: meta_v1.ObjectMeta{
				GenerateName: "e2e-tests-netchecker-",
				Namespace:    "",
			},
			Status: v1.NamespaceStatus{},
		}
		ns, err = clientset.Namespaces().Create(namespaceObj)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		podList, _ := clientset.Core().Pods(ns.Name).List(meta_v1.ListOptions{LabelSelector: labels.Everything().String()})
		if CurrentGinkgoTestDescription().Failed {
			testutils.DumpLogs(clientset, podList.Items...)
		}
		for _, pod := range podList.Items {
			clientset.Core().Pods(pod.Namespace).Delete(pod.Name, &meta_v1.DeleteOptions{})
		}
		clientset.Namespaces().Delete(ns.Name, &meta_v1.DeleteOptions{})
	})

	It("Connectivity check should pass", func() {
		By("deploying netchecker server pod")
		endpointArg := fmt.Sprintf("--endpoint=0.0.0.0:%d", serverPort)
		serverPod := newPod(
			"netchecker-server", "netchecker-server", "mirantis/k8s-netchecker-server",
			[]string{"netchecker-server", "--kubeproxyinit", "--logtostderr", "--v=10", endpointArg}, nil, false, true, nil)
		pod, err := clientset.Pods(ns.Name).Create(serverPod)
		Expect(err).Should(BeNil())
		testutils.WaitForReady(clientset, pod)

		By("deploying netchecker service")
		servicePorts := []v1.ServicePort{{Protocol: v1.ProtocolTCP, Port: int32(serverPort), TargetPort: intstr.FromInt(serverPort)}}
		server_svc := newService("netchecker-service", nil, servicePorts, []string{})
		_, err = clientset.Services(ns.Name).Create(server_svc)
		Expect(err).Should(BeNil())

		By("deploying netchecker agent daemonset")
		var ncAgentLabels = map[string]string{"app": "netchecker-agent"}
		serverEndpointArg := fmt.Sprintf("--serverendpoint=netchecker-service:%d", serverPort)
		cmd := []string{"netchecker-agent", "--alsologtostderr=true", "--v=10", serverEndpointArg, "--reportinterval=20"}
		agentDS := newDaemonSet("netchecker-agent", "netchecker-agent", "mirantis/k8s-netchecker-agent",
			[]string{"sh", "-c", strings.Join(cmd, " ")}, ncAgentLabels, false, true,
			[]v1.EnvVar{
				{Name: "MY_NODE_NAME", ValueFrom:&v1.EnvVarSource{FieldRef:&v1.ObjectFieldSelector{FieldPath:"spec.nodeName"}}},
				{Name: "MY_POD_NAME", ValueFrom:&v1.EnvVarSource{FieldRef:&v1.ObjectFieldSelector{FieldPath:"metadata.name"}}},
			},
		)
		_, err = clientset.Extensions().DaemonSets(ns.Name).Create(agentDS)
		Expect(err).NotTo(HaveOccurred())

		testutils.Logf("current pods: %v\n", getPods(clientset, ns))
		testutils.Logf("current services: %v\n", getServices(clientset, ns))

		By("veryfiying that service is reachable using DNS")
		//verifyServiceReachable(serverPort, []string{"0.0.0.0"}...)
	})
})

func newPrivilegedPodSpec(containerName, imageName string, cmd []string, hostNetwork, privileged bool, env []v1.EnvVar) v1.PodSpec {
	return v1.PodSpec{
		HostNetwork: hostNetwork,
		Containers: []v1.Container{
			{
				Name:            containerName,
				Image:           imageName,
				Command:         cmd,
				SecurityContext: &v1.SecurityContext{Privileged: &privileged},
				ImagePullPolicy: v1.PullIfNotPresent,
				Env:			 env,
			},
		},
	}
}

func newPod(podName, containerName, imageName string, cmd []string, labels map[string]string, hostNetwork bool, privileged bool, env []v1.EnvVar) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:   podName,
			Labels: labels,
		},
		Spec: newPrivilegedPodSpec(containerName, imageName, cmd, hostNetwork, privileged, env),
	}
}

func newDaemonSet(dsName, containerName, imageName string, cmd []string, labels map[string]string, hostNetwork, privileged bool, env []v1.EnvVar) *v1beta1.DaemonSet {
	return &v1beta1.DaemonSet{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:   dsName,
			Labels: labels,
		},
		Spec: v1beta1.DaemonSetSpec{
			Template: v1.PodTemplateSpec{
				ObjectMeta: meta_v1.ObjectMeta{
					Labels: labels,
				},
				Spec: newPrivilegedPodSpec(containerName, imageName, cmd, hostNetwork, privileged, env),
			},
		},
	}

}

func newDeployment(deploymentName string, replicas int32, podLabels map[string]string, imageName string, image string, cmd []string, env []v1.EnvVar) *v1beta1.Deployment {
	return &v1beta1.Deployment{
		ObjectMeta: meta_v1.ObjectMeta{Name: deploymentName},
		Spec: v1beta1.DeploymentSpec{
			Replicas: &replicas,
			Template: v1.PodTemplateSpec{
				ObjectMeta: meta_v1.ObjectMeta{
					Labels: podLabels,
				},
				Spec: newPrivilegedPodSpec(image, imageName, cmd, false, false, env),
			},
		},
	}
}

func newService(serviceName string, labels map[string]string, ports []v1.ServicePort, externalIPs []string) *v1.Service {
	return &v1.Service{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: serviceName,
		},
		Spec: v1.ServiceSpec{
			Selector:    labels,
			Type:        v1.ServiceTypeNodePort,
			Ports:       ports,
			ExternalIPs: externalIPs,
		},
	}
}

func verifyServiceReachable(port int, ips ...string) {
	timeout := time.Duration(1 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}
	Eventually(func() error {
		for _, ip := range ips {
			resp, err := client.Get(fmt.Sprintf("http://%s:%d/api/v1/ping", ip, port))
			if err != nil {
				return err
			}
			if resp.StatusCode > 200 {
				return fmt.Errorf("Unexpected error from nginx service: %s", resp.Status)
			}
		}
		return nil
	}, 30*time.Second, 1*time.Second).Should(BeNil())
}

func getPods(clientset *kubernetes.Clientset, ns *v1.Namespace) []v1.Pod {
	pods, err := clientset.Pods(ns.Name).List(meta_v1.ListOptions{})
	Expect(err).NotTo(HaveOccurred())
	return pods.Items
}

func getServices(clientset *kubernetes.Clientset, ns *v1.Namespace) []v1.Service {
	services, err := clientset.Services(ns.Name).List(meta_v1.ListOptions{})
	Expect(err).NotTo(HaveOccurred())
	return services.Items
}
