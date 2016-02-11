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
	//	"k8s.io/kubernetes/pkg/kubectl"
	//    "k8s.io/kubernetes/pkg/util/intstr"
)

// Specification for deployment from file
type AppUpdateByNodeFromFileSpec struct {
	// Name of the file
	Name string `json:"name"`

	// File content
	Content string `json:"content"`

	// Node Label name
	NodeLabel string `json:"nodelabel"`
}

/*
curl -XPOST -H "Content-Type:application/json;charset=UTF-8"  -d '{"nodelabel": "brun2_version", "name": "brun2-v2.0.0", "content": "apiVersion: v1\nkind: ReplicationController\nmetadata:\n  labels:\n    app: brun2-v2.0.0\n    version: v2.0.0\n  name: brun2-v2.0.0\nspec:\n  replicas: 8\n  selector:\n    app: brun2-v2.0.0\n  template:\n    metadata:\n      labels:\n        app: brun2-v2.0.0\n    spec:\n      nodeSelector:\n        brun: \"true\"\n        brun2_version: v2.0.0\n      containers:\n      - image: busybox\n        command:\n          - sleep\n          - \"360000\"\n        name: busybox\n        resources:\n          requests:\n            memory: \"128Mi\"\n            cpu: \"1\"\n          limits:\n            memory: \"128Mi\"\n            cpu: \"1\"\n" }' http://127.0.0.1:9090/api/replicationcontrollers/brun2/brun2-v1.0.0/rolling-update-by-node
*/

