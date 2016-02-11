// Copyright 2015 Google Inc. All Rights Reserved.
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

package main

import (
	//	"bytes"
	"errors"
	"fmt"
	//  "time"
	"strings"

	"k8s.io/kubernetes/pkg/api"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/fields"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	kubectlResource "k8s.io/kubernetes/pkg/kubectl/resource"
	"k8s.io/kubernetes/pkg/labels"
	//  "k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/api/unversioned"
	//	"k8s.io/kubernetes/pkg/kubectl"
	//    "k8s.io/kubernetes/pkg/util/intstr"
)

/*
curl -XPOST -H "Content-Type:application/json;charset=UTF-8"  -d '{"name": "brun2-v2.0.0", "content": "apiVersion: v1\nkind: ReplicationController\nmetadata:\n  labels:\n    app: brun2-v1.0.0\n    version: v2.0.0\n  name: brun2-v2.0.0\nspec:\n  replicas: 4\n  selector:\n    app: brun2-v2.0.0\n  template:\n    metadata:\n      labels:\n        app: brun2-v2.0.0\n    spec:\n      nodeSelector:\n        brun: \"true\"\n        brun_version: v1.0.0\n      containers:\n      - image: busybox\n        command:\n          - sleep\n          - \"360000\"\n        name: busybox\n        resources:\n          requests:\n            memory: \"128Mi\"\n            cpu: \"1\"\n          limits:\n            memory: \"128Mi\"\n            cpu: \"1\"\n" }' http://127.0.0.1:9090/api/replicationcontrollers/brun2/brun2-v1.0.0/rolling-update
*/

