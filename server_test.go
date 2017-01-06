package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/julienschmidt/httprouter"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/pkg/api/v1"
)

func cleanAgentCache() { agentCache = make(map[string]agentInfo) }

func agentExample() agentInfo {
	return agentInfo{
		ReportInterval: 5,
		PodName:        "test",
		HostDate:       time.Now(),
	}
}

func TestUpdateAgents(t *testing.T) {
	cleanAgentCache()
	defer cleanAgentCache()
	expectedAgent := agentExample()

	marshalled, err := json.Marshal(expectedAgent)
	if err != nil {
		t.Errorf("Fail to marshal expectedAgent. Details: %v", err)
	}

	rw := httptest.NewRecorder()
	r := httptest.NewRequest(
		"POST",
		"http://example.com/api/v1/agents/test",
		bytes.NewReader(marshalled))
	rp := httprouter.Params{httprouter.Param{Key: "name", Value: "test"}}

	updateAgents(rw, r, rp)

	aData, exists := agentCache["test"]
	if !exists {
		t.Errorf("agentCache does not contain key %v after updateAgents method call", "test")
	}

	//time.Now() is always different
	expectedAgent.LastUpdated = aData.LastUpdated

	expected, err := json.Marshal(expectedAgent)
	if err != nil {
		t.Errorf("Fail to marshal expected data with last_updated field. Details: %v", err)
	}

	actual, err := json.Marshal(aData)
	if err != nil {
		t.Errorf("Fail to marshal agent from the cache. Details: %v", err)
	}

	if !bytes.Equal(expected, actual) {
		t.Errorf(
			"Actual data from agentCache %v is not as expected %v",
			agentCache["test"],
			expectedAgent)
	}
}

func TestGetAgents(t *testing.T) {
	cleanAgentCache()
	defer cleanAgentCache()
	agentCache["test"] = agentExample()

	expected, err := json.Marshal(agentCache)
	if err != nil {
		t.Errorf("Fail to marshal agentCache (making expected byte array). Details: %v", err)
	}

	rw := httptest.NewRecorder()
	r := httptest.NewRequest(
		"GET",
		"http://example.com/api/v1/agents/test",
		nil)

	getAgents(rw, r, httprouter.Params{})

	if !bytes.Equal(expected, rw.Body.Bytes()) {
		t.Error("Response body for GET agents is not as expected")
	}
}

func CSwithPods() kubernetes.Interface {
	return fake.NewSimpleClientset(
		&v1.Pod{
			ObjectMeta: v1.ObjectMeta{
				Name:      "agent-pod",
				Labels:    map[string]string{"app": AgentLabelValues[0]},
				Namespace: v1.NamespaceDefault,
			},
		},
		&v1.Pod{
			ObjectMeta: v1.ObjectMeta{
				Name:      "agent-pod-hostnet",
				Labels:    map[string]string{"app": AgentLabelValues[0]},
				Namespace: v1.NamespaceDefault,
			},
		})
}

func TestConnectivityCheckSuccess(t *testing.T) {
	cleanAgentCache()
	defer cleanAgentCache()

	agent := agentExample()
	agent.LastUpdated = agent.HostDate

	agent.PodName = "agent-pod"
	agentCache[agent.PodName] = agent

	agent.PodName = "agent-pod-hostnet"
	agentCache[agent.PodName] = agent

	clientSet := CSwithPods()

	rw := httptest.NewRecorder()
	r := httptest.NewRequest(
		"GET",
		"http://example.com/api/v1/connectivity_check",
		nil)

	connectivityCheck(clientSet)(rw, r, httprouter.Params{})

	if rw.Code != http.StatusOK {
		t.Errorf(
			"Status code of connectivity check response must be OK instead it is %v",
			rw.Code)
	}

	result := &CheckConnectivityInfo{}
	err := json.Unmarshal(rw.Body.Bytes(), result)
	if err != nil {
		t.Errorf(
			"Fail to unmarshal connectivity check successfull response body. Details: %v",
			err)
	}

	successfullMsg := fmt.Sprintf(
		"All %v pods successfully reported back to the server", len(agentCache))
	if result.Message != successfullMsg {
		t.Errorf(
			"Unexpected message from successfull result payload. Actual: %v",
			result.Message)
	}
}

func TestConnectivityCheckFail(t *testing.T) {
	cleanAgentCache()
	defer cleanAgentCache()

	agent := agentExample()

	agent.PodName = "agent-pod-hostnet"
	//back to the past
	agent.LastUpdated = agent.HostDate.Add(
		-time.Second * time.Duration(agent.ReportInterval+1))
	agentCache[agent.PodName] = agent

	clientSet := CSwithPods()

	rw := httptest.NewRecorder()
	r := httptest.NewRequest(
		"GET",
		"http://example.com/api/v1/connectivity_check",
		nil)

	handleFunc := connectivityCheck(clientSet)
	handleFunc(rw, r, httprouter.Params{})

	result := &CheckConnectivityInfo{}
	err := json.Unmarshal(rw.Body.Bytes(), result)
	if err != nil {
		t.Errorf(
			"Fail to unmarshal connectivity check failed response body. Details: %v",
			err)
	}

	failMsg := fmt.Sprintf(
		"Connectivity check fails. Reason: %v",
		"there are absent or outdated pods; look up the payload")

	if result.Message != failMsg {
		t.Errorf(
			"Unexpected message from bad request result payload. Actual: %v",
			result.Message)
	}

	if result.Outdated[0] != "agent-pod-hostnet" {
		t.Errorf("agent-pod-hostnet must be returned in the payload in the 'outdated' array")
	}
	if result.Absent[0] != "agent-pod" {
		t.Errorf("agent-pod must be returned in the payload in the 'absent' array")
	}
}
