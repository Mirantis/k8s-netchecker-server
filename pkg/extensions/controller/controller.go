// Copyright 2017 Mirantis
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
    "context"

    "github.com/golang/glog"

    "k8s.io/apimachinery/pkg/fields"
    "k8s.io/apimachinery/pkg/runtime"
    apiv1 "k8s.io/client-go/pkg/api/v1"
    "k8s.io/client-go/rest"
    "k8s.io/client-go/tools/cache"

    agentv1 "github.com/Mirantis/k8s-netchecker-server/pkg/extensions/apis/agent/v1"
)

// Watcher is an example of watching on resource create/update/delete events
type AgentController struct {
    AgentClient *rest.RESTClient
    AgentScheme *runtime.Scheme
}

// Run starts an Example resource controller
func (c *AgentController) Run(ctx context.Context) error {
    glog.V(5).Info("Watch Example objects")

    // Watch Example objects
    _, err := c.watchAgents(ctx)
    if err != nil {
        glog.V(5).Infof("Failed to register watch for Example resource: %v\n", err)
        return err
    }

    <-ctx.Done()
    return ctx.Err()
}

func (c *AgentController) watchAgents(ctx context.Context) (cache.Controller, error) {
    source := cache.NewListWatchFromClient(
        c.AgentClient,
        agentv1.AgentResourcePlural,
        apiv1.NamespaceAll,
        fields.Everything())

    _, controller := cache.NewInformer(
        source,

        // The object type.
        &agentv1.Agent{},

        // resyncPeriod
        // Every resyncPeriod, all resources in the cache will retrigger events.
        // Set to 0 to disable the resync.
        0,

        // Your custom resource event handlers.
        cache.ResourceEventHandlerFuncs{
            AddFunc:    c.onAdd,
            UpdateFunc: c.onUpdate,
            DeleteFunc: c.onDelete,
        })

    go controller.Run(ctx.Done())
    return controller, nil
}

func (c *AgentController) onAdd(obj interface{}) {
    agent := obj.(*agentv1.Agent)
    glog.V(5).Infof("[CONTROLLER] OnAdd %s\n", agent.ObjectMeta.SelfLink)

    // NEVER modify objects from the store. It's a read-only, local cache.
    // You can use agentScheme.Copy() to make a deep copy of original object and modify this copy
    // Or create a copy manually for better performance
    copyObj, err := c.AgentScheme.Copy(agent)
    if err != nil {
        glog.V(5).Infof("ERROR creating a deep copy of example object: %v\n", err)
        return
    }

    agentCopy := copyObj.(*agentv1.Agent)
    agentCopy.Status = agentv1.AgentStatus{
        State:   agentv1.AgentStateProcessed,
        Message: "Successfully processed by controller",
    }

    err = c.AgentClient.Put().
        Name(agent.ObjectMeta.Name).
        Namespace(agent.ObjectMeta.Namespace).
        Resource(agentv1.AgentResourcePlural).
        Body(agentCopy).
        Do().
        Error()

    if err != nil {
        glog.V(5).Infof("ERROR updating status: %v\n", err)
    } else {
        glog.V(5).Infof("UPDATED status: %#v\n", agentCopy)
    }
}

func (c *AgentController) onUpdate(oldObj, newObj interface{}) {
    oldExample := oldObj.(*agentv1.Agent)
    newExample := newObj.(*agentv1.Agent)
    glog.V(5).Infof("[CONTROLLER] OnUpdate oldObj: %s\n", oldExample.ObjectMeta.SelfLink)
    glog.V(5).Infof("[CONTROLLER] OnUpdate newObj: %s\n", newExample.ObjectMeta.SelfLink)
}

func (c *AgentController) onDelete(obj interface{}) {
    agent := obj.(*agentv1.Agent)
    glog.V(5).Infof("[CONTROLLER] OnDelete %s\n", agent.ObjectMeta.SelfLink)
}
