package main

import (
	"flag"
	"net/http"

	"github.com/Mirantis/k8s-netchecker-server/pkg/utils"
	"github.com/golang/glog"
)

func main() {
	var endpoint string
	var initKubeProxy bool
	flag.StringVar(&endpoint, "endpoint", "0.0.0.0:8081", "End point (IP address, port) for server to listen on")
	flag.BoolVar(&initKubeProxy, "kubeproxyinit", false, "Control initialization kubernetes client set for connectivity check")
	flag.Parse()

	glog.V(5).Infof("Start listening on %v", endpoint)

	handler, err := utils.NewHandler(initKubeProxy)
	if err != nil {
		glog.Errorf("Error while setting up the handler. Details: %v", err)
		panic(err.Error())
	}

	http.ListenAndServe(endpoint, handler.HTTPHandler)
}
