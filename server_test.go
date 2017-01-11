package main

import (
	"bytes"
	"encoding/json"
	"errors"
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
		t.Errorf("Failed to marshal expectedAgent. Details: %v", err)
	}

	router := httprouter.New()
	router.POST("/api/v1/agents/:name", updateAgents)
	ts := httptest.NewServer(router)
	defer ts.Close()

	body := bytes.NewReader(marshalled)
	_, err = http.Post(
		ts.URL+"/api/v1/agents/"+expectedAgent.PodName,
		"application/json",
		body,
	)
	if err != nil {
		t.Errorf("Failed to post example agent to server. Details: %v", err)
	}

	aData, exists := agentCache["test"]
	if !exists {
		t.Errorf("agentCache does not contain key %v after updateAgents method call", "test")
	}

	//time.Now() is always different
	expectedAgent.LastUpdated = aData.LastUpdated

	expected, err := json.Marshal(expectedAgent)
	if err != nil {
		t.Errorf("Failed to marshal expected data with last_updated field. Details: %v", err)
	}

	actual, err := json.Marshal(aData)
	if err != nil {
		t.Errorf("Failed to marshal agent from the cache. Details: %v", err)
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
		t.Errorf("Failed to marshal agentCache (making expected byte array). Details: %v", err)
	}

	router := httprouter.New()
	router.GET("/api/v1/agents/", getAgents)
	ts := httptest.NewServer(router)
	defer ts.Close()

	res, err := http.Get(ts.URL + "/api/v1/agents/")
	if err != nil {
		t.Errorf("Failed to GET agents' data from server. Details: %v", err)
	}

	actual := make([]byte, res.ContentLength)
	n, err := res.Body.Read(actual)
	if n <= 0 && err != nil {
		t.Errorf("Failed to read response body of GET agents: Details: %v", err)
	}

	if !bytes.Equal(expected, actual) {
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

func createAgentChecker() *AgentChecker {
	return &AgentChecker{KubeProxy: &KubeProxy{Client: CSwithPods()}}
}

func createCnntyCheckTestServer(checker Checker) *httptest.Server {
	router := httprouter.New()
	router.GET("/api/v1/connectivity_check", connectivityCheck(checker))
	return httptest.NewServer(router)
}

func cnntyRespOrFail(serverURL string, expectedStatus int, t *testing.T) *http.Response {
	res, err := http.Get(serverURL + "/api/v1/connectivity_check")
	if err != nil {
		t.Errorf("Failed to GET successfull connectivity check from server. Details: %v", err)
	}
	if res.StatusCode != expectedStatus {
		t.Errorf(
			"Status code of connectivity check response must be %v instead it is %v",
			expectedStatus, res.StatusCode)
	}
	return res
}

func decodeCnntyRespOrFail(resp *http.Response, t *testing.T) *CheckConnectivityInfo {
	info := &CheckConnectivityInfo{}
	decoder := json.NewDecoder(resp.Body)
	err := decoder.Decode(info)
	if err != nil {
		t.Errorf(
			"Failed to decode connectivity check successfull response body. Details: %v",
			err)
	}
	return info
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

	aChecker := createAgentChecker()
	ts := createCnntyCheckTestServer(aChecker)
	defer ts.Close()

	actual := decodeCnntyRespOrFail(cnntyRespOrFail(ts.URL, http.StatusOK, t), t)
	successfullMsg := fmt.Sprintf(
		"All %v pods successfully reported back to the server", len(agentCache))
	if actual.Message != successfullMsg {
		t.Errorf(
			"Unexpected message from successfull result payload. Actual: %v",
			actual.Message)
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

	aChecker := createAgentChecker()
	ts := createCnntyCheckTestServer(aChecker)
	defer ts.Close()

	actual := decodeCnntyRespOrFail(cnntyRespOrFail(ts.URL, http.StatusBadRequest, t), t)
	failMsg := fmt.Sprintf(
		"Connectivity check fails. Reason: %v",
		"there are absent or outdated pods; look up the payload")

	if actual.Message != failMsg {
		t.Errorf(
			"Unexpected message from bad request result payload. Actual: %v",
			actual.Message)
	}
	if actual.Outdated[0] != "agent-pod-hostnet" {
		t.Errorf("agent-pod-hostnet must be returned in the payload in the 'outdated' array")
	}
	if actual.Absent[0] != "agent-pod" {
		t.Errorf("agent-pod must be returned in the payload in the 'absent' array")
	}
}

type FakeChecker struct {
	ErrorMessage string
}

func (fc *FakeChecker) Check() ([]string, []string, error) {
	return nil, nil, errors.New("test error")
}

func TestConnectivityCheckFailDueError(t *testing.T) {
	tChecker := &FakeChecker{ErrorMessage: "test error"}
	ts := createCnntyCheckTestServer(tChecker)
	defer ts.Close()

	actual := decodeCnntyRespOrFail(cnntyRespOrFail(ts.URL, http.StatusBadRequest, t), t)
	failMsg := fmt.Sprintf(
		"Connectivity check fails. Reason: %v",
		fmt.Sprintf("failed to check agents; details: %v", tChecker.ErrorMessage))
	if actual.Message != failMsg {
		t.Errorf(
			"Unexpected message from bad request result payload. Actual: %v",
			actual.Message)
	}
}
