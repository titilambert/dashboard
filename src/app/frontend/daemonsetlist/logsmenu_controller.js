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

import {StateParams} from 'logs/logs_state';
import {stateName as logsStateName} from 'logs/logs_state';

/**
 * Controller for the logs menu view.
 *
 * @final
 */
export default class LogsMenuController {
  /**
   * @param {!ui.router.$state} $state
   * @param {!angular.$log} $log
   * @param {!angular.$resource} $resource
   * @ngInject
   */
  constructor($state, $log, $resource) {
    /** @private {!ui.router.$state} */
    this.state_ = $state;

    /** @private {!angular.$resource} */
    this.resource_ = $resource;

    /** @private {!angular.$log} */
    this.log_ = $log;

    /**
     * This is initialized from the scope.
     * @export {string}
     */
    this.daemonSetName;

    /**
     * This is initialized from the scope.
     * @export {string}
     */
    this.namespace;

    /**
     * This is initialized on open menu.
     * @export {!Array<!backendApi.DaemonSetPodWithContainers>}
     */
    this.daemonSetPodsList;
  }

  /**
   * Opens menu with pods and link to logs.
   * @param  {!function(!MouseEvent)} $mdOpenMenu
   * @param  {!MouseEvent} $event
   * @export
   */
  openMenu($mdOpenMenu, $event) {
    // This is needed to resolve problem with data refresh.
    // Sometimes old data was included to the new one for a while.
    if (this.daemonSetPodsList) {
      this.daemonSetPodsList = [];
    }
    this.getDaemonSetPods_();
    $mdOpenMenu($event);
  }

  /**
   * @private
   */
  getDaemonSetPods_() {
    /** @type {!angular.Resource<!backendApi.DaemonSetPods>} */
    let resource = this.resource_(
        `api/v1/daemonsets/pods/${this.namespace}/` +
        `${this.daemonSetName}?limit=10`);

    resource.get(
        (daemonSetPods) => {
          this.log_.info(
              'Successfully fetched DaemonSet pods: ', daemonSetPods);
          this.daemonSetPodsList = daemonSetPods.pods;
        },
        (err) => { this.log_.error('Error fetching DaemonSet pods: ', err); });
  }

  /**
   * @param {string} podName
   * @return {string}
   * @export
   */
  getLogsHref(podName) {
    return this.state_.href(
        logsStateName, new StateParams(this.namespace, this.daemonSetName, podName));
  }

  /**
   * Checks if pod contains at least one container. Return true if yes, otherwise false.
   * @param {!backendApi.DaemonSetPodWithContainers} pod
   * @return {boolean}
   * @export
   */
  podContainerExists(pod) {
    if (pod.podContainers[0].name === undefined) {
      return false;
    }
    return true;
  }

  /**
   * Checks if pod containers were restarted. Return true if yes, otherwise false.
   * @param {backendApi.DaemonSetPodWithContainers} pod
   * @return {boolean}
   * @export
   */
  podContainersRestarted(pod) {
    if (pod) {
      return pod.totalRestartCount > 0;
    }
    return false;
  }
}
