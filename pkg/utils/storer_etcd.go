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
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	ext_v1 "github.com/Mirantis/k8s-netchecker-server/pkg/extensions/apis/v1"
	ext_client "github.com/Mirantis/k8s-netchecker-server/pkg/extensions/client"

	etcd "github.com/coreos/etcd/client"
	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
)

type EtcdConfig struct {
	client etcd.Client
	kAPI   etcd.KeysAPI
}

type K8sConnection struct {
	KubeClient          Proxy
	ExtensionsClientset ext_client.Clientset
}

type EtcdAgentStorage struct {
	sync.Mutex   // extend for ensures atomic writes; protects the following fields
	config       *AppConfig
	etcd         EtcdConfig
	k8s          K8sConnection
	NcAgentCache NcAgentCache
}

func NewEtcdStorer() (*EtcdAgentStorage, error) {
	var err error

	cfg := GetOrCreateConfig()
	glog.Infof("Endpoints '%s' will be used for connect to etcd.", cfg.EtcdEndpoints)

	rv := &EtcdAgentStorage{
		NcAgentCache: NcAgentCache{},
		config:       cfg,
	}

	// setup http/https transport, compatible with self-signed certs
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	httpsTransport := &http.Transport{
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     tlsConfig,
	}

	etcdConfig := etcd.Config{
		Endpoints: strings.Split(cfg.EtcdEndpoints, ","),
		Transport: httpsTransport,
	}

	// Configure ETCD client
	if rv.etcd.client, err = etcd.New(etcdConfig); err != nil {
		return nil, err
	}
	rv.etcd.kAPI = etcd.NewKeysAPI(rv.etcd.client)

	if err = rv.PingETCD(); err != nil {
		return nil, err
	}

	// Configure connection to k8s API
	rv.k8s.KubeClient, rv.k8s.ExtensionsClientset, err = connect2k8s()

	return rv, err
}

func (s *EtcdAgentStorage) PingETCD() error {
	var rv error
	ctx, cancel := context.WithTimeout(context.Background(), s.config.PingTimeout)
	defer cancel()
	// set a new key, ignoring it's previous state
	_, err := s.etcd.kAPI.Set(ctx, fmt.Sprintf("%s/ping", s.config.EtcdTree), "pong", nil)
	if err != nil {
		if err == context.DeadlineExceeded {
			rv = fmt.Errorf("Ping has no answer more than %d seconds", s.config.PingTimeout)
		} else {
			rv = fmt.Errorf("Ping to etcd failed: %v", err.Error())
		}
	}
	return rv
}

func (s *EtcdAgentStorage) agentTreeRoot(agentName string) string {
	return fmt.Sprintf("%s/agents/%s", s.config.EtcdTree, agentName)
}

func (s *EtcdAgentStorage) agentReportNodeName(agentData *ext_v1.AgentSpec) string {
	return fmt.Sprintf("%s/%d", s.agentTreeRoot(agentData.PodName), agentData.Uptime)
}

func (s *EtcdAgentStorage) createOrUpdateAgentTree(ctx context.Context, dirName string, update bool) {
	var (
		prevExists etcd.PrevExistType
		oper       string
		refresh    bool
	)
	if update {
		prevExists = etcd.PrevExist
		oper = "TTL update"
		refresh = true
	} else {
		prevExists = etcd.PrevNoExist
		refresh = false
		oper = "create"
	}
	_, err := s.etcd.kAPI.Set(ctx, dirName, "", &etcd.SetOptions{
		Dir:       true,
		PrevExist: prevExists,
		Refresh:   refresh,
		TTL:       s.config.ReportTTL,
	})
	if err != nil {
		glog.Errorf("Updating DIR '%s' failed: %v", dirName, err)
	} else {
		glog.Infof("DIR '%s' %sd successfully", dirName, oper)
	}
}

func (s *EtcdAgentStorage) checkOrCreateAgentTree(ctx context.Context, dirName string) {
	resp, err := s.etcd.kAPI.Get(ctx, dirName, &etcd.GetOptions{Quorum: true})
	if err == nil && !resp.Node.Dir {
		glog.Errorf("Key '%s' exists, but it's not a directory! Key will be re-created", dirName)
		if _, err := s.etcd.kAPI.Delete(ctx, dirName, &etcd.DeleteOptions{Dir: false}); err != nil {
			glog.Errorf("Can't remove etcd node: %v", err)
		} else {
			s.createOrUpdateAgentTree(ctx, dirName, false)
		}
	} else if err == nil && resp.Node.Dir {
		// all OK, update TTL
		s.createOrUpdateAgentTree(ctx, dirName, true)
	} else if err != nil && etcd.IsKeyNotFound(err) {
		// key not found, create directory
		s.createOrUpdateAgentTree(ctx, dirName, false)
	} else {
		glog.Fatalf("Unhandled error with etcd data structure '%s' failed: %v", dirName, err)
	}
}

