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

	go handler.CollectAgentsMetrics()
	glog.Fatal(http.ListenAndServe(endpoint, handler.HTTPHandler))
}
