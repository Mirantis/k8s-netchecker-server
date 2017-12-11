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

package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Mirantis/k8s-netchecker-server/pkg/utils"
	testutils "github.com/Mirantis/k8s-netchecker-server/test/e2e/utils"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"

	"io/ioutil"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rbac "k8s.io/client-go/pkg/apis/rbac/v1beta1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

var _ = ginkgo.Describe("Basic", func() {
	var clientset *kubernetes.Clientset
	var ns *v1.Namespace
	var cr *rbac.ClusterRole
	var crb *rbac.ClusterRoleBinding
	var serverPort int = 8989

	ginkgo.BeforeEach(func() {
		var err error
		clientset, err = testutils.KubeClient()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		namespaceObj := &v1.Namespace{
			ObjectMeta: meta_v1.ObjectMeta{
				GenerateName: "e2e-tests-netchecker-",
				Namespace:    "",
			},
			Status: v1.NamespaceStatus{},
		}
		cr_body := newClusterRole(
			"netchecker-server",
			[]rbac.PolicyRule{
				{Verbs: []string{"*"}, APIGroups: []string{"apiextensions.k8s.io"}, Resources: []string{"customresourcedefinitions"}},
				{Verbs: []string{"*"}, APIGroups: []string{"network-checker.ext"}, Resources: []string{"agents"}},
				{Verbs: []string{"get", "list"}, APIGroups: []string{""}, Resources: []string{"pods"}},
			},
		)
		cr, err = clientset.Rbac().ClusterRoles().Create(cr_body)
		crb_body := newClusterRoleBinding(
			"netchecker", "rbac.authorization.k8s.io", "ClusterRole",
			"netchecker-server", "rbac.authorization.k8s.io", "Group", "system:serviceaccounts")
		crb, err = clientset.Rbac().ClusterRoleBindings().Create(crb_body)
		ns, err = clientset.Namespaces().Create(namespaceObj)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.AfterEach(func() {
		podList, _ := clientset.Core().Pods(ns.Name).List(meta_v1.ListOptions{LabelSelector: labels.Everything().String()})
		if ginkgo.CurrentGinkgoTestDescription().Failed {
			testutils.DumpLogs(clientset, podList.Items...)
		}
		for _, pod := range podList.Items {
			clientset.Core().Pods(pod.Namespace).Delete(pod.Name, &meta_v1.DeleteOptions{})
		}
		clientset.Namespaces().Delete(ns.Name, &meta_v1.DeleteOptions{})
		clientset.Rbac().ClusterRoleBindings().Delete(crb.Name, &meta_v1.DeleteOptions{})
		clientset.Rbac().ClusterRoles().Delete(cr.Name, &meta_v1.DeleteOptions{})
	})

	ginkgo.It("Connectivity check should pass", func() {
		ginkgo.By("deploying netchecker server pod")
		endpointArg := fmt.Sprintf("--endpoint=0.0.0.0:%d", serverPort)
		serverLabels := map[string]string{"app": "netchecker-server"}
		serverPod := newPod(
			"netchecker-server", "netchecker-server", "mirantis/k8s-netchecker-server",
			[]string{"netchecker-server", "--kubeproxyinit", "--logtostderr", "--v=5", endpointArg}, serverLabels, false, true, nil)
		pod, err := clientset.Pods(ns.Name).Create(serverPod)
		gomega.Expect(err).Should(gomega.BeNil())
		testutils.WaitForReady(clientset, pod)

		ginkgo.By("deploying netchecker service")
		servicePorts := []v1.ServicePort{{Protocol: v1.ProtocolTCP, Port: int32(serverPort), TargetPort: intstr.FromInt(serverPort)}}
		serverSvc := newService("netchecker-service", serverLabels, servicePorts, []string{})
		_, err = clientset.Services(ns.Name).Create(serverSvc)
		gomega.Expect(err).Should(gomega.BeNil())

		ginkgo.By("deploying netchecker agent daemonset")
		var ncAgentLabels = map[string]string{"app": "netchecker-agent"}
		serverEndpointArg := fmt.Sprintf("--serverendpoint=netchecker-service:%d", serverPort)
		cmd := []string{"netchecker-agent", "--alsologtostderr=true", "--v=5", serverEndpointArg, "--reportinterval=10"}
		agentDS := newDaemonSet("netchecker-agent", "netchecker-agent", "mirantis/k8s-netchecker-agent",
			[]string{"sh", "-c", strings.Join(cmd, " ")}, ncAgentLabels, false, true,
			[]v1.EnvVar{
				{Name: "MY_NODE_NAME", ValueFrom: &v1.EnvVarSource{FieldRef: &v1.ObjectFieldSelector{FieldPath: "spec.nodeName"}}},
				{Name: "MY_POD_NAME", ValueFrom: &v1.EnvVarSource{FieldRef: &v1.ObjectFieldSelector{FieldPath: "metadata.name"}}},
			},
		)
		_, err = clientset.Extensions().DaemonSets(ns.Name).Create(agentDS)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		// ensure agents are up and they have sent their reports to the server
		time.Sleep(15 * time.Second)

		services := getServices(clientset, ns)
		ncService := false
		for _, svc := range services {
			if svc.ObjectMeta.Name == "netchecker-service" {
				ncService = true
				break
			}
		}
		gomega.Expect(ncService).To(gomega.BeTrue())

		pods := getPods(clientset, ns)
		ncServerIP := ""
		ncAgentNames := map[string]bool{}
		for _, pod := range pods {
			if pod.ObjectMeta.Name == "netchecker-server" {
				ncServerIP = pod.Status.PodIP
			} else if pod.ObjectMeta.Name[:16] == "netchecker-agent" {
				ncAgentNames[pod.ObjectMeta.Name] = true
			}
		}
		gomega.Expect(ncServerIP).NotTo(gomega.BeEmpty())

		ginkgo.By("verifying that server is fed by all the agents")
		agentsResp := map[string]utils.AgentInfo{}
		httpServiceGet(serverPort, ncServerIP, "api/v1/agents/", &agentsResp)
		for agentName := range agentsResp {
			// server has reports from every agent
			gomega.Expect(ncAgentNames[agentName]).To(gomega.BeTrue())
		}
		// agent count in server's data is the same as agent pod count
		gomega.Expect(len(ncAgentNames)).To(gomega.BeEquivalentTo(len(agentsResp)))

		ginkgo.By("verifying connectivity in cluster")
		ccResp := utils.CheckConnectivityInfo{}
		httpServiceGet(serverPort, ncServerIP, "api/v1/connectivity_check", &ccResp)
		// server has reports from all the agents
		gomega.Expect(ccResp.Absent).To(gomega.BeEmpty())
		// all the agents reports are up to date
		gomega.Expect(ccResp.Outdated).To(gomega.BeEmpty())
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
				Env:             env,
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

func newClusterRole(roleName string, rules []rbac.PolicyRule) *rbac.ClusterRole {
	return &rbac.ClusterRole{
		ObjectMeta: meta_v1.ObjectMeta{Name: roleName},
		Rules: rules,
	}
}

func newClusterRoleBinding(bindName string, roleApigroup string, roleKind string, roleName string, subjApigroup string, subjKind string, subjName string) *rbac.ClusterRoleBinding {
	return &rbac.ClusterRoleBinding{
		ObjectMeta: meta_v1.ObjectMeta{Name: bindName},
		RoleRef: rbac.RoleRef{
			APIGroup:	roleApigroup,
			Kind:		roleKind,
			Name:		roleName,
		},
		Subjects: []rbac.Subject{
			{
				APIGroup:	subjApigroup,
				Kind:		subjKind,
				Name:		subjName,
			},
		},
	}
}

func httpServiceGet(port int, ip string, uri string, dst interface{}) {
	timeout := time.Duration(1 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}
	gomega.Eventually(func() error {
		resp, err := client.Get(fmt.Sprintf("http://%s:%d/%s", ip, port, uri))
		if err != nil {
			return err
		}
		if resp.StatusCode > 200 {
			return fmt.Errorf("Unexpected error from nginx service: %s", resp.Status)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		resp.Body.Close()
		err = json.Unmarshal(body, dst)
		return err
	}, 10*time.Second, 1*time.Second).Should(gomega.BeNil())
}

func getPods(clientset *kubernetes.Clientset, ns *v1.Namespace) []v1.Pod {
	pods, err := clientset.Pods(ns.Name).List(meta_v1.ListOptions{})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return pods.Items
}

func getServices(clientset *kubernetes.Clientset, ns *v1.Namespace) []v1.Service {
	services, err := clientset.Services(ns.Name).List(meta_v1.ListOptions{})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return services.Items
}
