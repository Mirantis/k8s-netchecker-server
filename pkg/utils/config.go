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
	sync.Mutex                  // extend for ensures atomic writes; protects the following fields
	configured    bool          // flag whether App configured
	UseKubeClient bool          //
	EtcdEndpoints string        // how to connect to etcd
	EtcdTree      string        // Root of NetChecker tree into etcd
	HttpListen    string        // IPaddress:PORT for HTTP control API responder
	PingTimeout   time.Duration // Timeout for ping (to etcd) operations
	ReportTTL     time.Duration // TTL for Agent report
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
