package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/urfave/negroni"
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
	var message string

	body := make([]byte, r.ContentLength)
	n, err := r.Body.Read(body)
	if n <= 0 && err != nil {
		message = fmt.Sprintf("Error while reading request's body. Details: %v", err)
		glog.Errorf(message)
		http.Error(rw, message, http.StatusInternalServerError)
		return
	}

	agentData := agentInfo{}
	err = json.Unmarshal(body, &agentData)
	if err != nil {
		message = fmt.Sprintf("Error while unmarshaling request's data. Details: %v", err)
		glog.Errorf(message)
		http.Error(rw, message, http.StatusInternalServerError)
		return
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

func connectivityCheck(checker Checker) httprouter.Handle {
	return func(rw http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		res := &CheckConnectivityInfo{
			Message: fmt.Sprintf(
				"All %v pods successfully reported back to the server",
				len(agentCache)),
		}
		status := http.StatusOK
		errMsg := "Connectivity check fails. Reason: %v"

		absent, outdated, err := checker.Check()
		if err != nil {
			message := fmt.Sprintf(
				"Error occured while checking the agents. Details: %v", err)
			glog.Error(message)
			http.Error(rw, message, http.StatusInternalServerError)
			return
		}

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

		glog.V(10).Infof("Connectivity check result: %v", res)
		glog.V(10).Infof("Connectivity check HTTP response status code: %v", status)

		rw.WriteHeader(status)

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

func setupRouter(chkr Checker) *httprouter.Router {
	glog.V(10).Info("Setting up the url multiplexer")
	router := httprouter.New()
	router.POST("/api/v1/agents/:name", updateAgents)
	router.GET("/api/v1/agents/", getAgents)
	router.GET("/api/v1/connectivity_check", connectivityCheck(chkr))
	return router
}

func main() {
	var endpoint string
	var initKubeProxy bool
	flag.StringVar(&endpoint, "endpoint", "0.0.0.0:8081", "End point (IP address, port) for server to listen on")
	flag.BoolVar(&initKubeProxy, "kubeproxyinit", false, "Control initialization kubernetes client set for connectivity check")
	flag.Parse()

	glog.V(5).Infof("Start listening on %v", endpoint)

	checker, err := NewAgentChecker(initKubeProxy)
	if err != nil {
		glog.Errorf("Error while creating agent checker. Details: %v", err)
		panic(err.Error())
	}

	router := setupRouter(checker)

	n := negroni.New()
	n.Use(negroni.NewLogger())
	n.UseHandler(router)

	http.ListenAndServe(endpoint, n)
}
