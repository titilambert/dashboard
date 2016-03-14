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

import DaemonSetDetailController from './daemonsetdetail_controller';
import {stateName} from './daemonsetdetail_state';

/**
 * Configures states for the service view.
 *
 * @param {!ui.router.$stateProvider} $stateProvider
 * @ngInject
 */
export default function stateConfig($stateProvider) {
  $stateProvider.state(stateName, {
    controller: DaemonSetDetailController,
    controllerAs: 'ctrl',
    url: '/daemonsets/:namespace/:daemonSet',
    templateUrl: 'daemonsetdetail/daemonsetdetail.html',
    resolve: {
      'daemonSetSpecPodsResource': getDaemonSetSpecPodsResource,
      'daemonSetDetailResource': getDaemonSetDetailsResource,
      'daemonSetDetail': resolveDaemonSetDetails,
      'daemonSetEvents': resolveDaemonSetEvents,
    },
  });
}

/**
 * @param {!./daemonsetdetail_state.StateParams} $stateParams
 * @param {!angular.$resource} $resource
 * @return {!angular.Resource<!backendApi.DaemonSetDetail>}
 * @ngInject
 */
export function getDaemonSetDetailsResource($stateParams, $resource) {
  return $resource(
      `api/v1/daemonsets/${$stateParams.namespace}/` +
      `${$stateParams.daemonSet}`);
}

/**
 * @param {!./daemonsetdetail_state.StateParams} $stateParams
 * @param {!angular.$resource} $resource
 * @return {!angular.Resource<!backendApi.DaemonSetSpec>}
 * @ngInject
 */
export function getDaemonSetSpecPodsResource($stateParams, $resource) {
  return $resource(
      `api/v1/daemonsets/${$stateParams.namespace}/` +
      `${$stateParams.daemonSet}/update/pods`);
}

/**
 * @param {!angular.Resource<!backendApi.DaemonSetDetail>}
 * daemonSetDetailResource
 * @return {!angular.$q.Promise}
 * @ngInject
 */
function resolveDaemonSetDetails(daemonSetDetailResource) {
  return daemonSetDetailResource.get().$promise;
}

/**
 * @param {!./daemonsetdetail_state.StateParams} $stateParams
 * @param {!angular.$resource} $resource
 * @return {!angular.$q.Promise}
 * @ngInject
 */
function resolveDaemonSetEvents($stateParams, $resource) {
  /** @type {!angular.Resource<!backendApi.Events>} */
  let resource =
      $resource(`api/v1/events/${$stateParams.namespace}/${$stateParams.daemonSet}`);

  return resource.get().$promise;
}
