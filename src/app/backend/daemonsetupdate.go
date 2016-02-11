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
	//  "time"
	"strings"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/fields"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	kubectlResource "k8s.io/kubernetes/pkg/kubectl/resource"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/watch"
	//	"k8s.io/kubernetes/pkg/kubectl"
	//    "k8s.io/kubernetes/pkg/util/intstr"
)

/*
curl -XPOST -H "Content-Type:application/json;charset=UTF-8"  -d '{"name": "brun2-v2.0.0", "content": "apiVersion: v1\nkind: ReplicationController\nmetadata:\n  labels:\n    app: brun2-v1.0.0\n    version: v2.0.0\n  name: brun2-v2.0.0\nspec:\n  replicas: 4\n  selector:\n    app: brun2-v2.0.0\n  template:\n    metadata:\n      labels:\n        app: brun2-v2.0.0\n    spec:\n      nodeSelector:\n        brun: \"true\"\n        brun_version: v1.0.0\n      containers:\n      - image: busybox\n        command:\n          - sleep\n          - \"360000\"\n        name: busybox\n        resources:\n          requests:\n            memory: \"128Mi\"\n            cpu: \"1\"\n          limits:\n            memory: \"128Mi\"\n            cpu: \"1\"\n" }' http://127.0.0.1:9090/api/replicationcontrollers/brun2/brun2-v1.0.0/rolling-update
*/

// Updates number of replicas in Replication Controller based on Replication Controller Spec
func RollingUpdateDaemonSet(apiclient client.Interface, namespace, oldDsName string,
	newDsSpec *AppDeploymentFromFileSpec) error {
	// Set duration in seconds before the object should be deleted
	podsDeleteOptions := api.NewDeleteOptions(int64(5))
	// Get old DS
	oldDs, err := apiclient.Extensions().DaemonSets(namespace).Get(oldDsName)
	if err != nil {
		return err
	}

	// Get new DS
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
	reader := strings.NewReader(newDsSpec.Content)
	if namespace == "" {
		namespace = api.NamespaceDefault
	}

	r := kubectlResource.NewBuilder(mapper, typer, kubectlResource.ClientMapperFunc(factory.ClientForMapping), factory.Decoder(true)).
		Schema(schema).
		NamespaceParam(api.NamespaceDefault).DefaultNamespace().
		Stream(reader, newDsSpec.Name).
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
	newDs, ok := obj.(*extensions.DaemonSet)
	if !ok {
		return errors.New("Sent file has a bad format")
	}
	// Create the new DS
	_, err = apiclient.Extensions().DaemonSets(namespace).Create(newDs)
	if err != nil {
		return err
	}

	// Get all pods from the old DS
	/*
	   // TODO We should get DS selector with the following line
	   // But it doesn't work, we always get empty map
	   selector, err := extensions.LabelSelectorAsSelector(oldDs.Spec.Selector)
	   if err != nil {
	         return err
	   }
	*/
	// So we use pod template instead ... Could be dangerous...
	podDSLabelOld := labels.SelectorFromSet(labels.Set(oldDs.Spec.Template.Labels))

	listoptions := api.ListOptions{
		LabelSelector: podDSLabelOld,
		FieldSelector: fields.Everything(),
	}
	podOldList, err := apiclient.Pods(namespace).List(listoptions)
	if err != nil {
		return err
	}

	// Delete the old DS
	err = apiclient.Extensions().DaemonSets(namespace).Delete(oldDs.Name)
	if err != nil {
		return err
	}
	// TODO BLOCK until DS is deleted

	// Iterate on all pods
	for _, pod := range podOldList.Items {
		// Deleting pod
		// Pod label to filter
		podLabelOld := labels.SelectorFromSet(pod.Labels)
		fieldSelector, err := fields.ParseSelector("metadata.name=" + pod.Name)
		if err != nil {
			return err
		}

		// Watch for event with the label of the current pod
		listoptions2 := api.ListOptions{
			LabelSelector: podLabelOld,
			FieldSelector: fieldSelector,
		}
		watcherDelete, _ := apiclient.Pods(namespace).Watch(listoptions2)
		// Delete pod
		apiclient.Pods(namespace).Delete(pod.ObjectMeta.Name, podsDeleteOptions)
		// Waiting for pod deletion
		event := <-watcherDelete.ResultChan()
		for event.Type != watch.Deleted {
			event = <-watcherDelete.ResultChan()
		}

		// Preparing to wait pod creation
		podlabelNew := labels.SelectorFromSet(newDs.Spec.Template.Labels)
		fieldSelector2, err := fields.ParseSelector("spec.nodeName=" + pod.Spec.NodeName)

		listoptions4 := api.ListOptions{
			LabelSelector: podlabelNew,
			FieldSelector: fieldSelector2,
		}
		watcherCreate, _ := apiclient.Pods(namespace).Watch(listoptions4)

		// Waiting for pod creation
		running := false
		for !running {
			<-watcherCreate.ResultChan()
			podOldList, _ = apiclient.Pods(namespace).List(listoptions4)
			for _, pod := range podOldList.Items {
				// Wait for the pod to be ready
				if api.IsPodReady(&pod) {
					running = true
				}
			}
		}

	}

	return nil
}