// Updates number of replicas in Replication Controller based on Replication Controller Spec
func RollingUpdateByNodeReplicationController(client client.Interface, namespace, oldName string,
	newRCSpec *AppDeploymentFromFileSpec) error {
	// TODO get it from requests
	rollingUpdateLabel := "brun2_version"

	// Set duration in seconds before the object should be deleted
	podsDeleteOptions := api.NewDeleteOptions(int64(5))

	// Get old RC
	oldRc, err := client.ReplicationControllers(namespace).Get(oldName)
	if err != nil {
		return err
	}
	// Check if the old RC has the rollingUpdateLabel
	oldRollingUpdateLabelValue, ok := oldRc.Spec.Template.Spec.NodeSelector[rollingUpdateLabel]
	if ok != true {
		return errors.New(fmt.Sprintf("RC '%s'; have not NodeSelector: '%s'", oldRc.Name, rollingUpdateLabel))
	}
	// Get new RC
	const (
		validate      = true
		emptyCacheDir = ""
	)
	factory := cmdutil.NewFactory(nil)
	schema, err := factory.Validator(validate, emptyCacheDir)
	if err != nil {
		return err
	}
	mapper, typer := factory.Object()
	reader := strings.NewReader(newRCSpec.Content)
	if namespace == "" {
		namespace = api.NamespaceDefault
	}

	r := kubectlResource.NewBuilder(mapper, typer, factory.ClientMapperForCommand()).
		Schema(schema).
		NamespaceParam(api.NamespaceDefault).DefaultNamespace().
		Stream(reader, newRCSpec.Name).
		Flatten().
		Do()
	obj, err := r.Object()
	if err != nil {
		return err
	}
	// Handle filename input from stdin. The resource builder always returns an api.List
	// when creating resource(s) from a stream.
	if list, ok := obj.(*api.List); ok {
		if len(list.Items) > 1 {
			return errors.New("Sent file specifies multiple items")
		}
		obj = list.Items[0]
	}
	newRc, ok := obj.(*api.ReplicationController)
	if !ok {
		return errors.New("Sent file has a bad format")
	}
	// Check if the new RC has the rollingUpdateLabel
	newRollingUpdateLabelValue, ok := newRc.Spec.Template.Spec.NodeSelector[rollingUpdateLabel]
	if ok != true {
		return errors.New(fmt.Sprintf("The new RC '%s'; have not NodeSelector: '%s'", newRc.Name, rollingUpdateLabel))
	}
	// Set same namespace for the new Rc
	newRc.Namespace = oldRc.Namespace

	// Node label selector
	// TODO find a better way to get all nodes
	label := labels.SelectorFromSet(labels.Set(map[string]string{rollingUpdateLabel: oldRollingUpdateLabelValue}))
	listoptions := unversioned.ListOptions{
		LabelSelector: unversioned.LabelSelector{label},
		FieldSelector: unversioned.FieldSelector{fields.Everything()},
	}
	// Get nodes
	nodeList, err := client.Nodes().List(listoptions)
	if err != nil {
		return err
	}
	if len(nodeList.Items) == 0 {
		return errors.New(fmt.Sprintf("No node with label '%s' found", rollingUpdateLabel))
	}

	// Beginning rolling update by node
	// First create the new RC
	_, err = client.ReplicationControllers(namespace).Create(newRc)
	if err != nil {
		return err
	}
	// browsing node by node
	for _, node := range nodeList.Items {
		// Get last version of the current node
		node, err := client.Nodes().Get(node.Name)
		// Set label brun_version to v0.0.0 to the current node
		// TODO replace "v0.0.0" by random stuff
		node.ObjectMeta.Labels[rollingUpdateLabel] = "v0.0.0"
		node, err = client.Nodes().Update(node)
		if err != nil {
			return err
		}
		// Get all pods
		// Counting nb of pods running on this node
		// TODO save pod number in annotation ???
		nbPods := 0
		// Get all pods from current nodes and current RC
		oldPodsLabel := labels.SelectorFromSet(oldRc.Spec.Selector)
		oldPodsListOptions := unversioned.ListOptions{
			LabelSelector: unversioned.LabelSelector{oldPodsLabel},
			FieldSelector: unversioned.FieldSelector{fields.Everything()},
		}
		podList, err := client.Pods(namespace).List(oldPodsListOptions)
		if err != nil {
			return err
		}
		if len(podList.Items) == 0 {
			//TODO no pods found !?
			//return errors.New
		}
		for _, pod := range podList.Items {
			if node.ObjectMeta.Name == pod.Spec.NodeName {
				// Counting nb of pods running on this node
				nbPods += 1
				// Set label on rolling upgrade
				pod.ObjectMeta.Labels["on_rolling_update_by_node"] = "true"
				client.Pods(namespace).Update(&pod)
				// Delete pod from the current node
				client.Pods(namespace).Delete(pod.ObjectMeta.Name, podsDeleteOptions)
			}
		}
		// Waiting for pods deletion
		// Cleaning
		toto := make(map[string]string)
		for k, v := range oldRc.Spec.Selector {
			toto[k] = v
		}
		toto["on_rolling_update_by_node"] = "true"
		podlabel3 := labels.SelectorFromSet(toto)
		listoptions3 := unversioned.ListOptions{
			LabelSelector: unversioned.LabelSelector{podlabel3},
			FieldSelector: unversioned.FieldSelector{fields.Everything()},
		}
		watcher, _ := client.Pods(namespace).Watch(listoptions3)
		for nbPods > 0 {
			// Waiting for events
			<-watcher.ResultChan()
			// init counter
			nbPods = 0
			// Counting pod deleted still running on this node
			podList, _ = client.Pods(namespace).List(listoptions3)
			for _, pod := range podList.Items {
				if pod.Spec.NodeName == node.Name {
					nbPods += 1
				}
			}
		}
		// Here all pods (with the current RC) from the current node are stopped

		// Get last version of the current node
		node, err = client.Nodes().Get(node.Name)
		// Set the new label to the current node
		node.ObjectMeta.Labels[rollingUpdateLabel] = newRollingUpdateLabelValue
		node, err = client.Nodes().Update(node)
		// We have to wait that all pods are RUNNING on the current node
		// and for each events check the number of NON RUNNING pods on the current node
		podlabel4 := labels.SelectorFromSet(newRc.Spec.Selector)
		listoptions4 := unversioned.ListOptions{
			LabelSelector: unversioned.LabelSelector{podlabel4},
			FieldSelector: unversioned.FieldSelector{fields.Everything()},
		}
		watcher, _ = client.Pods(namespace).Watch(listoptions4)
		nonrunningPods := 0
		runningPods := 0
		// TODO Maybe we need to check readiness/health of each pod...
		for runningPods == 0 || nonrunningPods > 0 {
			// Waiting for events
			<-watcher.ResultChan()
			// init counter
			nonrunningPods = 0
			runningPods = 0
			// Counting
			podList, _ = client.Pods(namespace).List(listoptions4)
			for _, pod := range podList.Items {
				if pod.Spec.NodeName == node.Name {
					if pod.Status.Phase != api.PodRunning {
						nonrunningPods += 1
					}
					if pod.Status.Phase == api.PodRunning {
						runningPods += 1
					}
				}
			}
		}
		// Update finished on the current node
		// Go to the next one
	}
	// Rolling Update by Node finished

	// Delete the old RC
	err = client.ReplicationControllers(namespace).Delete(oldRc.Name)
	if err != nil {
		return err
	}
	// Delete old pods
	podlabel2 := labels.SelectorFromSet(oldRc.Spec.Selector)
	listoptions2 := unversioned.ListOptions{
		LabelSelector: unversioned.LabelSelector{podlabel2},
		FieldSelector: unversioned.FieldSelector{fields.Everything()},
	}
	podList, err := client.Pods(namespace).List(listoptions2)
	for _, pod := range podList.Items {
		client.Pods(namespace).Delete(pod.ObjectMeta.Name, podsDeleteOptions)
	}
	// TODO wait for deleting old pods

	return nil
}
