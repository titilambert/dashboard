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
	"errors"
	"fmt"
	"strings"
	"time"

	"k8s.io/kubernetes/pkg/api"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/kubectl"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	kubectlResource "k8s.io/kubernetes/pkg/kubectl/resource"
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

	// Interval
	PollInterval int64 `json:"poll-interval"`

	// Period
	Timeout int64 `json:"timeout"`

	// Deletion timeout
	DeletionTimeout int64 `json:"deletion-timeout"`

	// Creation timeout
	CreationTimeout int64 `json:"creation-timeout"`
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

	// Get old RC
	oldRc, err := client.ReplicationControllers(namespace).Get(oldName)
	if err != nil {
		return err
	}

	var interval time.Duration
	var timeout time.Duration
	var deletionTimeout time.Duration
	var creationTimeout time.Duration
	if newRCSpec.PollInterval == 0 {
		interval = time.Duration(3) * time.Second // default 3s
	} else {
		interval = time.Duration(newRCSpec.PollInterval) * time.Second
	}
	if newRCSpec.Timeout == 0 {
		timeout = time.Duration(300) * time.Second // default 5m0s
	} else {
		timeout = time.Duration(newRCSpec.Timeout) * time.Second
	}
	if newRCSpec.DeletionTimeout == 0 {
		deletionTimeout = time.Duration(600) * time.Second // default 10m0s
	} else {
		deletionTimeout = time.Duration(newRCSpec.DeletionTimeout) * time.Second
	}
	if newRCSpec.CreationTimeout == 0 {
		creationTimeout = time.Duration(900) * time.Second // default 15m0s
	} else {
		creationTimeout = time.Duration(newRCSpec.CreationTimeout) * time.Second
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

	// Prepare rolling updater config
	out := &bytes.Buffer{}
	updateCleanupPolicy := kubectl.DeleteRollingUpdateByNodeCleanupPolicy
	/*
	   // TODO enable this feature
	   if keepOldName {
	       updateCleanupPolicy = kubectl.RenameRollingUpdateCleanupPolicy
	   }
	*/

	// Set same namespace
	newRc.Namespace = oldRc.Namespace
	// Init rolling updater config
	config := &kubectl.RollingUpdaterByNodeConfig{
		Out:       out,
		OldRc:     oldRc,
		NewRc:     newRc,
		NodeLabel: rollingUpdateLabel,
		//        UpdatePeriod:   period,
		Interval:        interval,
		Timeout:         timeout,
		DeletionTimeout: deletionTimeout,
		CreationTimeout: creationTimeout,
		CleanupPolicy:   updateCleanupPolicy,
	}
	// Create rolling updater
	updater := kubectl.NewRollingUpdaterByNode(namespace, client)
	// Rolling udpate
	err = updater.Update(config)
	if err != nil {
		return err
	}

	return nil
}
