package main

import (
	"encoding/json"
	"flag"
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
)

type agentInfo struct {
	ReportInterval int                 `json:"report_interval"`
	PodName        string              `json:"podname"`
	HostDate       time.Time           `json:"hostdate"`
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
	agentCache[rp.ByName("name")] = agentData
}

func getAgents(rw http.ResponseWriter, r *http.Request, rp httprouter.Params) {
	body, err := json.Marshal(agentCache)
	if err != nil {
		glog.Errorf("Error while marshaling agents' data. Details: %v", err)
	}

	_, err = rw.Write(body)
	if err != nil {
		glog.Errorf("Error while writing response data for GET agents. Details: %v", err)
	}
}

func setupRouter() *httprouter.Router {
	router := httprouter.New()
	router.POST("/api/v1/agents/:name", updateAgents)
	router.GET("/api/v1/agents/", getAgents)

	return router
}

func main() {
	var endpoint string
	flag.StringVar(&endpoint, "endpoint", "0.0.0.0:8081", "End point (IP address, port) for server to listen on")
	flag.Parse()

	glog.V(5).Infof("Start listening on %v", endpoint)
	http.ListenAndServe(endpoint, setupRouter())
}