func (s *EtcdAgentStorage) UpdateAgents(rw http.ResponseWriter, r *http.Request, rp httprouter.Params) (ext_v1.AgentSpec, error) {
	var err error
	agentData := ext_v1.AgentSpec{}

	if err = ProcessRequest(r, &agentData, rw); err != nil {
		return ext_v1.AgentSpec{}, err
	}

	agentData.LastUpdated = time.Now()
	glog.V(10).Infof("Updating the agents resource with value: %v", agentData)

	dirName := s.agentTreeRoot(agentData.PodName)
	nodeName := s.agentReportNodeName(&agentData)

	ctx := context.Background() // TODO: handle timeout for bunch of operations

	// create/update agent's derectory
	s.checkOrCreateAgentTree(ctx, dirName)

	// create report node for agent
	hhj, _ := json.Marshal(agentData)
	_, err = s.etcd.kAPI.Set(ctx, nodeName, string(hhj), &etcd.SetOptions{
		Dir:       false,
		PrevExist: etcd.PrevNoExist,
		TTL:       s.config.ReportTTL,
	})
	if err != nil {
		glog.Errorf("Creating REC '%s' failed: %v", nodeName, err)
	} else {
		glog.Infof("Record '%s' created successfully", nodeName)
	}

	return agentData, nil
}

func (s *EtcdAgentStorage) getAgents() NcAgentCache {
	var (
		dirName        string
		agentsData     NcAgentCache
		max_AgentSpec  ext_v1.AgentSpec
		last_AgentSpec ext_v1.AgentSpec
		err            error
	)

	agentsData = NcAgentCache{}
	dirName = fmt.Sprintf("%s/agents", s.config.EtcdTree)

	ctx := context.Background()
	resp, err := s.etcd.kAPI.Get(ctx, dirName, &etcd.GetOptions{Quorum: true, Recursive: true})
	if err != nil {
		glog.Errorf("Can't fetch tree '%s' recursively from etcd: %v", dirName, err)
		return agentsData
	}

	// Iterate to nodes
	for _, node := range resp.Node.Nodes {
		npath := strings.Split(node.Key, "/")
		nname := npath[len(npath)-1]
		max_AgentSpec.Uptime = 0
		// Iterate to uptime records
		for _, n := range node.Nodes {
			if err = json.Unmarshal([]byte(n.Value), &last_AgentSpec); err != nil {
				glog.Error(err)
				continue
			}
			if last_AgentSpec.Uptime > max_AgentSpec.Uptime {
				max_AgentSpec = last_AgentSpec
			}
		}
		if max_AgentSpec.Uptime > 0 {
			agentsData[nname] = max_AgentSpec
			glog.Infof("%s: %#v", nname, last_AgentSpec)
		}
	}
	return agentsData
}

func (s *EtcdAgentStorage) GetAgents(rw http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ProcessResponse(rw, s.getAgents())
}

func (s *EtcdAgentStorage) getSingleAgent(name string) *ext_v1.AgentSpec {
	agentsData := s.getAgents()
	agentData := agentsData[name]
	return &agentData

}

func (s *EtcdAgentStorage) GetSingleAgent(rw http.ResponseWriter, r *http.Request, rp httprouter.Params) {
	agentName := rp.ByName("name")
	agentData := s.getSingleAgent(agentName)

	ProcessResponse(rw, agentData)
}

func (s *EtcdAgentStorage) CheckAgents() ([]string, []string, error) {

	absent := []string{}
	agents := s.getAgents()

	pods, err := s.k8s.KubeClient.Pods()
	if err != nil {
		return nil, nil, err
	}

	for _, pod := range pods.Items {
		agentName := pod.ObjectMeta.Name
		if _, ok := agents[agentName]; !ok {
			absent = append(absent, agentName)
		}
	}

	return absent, nil, nil
}

func (s *EtcdAgentStorage) AgentCache() NcAgentCache {
	rv := s.getAgents()
	return rv
}

func (h *EtcdAgentStorage) AgentCacheUpdate(key string, ag *ext_v1.Agent) {
	//todo: Whether should I implement this, or not???
}

func (h *EtcdAgentStorage) CleanCacheOnDemand(rw http.ResponseWriter) {
	// Do nothing, because no cache.
	// All data auto-purged by ETCD TTL feature
}
