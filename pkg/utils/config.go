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
	"fmt"
	"github.com/golang/glog"
	"gopkg.in/yaml.v2"
	"sync"
	"time"
)

type AppConfig struct {
	sync.Mutex            // extend for ensures atomic writes; protects the following fields
	configured     bool   // flag whether App configured
	UseKubeClient  bool   //
	EtcdEndpoints  string // how to connect to etcd
	HttpListen     string // IPaddress:PORT for HTTP control API responder
	PingTimeout    time.Duration
	ReportTimeout  time.Duration
	ReportTTL      time.Duration
	ReportInterval time.Duration
}

var main_config *AppConfig

func (c *AppConfig) ToJson() ([]byte, error) {
	var (
		rv  []byte
		err error
	)
	if rv, err = json.Marshal(c); err != nil {
		//rv = []byte([])
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
		//rv = []
		glog.Errorln(err.Error())
	}
	return rv, err
}

func (c *AppConfig) String() string {
	var (
		rv  string
		brv []byte
		err error
	)
	if brv, err = c.ToYaml(); err != nil {
		rv = fmt.Sprintf("---\nerror: %v", err.Error())
		glog.Errorln(err.Error())
	} else {
		rv = fmt.Sprintf("%s", brv)
	}
	return rv
}

func GetOrCreateConfig() *AppConfig {
	if !main_config.configured {
		main_config.PingTimeout = 5 * time.Second
		main_config.ReportTimeout = 10 * time.Second
		main_config.ReportTTL = 5 * time.Minute
		main_config.ReportInterval = 20 * time.Second
		main_config.configured = true
	}
	return main_config
}

func init() {
	main_config = new(AppConfig)
}
