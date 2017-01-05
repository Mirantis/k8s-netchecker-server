package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/julienschmidt/httprouter"
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
	cleanAgentCache()
}

func TestGetAgents(t *testing.T) {
	cleanAgentCache()
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

	cleanAgentCache()
}
