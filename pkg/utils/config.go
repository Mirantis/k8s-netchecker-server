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
	"encoding/json"
	"github.com/golang/glog"
	"gopkg.in/yaml.v2"
	"sync"
	"time"
)

type AppConfig struct {
	sync.Mutex                  // ensures atomic writes; protects the following fields
	UseKubeClient bool          // use k8s TPR (true) or etcd (false) as a data storage
	EtcdEndpoints string        // endpoints (IPaddress1:PORT1[,IPaddress2:PORT2]) of etcd server
	                            // when etcd is being used as a data storage
	EtcdTree      string        // Root of NetChecker server etcd tree
	EtcdCertFile  string        // SSL certificate file when using HTTPS to connect to etcd
	EtcdKeyFile   string        // SSL key file when using HTTPS to connect to etcd
	EtcdCAFile    string        // SSL CA file when using HTTPS to connect to etcd
	HttpListen    string        // REST API endpoint (IPaddress:PORT) for netchecker server to listen to
	PingTimeout   time.Duration // etcd ping timeout (sec)
	ReportTTL     time.Duration // TTL for Agent report data when etcd is in use (sec)
	CheckInterval time.Duration // Interval of checking that agents data is up-to-date
}

var main_config *AppConfig

func (c *AppConfig) ToJson() ([]byte, error) {
	var (
		rv  []byte
		err error
	)
	if rv, err = json.Marshal(c); err != nil {
		glog.Errorln(err.Error())
	}
	return rv, err
}

func (c *AppConfig) ToYaml() ([]byte, error) {
	var (
		rv  []byte
		err error
	)
	if rv, err = yaml.Marshal(c); err != nil {
		glog.Errorln(err.Error())
	}
	return rv, err
}

func GetOrCreateConfig() *AppConfig {
	return main_config
}

func init() {
	main_config = new(AppConfig)
}
