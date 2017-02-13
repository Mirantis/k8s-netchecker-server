package utils

import (
	"fmt"
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/urfave/negroni"
)

func SetupRouter(chkr Checker) *httprouter.Router {
	glog.V(10).Info("Setting up the url multiplexer")
	router := httprouter.New()
	router.POST("/api/v1/agents/:name", UpdateAgents)
	router.GET("/api/v1/agents/:name", GetSingleAgent)
	router.GET("/api/v1/agents/", GetAgents)
	router.GET("/api/v1/connectivity_check", ConnectivityCheck(chkr))
	return router
}

func AddMiddleware(handler http.Handler) http.Handler {
	n := negroni.New()
	n.Use(negroni.NewLogger())
	n.Use(negroni.NewRecovery())
	n.UseHandler(handler)
	return n
}

func UpdateAgents(rw http.ResponseWriter, r *http.Request, rp httprouter.Params) {
	agentData := AgentInfo{}
	if err := ProcessRequest(r, &agentData, rw); err != nil {
		return
	}

	agentData.LastUpdated = time.Now()
	glog.V(10).Infof("Updating the agents cache with value: %v", agentData)
	AgentCache[rp.ByName("name")] = agentData
}

func GetAgents(rw http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ProcessResponse(rw, AgentCache)
}

func GetSingleAgent(rw http.ResponseWriter, r *http.Request, rp httprouter.Params) {
	aName := rp.ByName("name")
	aData, exists := AgentCache[aName]
	if !exists {
		glog.V(5).Infof("Agent with name %v is not found in the cache", aName)
		http.Error(rw, "There is no such entry in the agent cache", http.StatusNotFound)
		return
	}
	ProcessResponse(rw, aData)
}

func ConnectivityCheck(checker Checker) httprouter.Handle {
	return func(rw http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		res := &CheckConnectivityInfo{
			Message: fmt.Sprintf(
				"All %v pods successfully reported back to the server",
				len(AgentCache)),
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

		ProcessResponse(rw, res)
	}
}
