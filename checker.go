package main

import (
	"time"

	"github.com/golang/glog"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/labels"
	"k8s.io/client-go/pkg/selection"
	"k8s.io/client-go/rest"
)

const AgentLabelKey = "app"

var AgentLabelValues = []string{"netchecker-agent", "netchecker-agent-hostnet"}

type Checker interface {
	Check() (absent, outdated []string, err error)
}

type KubeProxy struct {
	Client kubernetes.Interface
}

type AgentChecker struct {
	KubeProxy *KubeProxy
}

func NewAgentChecker(proxyInit bool) (*AgentChecker, error) {
	if !proxyInit {
		return &AgentChecker{}, nil
	}

	kProxy := &KubeProxy{}
	err := kProxy.SetupClientSet()
	if err != nil {
		return nil, err
	}
	return &AgentChecker{KubeProxy: kProxy}, nil
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
	selector := labels.NewSelector()
	requirement, err := labels.NewRequirement(AgentLabelKey, selection.In, AgentLabelValues)
	if err != nil {
		return nil, err
	}
	selector.Add(*requirement)

	glog.V(10).Infof("Selector for kubernetes pods: %v", selector.String())

	pods, err := kp.Client.Core().Pods("").List(v1.ListOptions{LabelSelector: selector.String()})
	return pods, err
}

func (ac *AgentChecker) Check() ([]string, []string, error) {
	absent := []string{}
	outdated := []string{}

	pods, err := ac.KubeProxy.Pods()
	if err != nil {
		return nil, nil, err
	}
	for _, pod := range pods.Items {
		agentName := pod.ObjectMeta.Name
		agentData, exists := agentCache[agentName]
		if !exists {
			absent = append(absent, agentName)
			continue
		}

		delta := time.Now().Sub(agentData.LastUpdated).Seconds()
		if delta > float64(agentData.ReportInterval) {
			outdated = append(outdated, agentName)
		}
	}

	return absent, outdated, nil
}
