// Copyright 2017 Mirantis
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/pkg/api/v1"
)

func newHandler() *Handler {
	return &Handler{
		AgentCache: map[string]AgentInfo{},
		Metrics:    map[string]AgentMetrics{},
	}
}

func agentExample() AgentInfo {
	return AgentInfo{
		ReportInterval: 5,
		NodeName:       "test-node",
		PodName:        "test",
		HostDate:       time.Now(),
		NetworkProbes:  []ProbeResult{{"http://0.0.0.0:8081", 1, 50, 1, 0, 0, 0, 0}},
	}
}

func checkRespStatus(expected, actual int, t *testing.T) {
	if actual != expected {
		t.Errorf("Response status code %v is not as expected %v", actual, expected)
	}
}

func checkCacheKey(h *Handler, key string, expected bool, t *testing.T) {
	_, exists := h.AgentCache[key]
	if exists != expected {
		t.Errorf("Presence of the key %v in AgentCache must be %v", key, expected)
	}
}

func readBodyBytesOrFail(resp *http.Response, t *testing.T) []byte {
	bData := make([]byte, resp.ContentLength)
	n, err := resp.Body.Read(bData)
	if n <= 0 && err != nil {
		t.Errorf("Error while reading response from UpdateAgents. Details: %v", err)
	}

	return bData
}

func marshalExpectedWithActualDate(expected, actual AgentInfo, t *testing.T) []byte {
	//time.Now() is always different
	expected.LastUpdated = actual.LastUpdated

	bExpected, err := json.Marshal(expected)
	if err != nil {
		t.Errorf("Failed to marshal expected data with last_updated field. Details: %v", err)
	}

	return bExpected
}

func TestUpdateAgents(t *testing.T) {
	expectedAgent := agentExample()
	marshalled, err := json.Marshal(expectedAgent)
	if err != nil {
		t.Errorf("Failed to marshal expectedAgent. Details: %v", err)
	}

	handler := newHandler()
	router := httprouter.New()
	router.POST("/api/v1/agents/:name", handler.UpdateAgents)
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

	checkCacheKey(handler, "test", true, t)

	aData := handler.AgentCache["test"]

	expected := marshalExpectedWithActualDate(expectedAgent, aData, t)

	actual, err := json.Marshal(aData)
	if err != nil {
		t.Errorf("Failed to marshal agent from the cache. Details: %v", err)
	}

	if !bytes.Equal(expected, actual) {
		t.Errorf(
			"Actual data from AgentCache %v is not as expected %v",
			handler.AgentCache["test"],
			expectedAgent)
	}
}

func TestUpdateAgentsFailedUnmarshal(t *testing.T) {
	handler := newHandler()
	router := httprouter.New()
	router.POST("/api/v1/agents/:name", handler.UpdateAgents)
	ts := httptest.NewServer(router)
	defer ts.Close()

	resp, err := http.Post(
		ts.URL+"/api/v1/agents/test", "text", strings.NewReader("some text"))
	if err != nil {
		t.Errorf("Failed to perform POST request on UpdateAgents. Details: %v", err)
	}

	checkRespStatus(http.StatusInternalServerError, resp.StatusCode, t)

	bData := readBodyBytesOrFail(resp, t)
	s := string(bData)
	expected := "Error while unmarshaling data."
	if !strings.Contains(s, expected) {
		t.Errorf("Response data should contains following message '%v'. Instead it is '%v'",
			expected, s)
	}

	checkCacheKey(handler, "test", false, t)
}

type Body struct {
	Message string
}

func (b *Body) Read(p []byte) (n int, err error) {
	return 0, errors.New(b.Message)
}

func TestUpdateAgentsFailReadBody(t *testing.T) {
	body := &Body{Message: "test error message"}
	r := httptest.NewRequest(
		"POST", "/api/v1/agents/test", body)
	r.ContentLength = 0
	rw := httptest.NewRecorder()

	handler := newHandler()
	handler.UpdateAgents(rw, r, httprouter.Params{httprouter.Param{Key: "name", Value: "test"}})

	checkRespStatus(http.StatusInternalServerError, rw.Code, t)

	s := string(rw.Body.Bytes())
	expected := "Error while reading bytes from the request's body."
	if !strings.Contains(s, expected) {
		t.Errorf("Response data should contains following message '%v'. Instead it is '%v'",
			expected, s)
	}
	checkCacheKey(handler, "test", false, t)
}

func TestGetAgents(t *testing.T) {
	handler := newHandler()
	handler.AgentCache["test"] = agentExample()
	expected, err := json.Marshal(handler.AgentCache)
	if err != nil {
		t.Errorf("Failed to marshal AgentCache (making expected byte array). Details: %v", err)
	}

	router := httprouter.New()
	router.GET("/api/v1/agents/", handler.CleanCache(handler.GetAgents))
	ts := httptest.NewServer(router)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/agents/")
	if err != nil {
		t.Errorf("Failed to GET agents' data from server. Details: %v", err)
	}

	actual := readBodyBytesOrFail(resp, t)
	if !bytes.Equal(expected, actual) {
		t.Error("Response body for GET agents is not as expected")
	}
}

func TestGetSingleAgent(t *testing.T) {
	handler := newHandler()
	handler.AgentCache["test"] = agentExample()

	router := httprouter.New()
	router.GET("/api/v1/agents/:name", handler.GetSingleAgent)
	ts := httptest.NewServer(router)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/agents/test")
	if err != nil {
		t.Errorf("Failed to GET agents' data from server. Details: %v", err)
	}

	actual := readBodyBytesOrFail(resp, t)

	bExpected, err := json.Marshal(handler.AgentCache["test"])
	if err != nil {
		t.Errorf("Failed to marshal expected data with last_updated field. Details: %v", err)
	}

	if !bytes.Equal(bExpected, actual) {
		t.Error("Response body for GET agents is not as expected")
	}
}

