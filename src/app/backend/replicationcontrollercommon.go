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
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
)

// ReplicationControllerWithPods is a structure representing replication controller and its pods.
type ReplicationControllerWithPods struct {
	ReplicationController *api.ReplicationController
	Pods                  *api.PodList
}

// ReplicationControllerPodInfo represents aggregate information about replication controller pods.
type ReplicationControllerPodInfo struct {
	// Number of pods that are created.
	Current int32 `json:"current"`

	// Number of pods that are desired in this Replication Controller.
	Desired int32 `json:"desired"`

	// Number of pods that are currently running.
	Running int32 `json:"running"`

	// Number of pods that are currently waiting.
	Pending int32 `json:"pending"`

	// Number of pods that are failed.
	Failed int32 `json:"failed"`

	// Unique warning messages related to pods in this Replication Controller.
	Warnings []Event `json:"warnings"`
}

// Returns structure containing ReplicationController and Pods for the given replication controller.
func getRawReplicationControllerWithPods(client client.Interface, namespace, name string) (
	*ReplicationControllerWithPods, error) {
	replicationController, err := client.ReplicationControllers(namespace).Get(name)
	if err != nil {
		return nil, err
	}

	labelSelector := labels.SelectorFromSet(replicationController.Spec.Selector)
	pods, err := client.Pods(namespace).List(
		api.ListOptions{
			LabelSelector: labelSelector,
			FieldSelector: fields.Everything(),
		})

	if err != nil {
		return nil, err
	}

	replicationControllerAndPods := &ReplicationControllerWithPods{
		ReplicationController: replicationController,
		Pods: pods,
	}
	return replicationControllerAndPods, nil
}

// Retrieves Pod list that belongs to a Replication Controller.
func getRawReplicationControllerPods(client client.Interface, namespace, name string) (*api.PodList, error) {
	replicationControllerAndPods, err := getRawReplicationControllerWithPods(client, namespace, name)
	if err != nil {
		return nil, err
	}
	return replicationControllerAndPods.Pods, nil
}

// Returns aggregate information about replication controller pods.
func getReplicationControllerPodInfo(replicationController *api.ReplicationController, pods []api.Pod) ReplicationControllerPodInfo {
	result := ReplicationControllerPodInfo{
		Current: replicationController.Status.Replicas,
		Desired: replicationController.Spec.Replicas,
	}

	for _, pod := range pods {
		switch pod.Status.Phase {
		case api.PodRunning:
			result.Running++
		case api.PodPending:
			result.Pending++
		case api.PodFailed:
			result.Failed++
		}
	}

	return result
}

// Transforms simple selector map to labels.Selector object that can be used when querying for
// object.
func toLabelSelector(selector map[string]string) (labels.Selector, error) {
	labelSelector, err := unversioned.LabelSelectorAsSelector(&unversioned.LabelSelector{MatchLabels: selector})

	if err != nil {
		return nil, err
	}

	return labelSelector, nil
}

// Based on given selector returns list of services that are candidates for deletion.
// Services are matched by replication controllers' label selector. They are deleted if given
// label selector is targeting only 1 replication controller.
func getServicesForDeletion(client client.Interface, labelSelector labels.Selector,
	namespace string) ([]api.Service, error) {

	replicationControllers, err := client.ReplicationControllers(namespace).List(api.ListOptions{
		LabelSelector: labelSelector,
		FieldSelector: fields.Everything(),
	})
	if err != nil {
		return nil, err
	}

	// if label selector is targeting only 1 replication controller
	// then we can delete services targeted by this label selector,
	// otherwise we can not delete any services so just return empty list
	if len(replicationControllers.Items) != 1 {
		return []api.Service{}, nil
	}

	services, err := client.Services(namespace).List(api.ListOptions{
		LabelSelector: labelSelector,
		FieldSelector: fields.Everything(),
	})
	if err != nil {
		return nil, err
	}

	return services.Items, nil
}
