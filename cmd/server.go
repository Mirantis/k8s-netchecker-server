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
	"time"

	"github.com/Mirantis/k8s-netchecker-server/pkg/utils"
	"github.com/golang/glog"
)

var version string

func main() {
	var (
		repTTL      int
		pingTimeout int
	)

	config := utils.GetOrCreateConfig()

	flag.StringVar(&config.HttpListen, "endpoint", "0.0.0.0:8081", "Endpoint (IP address, port) for server to listen on")
	flag.BoolVar(&config.UseKubeClient, "kubeproxyinit", false, "use k8s TPR (true) or Etcd (false) as a data storage")
	flag.IntVar(&repTTL, "report-ttl", 300, "TTL for agents reports data stored in Etcd (sec)")
	flag.IntVar(&pingTimeout, "ping-timeout", 5, "Etcd server ping timeout (sec)")
	flag.StringVar(&config.EtcdEndpoints, "etcd-endpoints", "", "Etcd server endpoints list")
	flag.StringVar(&config.EtcdTree, "etcd-tree", "netchecker", "Root of Etcd tree")
	flag.StringVar(&config.EtcdKeyFile, "etcd-key", "", "SSL key file when using HTTPS to connect to etcd")
	flag.StringVar(&config.EtcdCertFile, "etcd-cert", "", "SSL certificate file when using HTTPS to connect to etcd")
	flag.StringVar(&config.EtcdCAFile, "etcd-ca", "", "SSL CA file when using HTTPS to connect to etcd")
	flag.Parse()
	glog.Infof("K8s netchecker. Compiled at: %s", version)

	config.ReportTTL = time.Duration(repTTL) * time.Second
	config.PingTimeout = time.Duration(pingTimeout) * time.Second

	glog.V(5).Infof("Start listening on %v", config.HttpListen)

	handler, err := utils.NewHandler()
	if err != nil {
		glog.Errorf("Error while setting up the handler. Details: %v", err)
		panic(err.Error())
	}

	go handler.CollectAgentsMetrics()
	glog.Fatal(http.ListenAndServe(config.HttpListen, handler.HTTPHandler))
}
