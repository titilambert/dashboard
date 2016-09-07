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
	"bytes"
	"log"

	"k8s.io/kubernetes/pkg/api"
	unversioned "k8s.io/kubernetes/pkg/api/unversioned"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
)

// ReplicationControllerDetail represents detailed information about a Replication Controller.
type ReplicationControllerDetail struct {
	// Name of the Replication Controller.
	Name string `json:"name"`

	// Namespace the Replication Controller is in.
	Namespace string `json:"namespace"`

	// Label mapping of the Replication Controller.
	Labels map[string]string `json:"labels"`

	// Label selector of the Replication Controller.
	LabelSelector map[string]string `json:"labelSelector"`

	// Container image list of the pod template specified by this Replication Controller.
	ContainerImages []string `json:"containerImages"`

	// Aggregate information about pods of this replication controller.
	PodInfo ReplicationControllerPodInfo `json:"podInfo"`

	// Detailed information about Pods belonging to this Replication Controller.
	Pods []ReplicationControllerPod `json:"pods"`

	// Detailed information about service related to Replication Controller.
	Services []ServiceDetail `json:"services"`

	// True when the data contains at least one pod with metrics information, false otherwise.
	HasMetrics bool `json:"hasMetrics"`
}

// ReplicationControllerPod is a representation of a Pod that belongs to a Replication Controller.
type ReplicationControllerPod struct {
	// Name of the Pod.
	Name string `json:"name"`

	// Status of the Pod. See Kubernetes API for reference.
	PodPhase api.PodPhase `json:"podPhase"`

	// Time the Pod has started. Empty if not started.
	StartTime unversioned.Time `json:"startTime"`

	// IP address of the Pod.
	PodIP string `json:"podIP"`

	// Name of the Node this Pod runs on.
	NodeName string `json:"nodeName"`

	// Count of containers restarts.
	RestartCount int `json:"restartCount"`

	// Pod metrics.
	Metrics *PodMetrics `json:"metrics"`
}

// ServiceDetail is a representation of a Service connected to Replication Controller.
type ServiceDetail struct {
	// Name of the service.
	Name string `json:"name"`

	// Internal endpoints of all Kubernetes services that have the same label selector as
	// connected Replication Controller.
	// Endpoint is DNS name merged with ports.
	InternalEndpoint Endpoint `json:"internalEndpoint"`

	// External endpoints of all Kubernetes services that have the same label selector as
	// connected Replication Controller.
	// Endpoint is external IP address name merged with ports.
	ExternalEndpoints []Endpoint `json:"externalEndpoints"`

	// Label selector of the service.
	Selector map[string]string `json:"selector"`
}

// ServicePort is a pair of port and protocol, e.g. a service endpoint.
type ServicePort struct {
	// Positive port number.
	Port int32 `json:"port"`

	// Protocol name, e.g., TCP or UDP.
	Protocol api.Protocol `json:"protocol"`
}

// Endpoint describes an endpoint that is host and a list of available ports for that host.
type Endpoint struct {
	// Hostname, either as a domain name or IP address.
	Host string `json:"host"`

	// List of ports opened for this endpoint on the hostname.
	Ports []ServicePort `json:"ports"`
}

// ReplicationControllerSpec contains information needed to update replication controller.
type ReplicationControllerSpec struct {
	// Replicas (pods) number in replicas set
	Replicas int32 `json:"replicas"`
}