func TestGetSingleAgentCleanCache(t *testing.T) {
	handler := newHandler()
	handler.AgentCache["test"] = agentExample()
	handler.AgentCache["agent-pod"] = agentExample()

	handler.KubeClient = &KubeProxy{Client: CSwithPods()}

	router := httprouter.New()
	router.GET("/api/v1/agents/:name", handler.CleanCache(handler.GetSingleAgent))
	ts := httptest.NewServer(router)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/agents/test")
	if err != nil {
		t.Errorf("Failed to GET agents' data from server. Details: %v", err)
	}

	readBodyBytesOrFail(resp, t)

	if _, exists := handler.AgentCache["test"]; exists {
		t.Errorf("Key %v should not be present in the cache", "test")
	}
}

func CSwithPods() kubernetes.Interface {
	return fake.NewSimpleClientset(
		&v1.Pod{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "agent-pod",
				Labels:    map[string]string{"app": AgentLabelValues[0]},
				Namespace: v1.NamespaceDefault,
			},
		},
		&v1.Pod{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "agent-pod-hostnet",
				Labels:    map[string]string{"app": AgentLabelValues[0]},
				Namespace: v1.NamespaceDefault,
			},
		},
		&v1.Pod{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "agent-pod-test",
				Labels:    map[string]string{"app": "test"},
				Namespace: v1.NamespaceDefault,
			},
		},
	)
}

func createCnntyCheckTestServer(handler *Handler) *httptest.Server {
	router := httprouter.New()
	router.GET("/api/v1/connectivity_check", handler.CleanCache(handler.ConnectivityCheck))
	router.Handler("GET", "/metrics", promhttp.Handler())
	return httptest.NewServer(router)
}

func cnntyRespOrFail(serverURL string, expectedStatus int, t *testing.T) *http.Response {
	res, err := http.Get(serverURL + "/api/v1/connectivity_check")
	if err != nil {
		t.Errorf("Failed to GET successful connectivity check from server. Details: %v", err)
	}
	checkRespStatus(expectedStatus, res.StatusCode, t)
	return res
}

func metricsRespOrFail(serverURL string, expectedStatus int, t *testing.T) *http.Response {
	res, err := http.Get(serverURL + "/metrics")
	if err != nil {
		t.Errorf("Failed to GET metrics from server. Details: %v", err)
	}
	checkRespStatus(expectedStatus, res.StatusCode, t)
	return res
}

func decodeCnntyRespOrFail(resp *http.Response, t *testing.T) *CheckConnectivityInfo {
	info := &CheckConnectivityInfo{}
	decoder := json.NewDecoder(resp.Body)
	err := decoder.Decode(info)
	if err != nil {
		t.Errorf(
			"Failed to decode connectivity check successful response body. Details: %v",
			err)
	}
	return info
}

func TestConnectivityCheckSuccess(t *testing.T) {
	handler := newHandler()
	handler.KubeClient = &KubeProxy{Client: CSwithPods()}

	agent := agentExample()
	agent.LastUpdated = agent.HostDate

	agent.PodName = "agent-pod"
	handler.AgentCache[agent.PodName] = agent

	agent.PodName = "agent-pod-hostnet"
	handler.AgentCache[agent.PodName] = agent

	ts := createCnntyCheckTestServer(handler)
	defer ts.Close()

	actual := decodeCnntyRespOrFail(cnntyRespOrFail(ts.URL, http.StatusOK, t), t)
	successfulMsg := fmt.Sprintf(
		"All %v pods successfully reported back to the server", len(handler.AgentCache))
	if actual.Message != successfulMsg {
		t.Errorf(
			"Unexpected message from successful result payload. Actual: %v",
			actual.Message)
	}
}

func TestMetricsGetSuccess(t *testing.T) {
	handler := newHandler()
	handler.KubeClient = &KubeProxy{Client: CSwithPods()}

	agent := agentExample()
	agent.PodName = "agent-pod"
	handler.AgentCache[agent.PodName] = agent

	ts := createCnntyCheckTestServer(handler)
	defer ts.Close()

	metricsRespOrFail(ts.URL, http.StatusOK, t)
}

func TestConnectivityCheckFail(t *testing.T) {
	handler := newHandler()
	handler.KubeClient = &KubeProxy{Client: CSwithPods()}

	agent := agentExample()

	agent.PodName = "agent-pod-hostnet"
	//back to the past
	agent.LastUpdated = agent.HostDate.Add(
		-time.Second * time.Duration(agent.ReportInterval*2+1))

	handler.AgentCache[agent.PodName] = agent

	ts := createCnntyCheckTestServer(handler)
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

type FakeProxy struct {
}

func (fp *FakeProxy) Pods() (*v1.PodList, error) {
	return nil, errors.New("test error")
}

func TestConnectivityCheckFailDueError(t *testing.T) {
	handler := newHandler()
	handler.KubeClient = &FakeProxy{}
	ts := createCnntyCheckTestServer(handler)
	defer ts.Close()

	resp := cnntyRespOrFail(ts.URL, http.StatusInternalServerError, t)
	bData := readBodyBytesOrFail(resp, t)
	actual := string(bData)

	failMsg := fmt.Sprintf(
		"Failed to get pods from k8s cluster. Details: test error\n")

	if !strings.Contains(actual, failMsg) {
		t.Errorf(
			"Unexpected message from bad request result payload. Actual: %v",
			actual)
	}
}
