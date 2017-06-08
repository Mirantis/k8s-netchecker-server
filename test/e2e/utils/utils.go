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

package utils

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/httpstream/spdy"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubernetes/pkg/client/unversioned/remotecommand"
	remotecommandserver "k8s.io/kubernetes/pkg/kubelet/server/remotecommand"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
)

var apiMaster string

func init() {
	flag.StringVar(&apiMaster, "master", "http://localhost:8080", "apiserver address to use with restclient")
}

// Logf
func Logf(format string, a ...interface{}) {
	fmt.Fprintf(ginkgo.GinkgoWriter, format, a...)
}

func loadConfig() *rest.Config {
	config, err := clientcmd.BuildConfigFromFlags(apiMaster, "")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return config
}

// KubeClient
func KubeClient() (*kubernetes.Clientset, error) {
	Logf("Using master %v\n", apiMaster)
	config := loadConfig()
	clientset, err := kubernetes.NewForConfig(config)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return clientset, nil
}

// WaitForReady
func WaitForReady(clientset *kubernetes.Clientset, pod *v1.Pod) {
	gomega.Eventually(func() error {
		podUpdated, err := clientset.Core().Pods(pod.Namespace).Get(pod.Name, meta_v1.GetOptions{})
		if err != nil {
			return err
		}
		if podUpdated.Status.Phase != v1.PodRunning {
			return fmt.Errorf("pod %v is not running phase: %v", podUpdated.Name, podUpdated.Status.Phase)
		}
		return nil
	}, 120*time.Second, 5*time.Second).Should(gomega.BeNil())
}

// DumpLogs
func DumpLogs(clientset *kubernetes.Clientset, pods ...v1.Pod) {
	for _, pod := range pods {
		dumpLogs(clientset, pod)
	}
}

func dumpLogs(clientset *kubernetes.Clientset, pod v1.Pod) {
	req := clientset.Core().Pods(pod.Namespace).GetLogs(pod.Name, &v1.PodLogOptions{})
	readCloser, err := req.Stream()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	defer readCloser.Close()
	Logf("\n Dumping logs for %v:%v \n", pod.Namespace, pod.Name)
	_, err = io.Copy(ginkgo.GinkgoWriter, readCloser)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}

// ExecInPod
func ExecInPod(clientset *kubernetes.Clientset, pod v1.Pod, cmd ...string) (string, string, error) {
	Logf("Running %v in %v\n", cmd, pod.Name)

	container := pod.Spec.Containers[0].Name
	var stdout, stderr bytes.Buffer
	config := loadConfig()
	client := clientset.CoreV1Client.RESTClient()
	req := client.Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec").
		Param("container", container)
	req.VersionedParams(&api.PodExecOptions{
		Container: container,
		Command:   cmd,
		TTY:       false,
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
	}, api.ParameterCodec)
	err := execute("POST", req.URL(), config, nil, &stdout, &stderr, false)
	Logf("Error %v: %v\n", cmd, stderr.String())
	Logf("Output %v: %v\n", cmd, stdout.String())
	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
}

func execute(method string, url *url.URL, config *rest.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool) error {
	tlsConfig, err := rest.TLSConfigFor(config)
	if err != nil {
		return err
	}
	upgrader := spdy.NewRoundTripper(tlsConfig)
	exec, err := remotecommand.NewStreamExecutor(upgrader, nil, method, url)
	if err != nil {
		return err
	}
	return exec.Stream(remotecommand.StreamOptions{
		SupportedProtocols: remotecommandserver.SupportedStreamingProtocols,
		Stdin:              stdin,
		Stdout:             stdout,
		Stderr:             stderr,
		Tty:                tty,
	})
}
