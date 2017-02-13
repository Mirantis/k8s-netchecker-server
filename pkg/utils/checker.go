package utils

import "time"

type Checker interface {
	Check() (absent, outdated []string, err error)
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

func (ac *AgentChecker) Check() ([]string, []string, error) {
	absent := []string{}
	outdated := []string{}

	pods, err := ac.KubeProxy.Pods()
	if err != nil {
		return nil, nil, err
	}
	for _, pod := range pods.Items {
		agentName := pod.ObjectMeta.Name
		agentData, exists := AgentCache[agentName]
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