// GetReplicationControllerDetail returns detailed information about the given replication
// controller in the given namespace.
func GetReplicationControllerDetail(client client.Interface, heapsterClient HeapsterClient,
	namespace, name string) (*ReplicationControllerDetail, error) {
	log.Printf("Getting details of %s replication controller in %s namespace", name, namespace)

	replicationControllerWithPods, err := getRawReplicationControllerWithPods(client, namespace, name)
	if err != nil {
		return nil, err
	}
	replicationController := replicationControllerWithPods.ReplicationController
	pods := replicationControllerWithPods.Pods

	replicationControllerMetricsByPod, err := getReplicationControllerPodsMetrics(pods, heapsterClient, namespace, name)
	if err != nil {
		log.Printf("Skipping Heapster metrics because of error: %s\n", err)
	}

	services, err := client.Services(namespace).List(api.ListOptions{
		LabelSelector: labels.Everything(),
		FieldSelector: fields.Everything(),
	})

	if err != nil {
		return nil, err
	}

	replicationControllerDetail := &ReplicationControllerDetail{
		Name:          replicationController.Name,
		Namespace:     replicationController.Namespace,
		Labels:        replicationController.ObjectMeta.Labels,
		LabelSelector: replicationController.Spec.Selector,
		PodInfo:       getReplicationControllerPodInfo(replicationController, pods.Items),
	}

	matchingServices := getMatchingServices(services.Items, replicationController)

	// Anonymous callback function to get nodes by their names.
	getNodeFn := func(nodeName string) (*api.Node, error) {
		return client.Nodes().Get(nodeName)
	}

	for _, service := range matchingServices {
		replicationControllerDetail.Services = append(replicationControllerDetail.Services,
			getServiceDetail(service, *replicationController, pods.Items, getNodeFn))
	}

	for _, container := range replicationController.Spec.Template.Spec.Containers {
		replicationControllerDetail.ContainerImages = append(replicationControllerDetail.ContainerImages,
			container.Image)
	}

	for _, pod := range pods.Items {
		podDetail := ReplicationControllerPod{
			Name:         pod.Name,
			PodPhase:     pod.Status.Phase,
			StartTime:    pod.CreationTimestamp,
			PodIP:        pod.Status.PodIP,
			NodeName:     pod.Spec.NodeName,
			RestartCount: getRestartCount(pod),
		}
		if replicationControllerMetricsByPod != nil {
			metric := replicationControllerMetricsByPod.MetricsMap[pod.Name]
			podDetail.Metrics = &metric
			replicationControllerDetail.HasMetrics = true
		}
		replicationControllerDetail.Pods = append(replicationControllerDetail.Pods, podDetail)
	}

	return replicationControllerDetail, nil
}

// TODO(floreks): This should be transactional to make sure that RC will not be deleted without pods
// DeleteReplicationController deletes replication controller with given name in given namespace and
// related pods. Also deletes services related to replication controller if deleteServices is true.
func DeleteReplicationController(client client.Interface, namespace, name string,
	deleteServices bool) error {

	log.Printf("Deleting %s replication controller from %s namespace", name, namespace)

	if deleteServices {
		if err := DeleteReplicationControllerServices(client, namespace, name); err != nil {
			return err
		}
	}

	pods, err := getRawReplicationControllerPods(client, namespace, name)
	if err != nil {
		return err
	}

	if err := client.ReplicationControllers(namespace).Delete(name); err != nil {
		return err
	}

	for _, pod := range pods.Items {
		if err := client.Pods(namespace).Delete(pod.Name, &api.DeleteOptions{}); err != nil {
			return err
		}
	}

	log.Printf("Successfully deleted %s replication controller from %s namespace", name, namespace)

	return nil
}

// DeleteReplicationControllerServices deletes services related to replication controller with given
// name in given namespace.
func DeleteReplicationControllerServices(client client.Interface, namespace, name string) error {
	log.Printf("Deleting services related to %s replication controller from %s namespace", name,
		namespace)

	replicationController, err := client.ReplicationControllers(namespace).Get(name)
	if err != nil {
		return err
	}

	labelSelector, err := toLabelSelector(replicationController.Spec.Selector)
	if err != nil {
		return err
	}

	services, err := getServicesForDeletion(client, labelSelector, namespace)
	if err != nil {
		return err
	}

	for _, service := range services {
		if err := client.Services(namespace).Delete(service.Name); err != nil {
			return err
		}
	}

	log.Printf("Successfully deleted services related to %s replication controller from %s namespace",
		name, namespace)

	return nil
}

// UpdateReplicasCount updates number of replicas in Replication Controller based on Replication
// Controller Spec
func UpdateReplicasCount(client client.Interface, namespace, name string,
	replicationControllerSpec *ReplicationControllerSpec) error {
	log.Printf("Updating replicas count to %d for %s replication controller from %s namespace",
		replicationControllerSpec.Replicas, name, namespace)

	replicationController, err := client.ReplicationControllers(namespace).Get(name)
	if err != nil {
		return err
	}

	replicationController.Spec.Replicas = replicationControllerSpec.Replicas

	_, err = client.ReplicationControllers(namespace).Update(replicationController)
	if err != nil {
		return err
	}

	log.Printf("Successfully updated replicas count to %d for %s replication controller from %s namespace",
		replicationControllerSpec.Replicas, name, namespace)

	return nil
}

// Returns detailed information about service from given service
func getServiceDetail(service api.Service, replicationController api.ReplicationController,
	pods []api.Pod, getNodeFn GetNodeFunc) ServiceDetail {
	return ServiceDetail{
		Name: service.ObjectMeta.Name,
		InternalEndpoint: getInternalEndpoint(service.Name, service.Namespace,
			service.Spec.Ports),
		ExternalEndpoints: getExternalEndpoints(replicationController, pods, service, getNodeFn),
		Selector:          service.Spec.Selector,
	}
}