// Updates number of replicas in Replication Controller based on Replication Controller Spec
func RollingUpdateByNodeReplicationController(client client.Interface, namespace, oldName string,
	newRCSpec *AppUpdateByNodeFromFileSpec) error {
	// Get nodelabel from POST data
	rollingUpdateLabel := newRCSpec.NodeLabel
	if rollingUpdateLabel == "" {
		return errors.New(fmt.Sprintf("Node label name is empty"))
	}

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

	r := kubectlResource.NewBuilder(mapper, typer, kubectlResource.ClientMapperFunc(factory.ClientForMapping), factory.Decoder(true)).
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
	// Set replicas number to 0
	newReplicasNumber := newRc.Spec.Replicas
	newRc.Spec.Replicas = 0

	// Node label selector
	label := labels.SelectorFromSet(labels.Set(map[string]string{rollingUpdateLabel: oldRollingUpdateLabelValue}))
	listOptions := api.ListOptions{
		LabelSelector: label,
		FieldSelector: fields.Everything(),
	}
	// Get nodes
	nodeList, err := client.Nodes().List(listOptions)
	if err != nil {
		return err
	}
	if len(nodeList.Items) == 0 {
		return errors.New(fmt.Sprintf("No node with label '%s' found", rollingUpdateLabel))
	}
	// Prepare labels and fields for old pods
	oldPodsLabel := labels.SelectorFromSet(oldRc.Spec.Selector)
	allOldPodsListOptions := api.ListOptions{
		LabelSelector: oldPodsLabel,
		FieldSelector: fields.Everything(),
	}
	// Get the number of pod running the old version
	podList, err := client.Pods(namespace).List(allOldPodsListOptions)
	if err != nil {
		return err
	}
	// Get nb of old pods by nodes
	nbPodByNode := make(map[string]int)
	for _, pod := range podList.Items {
		if _, ok := nbPodByNode[pod.Spec.NodeName]; !ok {
			nbPodByNode[pod.Spec.NodeName] = 0
		}
		nbPodByNode[pod.Spec.NodeName] += 1
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
		// Set label nodelabel to v0.0.0 to the current node
		// TODO replace "v0.0.0" by random stuff
		node.ObjectMeta.Labels[rollingUpdateLabel] = "v0.0.0"
		node, err = client.Nodes().Update(node)
		if err != nil {
			return err
		}
		// Counting nb of pods running on this node
		// TODO save pod number in annotation ???
		oldPodNumber := 0
		oldPodNumber, _ = nbPodByNode[node.Name]
		/* For now we do NOT decrease replicas of oldRC
		      To be sure to not restart a pod which can
		      be delete few minutes later
		   // Decrease the number of replicas of the old Replication Controller
		   oldRc, err = client.ReplicationControllers(namespace).Get(oldRc.Name)
		   if err != nil {
		       return err
		   }
		   oldRc.Spec.Replicas = oldRc.Spec.Replicas - oldPodNumber
		   if oldRc.Spec.Replicas < 0 {
		       oldRc.Spec.Replicas = 0
		   }
		   _, err = client.ReplicationControllers(namespace).Update(oldRc)
		   if err != nil {
		       return err
		   }
		*/
		// Get all pods from current nodes and current RC
		podsFields := fields.Set{"spec.nodeName": node.Name}
		oldPodsListOptions := api.ListOptions{
			LabelSelector: oldPodsLabel,
			FieldSelector: podsFields.AsSelector(),
		}
		podList, err = client.Pods(namespace).List(oldPodsListOptions)
		if err != nil {
			return err
		}
		// Delete old pods
		nbPods := 0
		for _, pod := range podList.Items {
			// Counting nb of pods running on this node
			nbPods += 1
			// Set label on rolling upgrade
			client.Pods(namespace).Update(&pod)
			// Delete pod from the current node
			client.Pods(namespace).Delete(pod.ObjectMeta.Name, podsDeleteOptions)
		}
		if oldPodNumber > 0 {
			// Waiting for pods deletion
			// Cleaning
			watcher, _ := client.Pods(namespace).Watch(oldPodsListOptions)
			for nbPods > 0 {
				// Waiting for events
				<-watcher.ResultChan()
				// init counter
				nbPods = 0
				// Counting pod deleted still running on this node
				podList, _ = client.Pods(namespace).List(oldPodsListOptions)
				nbPods = len(podList.Items)
			}
		}
		// Here all pods (with the current RC) from the current node are stopped

		// Increase the number of replicas of the new Replication Controller
		newRc, err = client.ReplicationControllers(namespace).Get(newRc.Name)
		if err != nil {
			return err
		}
		newRc.Spec.Replicas = newRc.Spec.Replicas + oldPodNumber
		if newRc.Spec.Replicas > newReplicasNumber {
			newRc.Spec.Replicas = newReplicasNumber
		}
		_, err = client.ReplicationControllers(namespace).Update(newRc)
		if err != nil {
			return err
		}
		// Get last version of the current node
		node, err = client.Nodes().Get(node.Name)
		// Set the new label to the current node
		node.ObjectMeta.Labels[rollingUpdateLabel] = newRollingUpdateLabelValue
		node, err = client.Nodes().Update(node)

		if oldPodNumber > 0 {
			// We have to wait that all pods are RUNNING on the current node
			// and for each events check the number of NON RUNNING pods on the current node
			newPodsLabel := labels.SelectorFromSet(newRc.Spec.Selector)
			NewPodsListOptions := api.ListOptions{
				LabelSelector: newPodsLabel,
				FieldSelector: fields.Everything(),
			}
			watcher, _ := client.Pods(namespace).Watch(NewPodsListOptions)
			runningPods := 0
			// TODO Maybe we need to check readiness/health of each pod...
			for runningPods < newRc.Spec.Replicas {
				// Waiting for events
				<-watcher.ResultChan()
				// init counter
				runningPods = 0
				// Counting
				podList, _ = client.Pods(namespace).List(NewPodsListOptions)
				for _, pod := range podList.Items {
					if api.IsPodReady(&pod) {
						runningPods += 1
					}
				}
			}
		}
		// Update finished on the current node
		// Go to the next one
	}
	// Rolling Update by Node finished

	// Scale to newReplicasNumber is too low
	if newRc.Spec.Replicas < newReplicasNumber {
		newRc, err = client.ReplicationControllers(namespace).Get(newRc.Name)
		if err != nil {
			return err
		}
		newRc.Spec.Replicas = newReplicasNumber
		_, err = client.ReplicationControllers(namespace).Update(newRc)
		if err != nil {
			return err
		}
	}

	// Delete the old RC
	err = client.ReplicationControllers(namespace).Delete(oldRc.Name)
	if err != nil {
		return err
	}
	// Delete old pods
	podList, err = client.Pods(namespace).List(allOldPodsListOptions)
	for _, pod := range podList.Items {
		client.Pods(namespace).Delete(pod.ObjectMeta.Name, podsDeleteOptions)
	}
	// TODO wait for deleting old pods

	return nil
}
