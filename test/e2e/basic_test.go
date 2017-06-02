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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/pkg/labels"
	"k8s.io/client-go/pkg/util/intstr"
)

var _ = Describe("Basic", func() {
	var clientset *kubernetes.Clientset
	var ns *v1.Namespace

	BeforeEach(func() {
		var err error
		clientset, err = testutils.KubeClient()
		Expect(err).NotTo(HaveOccurred())
		namespaceObj := &v1.Namespace{
			ObjectMeta: v1.ObjectMeta{
				GenerateName: "e2e-tests-netchecker-",
				Namespace:    "",
			},
			Status: v1.NamespaceStatus{},
		}
		ns, err = clientset.Namespaces().Create(namespaceObj)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		podList, _ := clientset.Core().Pods(ns.Name).List(v1.ListOptions{LabelSelector: labels.Everything().String()})
		if CurrentGinkgoTestDescription().Failed {
			testutils.DumpLogs(clientset, podList.Items...)
		}
		for _, pod := range podList.Items {
			clientset.Core().Pods(pod.Namespace).Delete(pod.Name, &v1.DeleteOptions{})
		}
		clientset.Namespaces().Delete(ns.Name, &v1.DeleteOptions{})
	})

	It("Connectivity check should pass", func() {
		By("deploying netchecker server pod")
		server_pod := newPod(
			"netchecker-server", "netchecker-server", "mirantis/k8s-netchecker-server",
			[]string{"netchecker-server", "--kubeproxyinit", "--logtostderr", "--v=5", "--endpoint=0.0.0.0:8081"}, nil, false, true)
		pod, err := clientset.Pods(ns.Name).Create(server_pod)
		Expect(err).Should(BeNil())
		testutils.WaitForReady(clientset, pod)

		By("deploying netchecker service")
		servicePorts := []v1.ServicePort{{Protocol: v1.ProtocolTCP, Port: 8081, TargetPort: intstr.FromInt(8081)}}
		server_svc := newService("netchecker-service", nil, servicePorts, []string{})
		_, err = clientset.Services(ns.Name).Create(server_svc)
		Expect(err).Should(BeNil())

		By("deploying netchecker agent daemonset")
		var ncagentLabels = map[string]string{"app": "netchecker-agent"}
		cmd := []string{"netchecker-agent", "--alsologtostderr=true", "--v=5", "--serverendpoint=netchecker-service:8081", "reportinterval=20"}
		agent_ds := newDaemonSet("netchecker-agent", "netchecker-agent", "mirantis/k8s-netchecker-agent",
			[]string{"sh", "-c", strings.Join(cmd, " ")}, ncagentLabels, false, true)
		_, err = clientset.Extensions().DaemonSets(ns.Name).Create(agent_ds)
		Expect(err).NotTo(HaveOccurred())

		By("veryfiying that service is reachable using DNS")
		verifyServiceReachable(8081, []string{"netchecker-service"}...)
	})
})

func newPrivilegedPodSpec(containerName, imageName string, cmd []string, hostNetwork, privileged bool) v1.PodSpec {
	return v1.PodSpec{
		HostNetwork: hostNetwork,
		Containers: []v1.Container{
			{
				Name:            containerName,
				Image:           imageName,
				Command:         cmd,
				SecurityContext: &v1.SecurityContext{Privileged: &privileged},
				ImagePullPolicy: v1.PullIfNotPresent,
			},
		},
	}
}

func newPod(podName, containerName, imageName string, cmd []string, labels map[string]string, hostNetwork bool, privileged bool) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:   podName,
			Labels: labels,
		},
		Spec: newPrivilegedPodSpec(containerName, imageName, cmd, hostNetwork, privileged),
	}
}

func newDaemonSet(dsName, containerName, imageName string, cmd []string, labels map[string]string, hostNetwork, privileged bool) *v1beta1.DaemonSet {
	return &v1beta1.DaemonSet{
		ObjectMeta: v1.ObjectMeta{
			Name:   dsName,
			Labels: labels,
		},
		Spec: v1beta1.DaemonSetSpec{
			Template: v1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: labels,
				},
				Spec: newPrivilegedPodSpec(containerName, imageName, cmd, hostNetwork, privileged),
			},
		},
	}

}

func newDeployment(deploymentName string, replicas int32, podLabels map[string]string, imageName string, image string, cmd []string) *v1beta1.Deployment {
	return &v1beta1.Deployment{
		ObjectMeta: v1.ObjectMeta{Name: deploymentName},
		Spec: v1beta1.DeploymentSpec{
			Replicas: &replicas,
			Template: v1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: podLabels,
				},
				Spec: newPrivilegedPodSpec(image, imageName, cmd, false, false),
			},
		},
	}
}

func newService(serviceName string, labels map[string]string, ports []v1.ServicePort, externalIPs []string) *v1.Service {
	return &v1.Service{
		ObjectMeta: v1.ObjectMeta{
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

func verifyServiceReachable(port int32, ips ...string) {
	timeout := time.Duration(1 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}
	Eventually(func() error {
		for _, ip := range ips {
			resp, err := client.Get(fmt.Sprintf("http://%s:%d", ip, port))
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

func getPodsByLabels(clientset *kubernetes.Clientset, ns *v1.Namespace, podLabels map[string]string) []v1.Pod {
	selector := labels.Set(podLabels).AsSelector().String()
	pods, err := clientset.Pods(ns.Name).List(v1.ListOptions{LabelSelector: selector})
	Expect(err).NotTo(HaveOccurred())
	return pods.Items
}
