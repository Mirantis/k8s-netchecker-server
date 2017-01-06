package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"

	"k8s.io/client-go/kubernetes"
)

type agentInfo struct {
	ReportInterval int                 `json:"report_interval"`
	PodName        string              `json:"podname"`
	HostDate       time.Time           `json:"hostdate"`
	LastUpdated    time.Time           `json:"last_updated"`
	LookupHost     map[string][]string `json:"nslookup"`
	IPs            map[string][]string `json:"ips"`
}

type CheckConnectivityInfo struct {
	Message  string   `json="message"`
	Absent   []string `json="outdated,omitempty"`
	Outdated []string `json="absent,omitempty"`
}

var agentCache = make(map[string]agentInfo)

func updateAgents(rw http.ResponseWriter, r *http.Request, rp httprouter.Params) {
	body := make([]byte, r.ContentLength)
	n, err := r.Body.Read(body)
	if n <= 0 && err != nil {
		glog.Errorf("Error while reading request's body. Details: %v", err)
	}

	agentData := agentInfo{}
	err = json.Unmarshal(body, &agentData)
	if err != nil {
		glog.Errorf("Error while unmarshaling request's data. Details: %v", err)
	}
	agentData.LastUpdated = time.Now()
	glog.V(10).Infof("Updating the agents cache with value: %v", agentData)
	agentCache[rp.ByName("name")] = agentData
}

func getAgents(rw http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	body, err := json.Marshal(agentCache)
	if err != nil {
		glog.Errorf("Error while marshaling agents' data. Details: %v", err)
	}

	_, err = rw.Write(body)
	if err != nil {
		glog.Errorf("Error while writing response data for GET agents. Details: %v", err)
	}
}

func connectivityCheck(kcs kubernetes.Interface) httprouter.Handle {
	return func(rw http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		pods, err := kubePods(kcs)

		res := &CheckConnectivityInfo{}
		errMsg := "Connectivity check fails. Reason: %v"
		status := http.StatusOK

		if err != nil {
			message := fmt.Sprintf(
				"failed to retrieve pods from kubernetes; details: %v", err.Error())
			glog.Error(message)
			res = &CheckConnectivityInfo{Message: fmt.Sprintf(errMsg, message)}
			status = http.StatusBadRequest
		}

		absent, outdated := checkKubeDataAgainstCache(pods)
		if len(absent) != 0 || len(outdated) != 0 {
			glog.V(5).Infof(
				"Absent|outdated agents detected. Absent -> %v; outdated -> %v",
				absent, outdated,
			)
			res.Message = fmt.Sprintf(errMsg,
				"there are absent or outdated pods; look up the payload")
			res.Absent = absent
			res.Outdated = outdated

			status = http.StatusBadRequest
		}

		if status == http.StatusOK {
			message := fmt.Sprintf(
				"All %v pods successfully reported back to the server", len(agentCache))
			glog.V(10).Info(message)
			res.Message = message
		}

		body, err := json.Marshal(res)
		if err != nil {
			glog.Errorf("Marshalling of body for connectivity check response failed. Details: %v", err)
		}

		_, err = rw.Write(body)
		if err != nil {
			glog.Errorf("Writing response body failed. Details: %v", err)
		}

		rw.WriteHeader(status)
	}
}

func setupRouter(kcs kubernetes.Interface) *httprouter.Router {
	glog.V(10).Info("Setting up the url multiplexer")
	router := httprouter.New()
	router.POST("/api/v1/agents/:name", updateAgents)
	router.GET("/api/v1/agents/", getAgents)
	router.GET("/api/v1/connectivity_check", connectivityCheck(kcs))
	return router
}

func main() {
	var endpoint string
	flag.StringVar(&endpoint, "endpoint", "0.0.0.0:8081", "End point (IP address, port) for server to listen on")
	flag.Parse()

	glog.V(5).Infof("Start listening on %v", endpoint)

	clientSet, err := kubeClientSet()
	if err != nil {
		glog.Errorf("Error while creating k8s client set. Details: %v", err)
		panic(err.Error())
	}

	router := setupRouter(clientSet)
	http.ListenAndServe(endpoint, router)
}
