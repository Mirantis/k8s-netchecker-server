package utils

import (
	"github.com/golang/glog"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/labels"
	"k8s.io/client-go/pkg/selection"
	"k8s.io/client-go/rest"
)

const AgentLabelKey = "app"

var AgentLabelValues = []string{"netchecker-agent", "netchecker-agent-hostnet"}

type KubeProxy struct {
	Client kubernetes.Interface
}

func (kp *KubeProxy) SetupClientSet() error {
	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}
	clientSet, err := kubernetes.NewForConfig(config)

	if err != nil {
		return err
	}

	kp.Client = clientSet
	return nil
}

func (kp *KubeProxy) Pods() (*v1.PodList, error) {
	requirement, err := labels.NewRequirement(AgentLabelKey, selection.In, AgentLabelValues)
	if err != nil {
		return nil, err
	}
	glog.V(10).Infof("Selector for kubernetes pods: %v", requirement.String())

	pods, err := kp.Client.Core().Pods("").List(v1.ListOptions{LabelSelector: requirement.String()})
	return pods, err
}
