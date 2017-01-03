package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"http"
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/labels"
	"k8s.io/client-go/pkg/selection"
	"k8s.io/client-go/rest"
)

const AgentLabelKey = "app"
const AgentLabelValues = []string{"netchecker-agent", "netchecker-agent-hostnet"}

type agentInfo struct {
	ReportInterval int                 `json:"report_interval"`
	PodName        string              `json:"podname"`
	HostDate       time.Time           `json:"hostdate"`
	LastUpdated    time.Time           `json:"last_updated"`
	LookupHost     map[string][]string `json:"nslookup"`
	IPs            map[string][]string `json:"ips"`
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

func checkAgentsData(pods v1.PodList) ([]string, []string) {
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
		if delta > agentData.ReportInterval.(float64) {
			outdated = append(outdated, agentName)
		}
	}

	return absent, outdated
}

func connectivityCheck(kcs kubernetes.Interface) httprouter.Handle {
	return func(rw http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		selector := labels.NewSelector()
		requirement, err := labels.NewRequirement(AgentLabelKey, selection.In, AgentLabelValues)
		if err != nil {
			glog.Errorf("Error while creating requirement for the agent's pods' labels. Details: %v", err)
		}
		selector.Add(requirement)

		pods, err := kcs.Core().Pods("").List(v1.ListOptions{LabelSelector: selector.String()})
		if err != nil {
			glog.Errorf("Fail to get list of the agent's pods. Details: %v", err)
		}

		absent, outdated := checkAgentsData(pods)

		type result struct {
			Message  string   `json="message"`
			Absent   []string `json="outdated,omitempty"`
			Outdated []string `json="absent,omitempty"`
		}

		res := &result{}
		if len(absent) != 0 || len(outdated) != 0 {
			res.Message = "Connectivity check fails. Inspect the payload for details"
			res.Absent = absent
			res.Outdated = outdated

			rw.WriteHeader(http.StatusBadRequest)
		} else {
			res.Message = fmt.Sprintf(
				"All %v pods successfully reported back to the server", len(agentCache))
		}

		body, err := json.Marshal(res)
		if err != nil {
			glog.Errorf("Marshalling of body for connectivity check response failed. Details: %v", err)
		}

		_, err = rw.Write(body)
		if err != nil {
			glog.Errorf("Writing response body failed. Details: %v", err)
		}
	}
}

func kubeClientSet() (kubernetes.Clientset, error) {
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

func setupRouter() (*httprouter.Router, error) {
	glog.V(10).Info("Setting up the url multiplexer")
	router := httprouter.New()
	router.POST("/api/v1/agents/:name", updateAgents)
	router.GET("/api/v1/agents/", getAgents)

	clientSet, err := kubeClientSet()
	if err != nil {
		return nil, err
	}
	router.GET("/api/v1/connectivity_check", connectivityCheck(clientSet))

	return router, nil
}

func main() {
	var endpoint string
	flag.StringVar(&endpoint, "endpoint", "0.0.0.0:8081", "End point (IP address, port) for server to listen on")
	flag.Parse()

	glog.V(5).Infof("Start listening on %v", endpoint)
	router, err := setupRouter()
	if err != nil {
		glog.Errorf("Error while setting up the http router")
		panic(err.Error())
	}
	http.ListenAndServe(endpoint, setupRouter())
}