// Gets restart count of given pod (total number of its containers restarts).
func getRestartCount(pod api.Pod) int {
	restartCount := 0
	for _, containerStatus := range pod.Status.ContainerStatuses {
		restartCount += int(containerStatus.RestartCount)
	}
	return restartCount
}

// Returns internal endpoint name for the given service properties, e.g.,
// "my-service.namespace 80/TCP" or "my-service 53/TCP,53/UDP".
func getInternalEndpoint(serviceName, namespace string, ports []api.ServicePort) Endpoint {

	name := serviceName
	if namespace != api.NamespaceDefault {
		bufferName := bytes.NewBufferString(name)
		bufferName.WriteString(".")
		bufferName.WriteString(namespace)
		name = bufferName.String()
	}

	return Endpoint{
		Host:  name,
		Ports: getServicePorts(ports),
	}
}

// Returns array of external endpoints for a replication controller.
func getExternalEndpoints(replicationController api.ReplicationController, pods []api.Pod,
	service api.Service, getNodeFn GetNodeFunc) []Endpoint {
	var externalEndpoints []Endpoint
	replicationControllerPods := filterReplicationControllerPods(replicationController, pods)

	if service.Spec.Type == api.ServiceTypeNodePort {
		externalEndpoints = getNodePortEndpoints(replicationControllerPods, service, getNodeFn)
	} else if service.Spec.Type == api.ServiceTypeLoadBalancer {
		for _, ingress := range service.Status.LoadBalancer.Ingress {
			externalEndpoints = append(externalEndpoints, getExternalEndpoint(ingress,
				service.Spec.Ports))
		}

		if len(externalEndpoints) == 0 {
			externalEndpoints = getNodePortEndpoints(replicationControllerPods, service, getNodeFn)
		}
	}

	if len(externalEndpoints) == 0 && (service.Spec.Type == api.ServiceTypeNodePort ||
		service.Spec.Type == api.ServiceTypeLoadBalancer) {
		externalEndpoints = getLocalhostEndpoints(service)
	}

	return externalEndpoints
}

// Returns localhost endpoints for specified node port or load balancer service.
func getLocalhostEndpoints(service api.Service) []Endpoint {
	var externalEndpoints []Endpoint
	for _, port := range service.Spec.Ports {
		externalEndpoints = append(externalEndpoints, Endpoint{
			Host: "localhost",
			Ports: []ServicePort{
				{
					Protocol: port.Protocol,
					Port:     port.NodePort,
				},
			},
		})
	}
	return externalEndpoints
}

// Returns pods that belong to specified replication controller.
func filterReplicationControllerPods(replicationController api.ReplicationController,
	allPods []api.Pod) []api.Pod {
	var pods []api.Pod
	for _, pod := range allPods {
		if isLabelSelectorMatching(replicationController.Spec.Selector, pod.Labels) {
			pods = append(pods, pod)
		}
	}
	return pods
}

// Returns array of external endpoints for specified pods.
func getNodePortEndpoints(pods []api.Pod, service api.Service, getNodeFn GetNodeFunc) []Endpoint {
	var externalEndpoints []Endpoint
	var externalIPs []string
	for _, pod := range pods {
		node, err := getNodeFn(pod.Spec.NodeName)
		if err != nil {
			continue
		}
		for _, adress := range node.Status.Addresses {
			if adress.Type == api.NodeExternalIP && len(adress.Address) > 0 &&
				isExternalIPUniqe(externalIPs, adress.Address) {
				externalIPs = append(externalIPs, adress.Address)
				for _, port := range service.Spec.Ports {
					externalEndpoints = append(externalEndpoints, Endpoint{
						Host: adress.Address,
						Ports: []ServicePort{
							{
								Protocol: port.Protocol,
								Port:     port.NodePort,
							},
						},
					})
				}
			}
		}
	}
	return externalEndpoints
}

// Returns true if given external IP is not part of given array.
func isExternalIPUniqe(externalIPs []string, externalIP string) bool {
	for _, h := range externalIPs {
		if h == externalIP {
			return false
		}
	}
	return true
}

// Returns external endpoint name for the given service properties.
func getExternalEndpoint(ingress api.LoadBalancerIngress, ports []api.ServicePort) Endpoint {
	var host string
	if ingress.Hostname != "" {
		host = ingress.Hostname
	} else {
		host = ingress.IP
	}
	return Endpoint{
		Host:  host,
		Ports: getServicePorts(ports),
	}
}

// Gets human readable name for the given service ports list.
func getServicePorts(apiPorts []api.ServicePort) []ServicePort {
	var ports []ServicePort
	for _, port := range apiPorts {
		ports = append(ports, ServicePort{port.Port, port.Protocol})
	}
	return ports
}
