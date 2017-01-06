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

func kubeClientSet() (kubernetes.Interface, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientSet, nil
}

func kubePods(kcs kubernetes.Interface) (*v1.PodList, error) {
	selector := labels.NewSelector()
	requirement, err := labels.NewRequirement(AgentLabelKey, selection.In, AgentLabelValues)
	if err != nil {
		return nil, err
	}
	selector.Add(*requirement)

	glog.V(10).Infof("Selector for kubernetes pods: %v", selector.String())

	pods, err := kcs.Core().Pods("").List(v1.ListOptions{LabelSelector: selector.String()})
	return pods, err
}

func checkKubeDataAgainstCache(pods *v1.PodList) ([]string, []string) {
	absent := []string{}
	outdated := []string{}

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

	return absent, outdated
}
